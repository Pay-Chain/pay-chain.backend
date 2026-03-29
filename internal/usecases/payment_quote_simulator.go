package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/infrastructure/blockchain"
)

const (
	defaultSimulatorQuoteTimeout = 8 * time.Second
)

var commonUniswapV3FeeTiers = []uint32{100, 500, 3000, 10000}

type simulatorQuoteRequest struct {
	ChainID        string `json:"chain_id"`
	ChainUUID      string `json:"chain_uuid"`
	RPCURL         string `json:"rpc_url,omitempty"`
	SwapperAddress string `json:"swapper_address,omitempty"`
	TokenIn        string `json:"token_in"`
	TokenOut       string `json:"token_out"`
	AmountIn       string `json:"amount_in"`
}

func (u *PaymentUsecase) getSimulatorBackedPartnerQuote(
	ctx context.Context,
	chainID uuid.UUID,
	tokenIn, tokenOut string,
	amountIn *big.Int,
) (*AccurateSwapQuoteResult, error) {
	createPaymentTraceDebug(ctx, "payment_quote_simulator.start",
		zap.String("chain_uuid", chainID.String()),
		zap.String("token_in", normalizeEvmAddress(tokenIn)),
		zap.String("token_out", normalizeEvmAddress(tokenOut)),
		zap.String("amount_in_atomic", safeBigIntString(amountIn)),
	)
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, fmt.Errorf("amountIn must be positive")
	}
	if strings.EqualFold(tokenIn, tokenOut) {
		return &AccurateSwapQuoteResult{
			AmountOut:   new(big.Int).Set(amountIn),
			PriceSource: "identity",
		}, nil
	}

	// Primary free fallback: use RPC dry-run quote path (no external simulator service required).
	var rpcDryRunErr error
	if rpcQuote, rpcErr := u.getRPCDryRunPartnerQuote(ctx, chainID, tokenIn, tokenOut, amountIn); rpcErr == nil && rpcQuote != nil {
		createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_success",
			zap.String("chain_uuid", chainID.String()),
			zap.String("token_in", normalizeEvmAddress(tokenIn)),
			zap.String("token_out", normalizeEvmAddress(tokenOut)),
			zap.String("amount_in_atomic", amountIn.String()),
			zap.String("amount_out_atomic", safeBigIntString(rpcQuote.AmountOut)),
			zap.String("price_source", strings.TrimSpace(rpcQuote.PriceSource)),
			zap.String("route_summary", strings.TrimSpace(rpcQuote.RouteSummary)),
		)
		return rpcQuote, nil
	} else if rpcErr != nil {
		rpcDryRunErr = rpcErr
		createPaymentTraceWarn(ctx, "payment_quote_simulator.rpc_dry_run_failed",
			zap.String("chain_uuid", chainID.String()),
			zap.String("token_in", normalizeEvmAddress(tokenIn)),
			zap.String("token_out", normalizeEvmAddress(tokenOut)),
			zap.String("amount_in_atomic", amountIn.String()),
			zap.Error(rpcErr),
		)
	}

	// Optional secondary fallback: external simulator service (if configured).
	endpoint := strings.TrimSpace(os.Getenv("PAYMENT_SIMULATOR_QUOTE_URL"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("SIMULATOR_QUOTE_URL"))
	}
	if endpoint == "" {
		if rpcDryRunErr != nil {
			return nil, fmt.Errorf("rpc dry-run quote unavailable: %v; PAYMENT_SIMULATOR_QUOTE_URL is not configured", rpcDryRunErr)
		}
		return nil, fmt.Errorf("rpc dry-run quote unavailable and PAYMENT_SIMULATOR_QUOTE_URL is not configured")
	}

	chain, err := u.chainRepo.GetByID(ctx, chainID)
	if err != nil {
		return nil, err
	}

	swapperAddress := ""
	if contract, contractErr := u.contractRepo.GetActiveContract(ctx, chain.ID, entities.ContractTypeTokenSwapper); contractErr == nil && contract != nil {
		swapperAddress = contract.ContractAddress
	}
	createPaymentTraceDebug(ctx, "payment_quote_simulator.external_request_start",
		zap.String("chain_caip2", chain.GetCAIP2ID()),
		zap.String("chain_uuid", chain.ID.String()),
		zap.String("token_in", normalizeEvmAddress(tokenIn)),
		zap.String("token_out", normalizeEvmAddress(tokenOut)),
		zap.String("amount_in_atomic", amountIn.String()),
		zap.String("rpc_url", strings.TrimSpace(chain.RPCURL)),
		zap.String("swapper_address", strings.TrimSpace(swapperAddress)),
		zap.String("simulator_endpoint", endpoint),
	)

	requestPayload := simulatorQuoteRequest{
		ChainID:        chain.GetCAIP2ID(),
		ChainUUID:      chain.ID.String(),
		RPCURL:         strings.TrimSpace(chain.RPCURL),
		SwapperAddress: strings.TrimSpace(swapperAddress),
		TokenIn:        tokenIn,
		TokenOut:       tokenOut,
		AmountIn:       amountIn.String(),
	}
	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, err
	}

	timeout := defaultSimulatorQuoteTimeout
	if rawTimeout := strings.TrimSpace(os.Getenv("PAYMENT_SIMULATOR_TIMEOUT_MS")); rawTimeout != "" {
		if timeoutMs, parseErr := strconv.Atoi(rawTimeout); parseErr == nil && timeoutMs > 0 {
			timeout = time.Duration(timeoutMs) * time.Millisecond
		}
	}
	client := &http.Client{Timeout: timeout}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(os.Getenv("PAYMENT_SIMULATOR_QUOTE_API_KEY")); apiKey != "" {
		httpReq.Header.Set("X-Api-Key", apiKey)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("simulator quote request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		createPaymentTraceWarn(ctx, "payment_quote_simulator.external_request_failed",
			zap.String("simulator_endpoint", endpoint),
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, fmt.Errorf("simulator quote failed with status %d", resp.StatusCode)
	}

	amountOut, priceSource, routeSummary, err := parseSimulatorQuoteResponse(respBody)
	if err != nil {
		return nil, err
	}

	if amountOut == nil || amountOut.Sign() <= 0 {
		return nil, fmt.Errorf("invalid simulator quote amount")
	}
	if amountOut.Cmp(amountIn) == 0 && !strings.EqualFold(tokenIn, tokenOut) {
		return nil, fmt.Errorf("simulator quote returned 1:1 placeholder amount")
	}
	createPaymentTraceDebug(ctx, "payment_quote_simulator.external_request_success",
		zap.String("chain_caip2", chain.GetCAIP2ID()),
		zap.String("token_in", normalizeEvmAddress(tokenIn)),
		zap.String("token_out", normalizeEvmAddress(tokenOut)),
		zap.String("amount_in_atomic", amountIn.String()),
		zap.String("amount_out_atomic", amountOut.String()),
		zap.String("price_source", strings.TrimSpace(priceSource)),
		zap.String("route_summary", strings.TrimSpace(routeSummary)),
	)

	return &AccurateSwapQuoteResult{
		AmountOut:    amountOut,
		PriceSource:  priceSource,
		RouteSummary: routeSummary,
	}, nil
}

func (u *PaymentUsecase) getRPCDryRunPartnerQuote(
	ctx context.Context,
	chainID uuid.UUID,
	tokenIn, tokenOut string,
	amountIn *big.Int,
) (*AccurateSwapQuoteResult, error) {
	ctx = withQuoteRequestCache(ctx)
	createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_start",
		zap.String("chain_uuid", chainID.String()),
		zap.String("token_in", normalizeEvmAddress(tokenIn)),
		zap.String("token_out", normalizeEvmAddress(tokenOut)),
		zap.String("amount_in_atomic", safeBigIntString(amountIn)),
	)
	chain, err := u.chainRepo.GetByID(ctx, chainID)
	if err != nil {
		return nil, err
	}
	swapper, err := u.getCachedActiveContract(ctx, chain.ID, entities.ContractTypeTokenSwapper)
	if err != nil || swapper == nil {
		return nil, fmt.Errorf("active swapper not found")
	}
	client, err := u.clientFactory.GetEVMClient(chain.RPCURL)
	if err != nil {
		return nil, err
	}
	createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_contracts",
		zap.String("chain_caip2", chain.GetCAIP2ID()),
		zap.String("chain_name", chain.Name),
		zap.String("swapper_address", strings.TrimSpace(swapper.ContractAddress)),
	)
	swapperABI, err := u.getCachedResolvedABI(ctx, chain.ID, entities.ContractTypeTokenSwapper)
	if err != nil {
		return nil, err
	}

	routePath, err := u.getCachedRoutePath(ctx, client, chain.ID, swapper.ContractAddress, swapperABI, tokenIn, tokenOut)
	if err != nil {
		return nil, err
	}
	if len(routePath) < 2 {
		return nil, fmt.Errorf("rpc dry-run quote unavailable because route path is empty")
	}
	createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_route",
		zap.String("chain_caip2", chain.GetCAIP2ID()),
		zap.Strings("route_path", routePath),
	)

	quoterV3, err := u.getCachedQuoterV3(ctx, client, chain.ID, chain, swapper.ContractAddress)
	if err != nil {
		return nil, err
	}
	createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_quoter",
		zap.String("chain_caip2", chain.GetCAIP2ID()),
		zap.String("quoter_v3", quoterV3.Hex()),
	)

	currentAmount := new(big.Int).Set(amountIn)
	for i := 0; i < len(routePath)-1; i++ {
		hopIn := normalizeEvmAddress(routePath[i])
		hopOut := normalizeEvmAddress(routePath[i+1])

		active, feeTier, _ := u.getCachedV3PoolConfig(ctx, client, chain.ID, swapper.ContractAddress, hopIn, hopOut)
		createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_hop_start",
			zap.String("chain_caip2", chain.GetCAIP2ID()),
			zap.Int("hop_index", i),
			zap.String("hop_in", hopIn),
			zap.String("hop_out", hopOut),
			zap.Bool("configured_active", active),
			zap.Uint32("configured_fee_tier", feeTier),
			zap.String("amount_in_atomic", currentAmount.String()),
		)
		quotedHopAmount, quoteErr := quoteHopV3WithDryRunFees(ctx, client, quoterV3.Hex(), hopIn, hopOut, currentAmount, active, feeTier)
		if quoteErr != nil {
			return nil, fmt.Errorf("rpc dry-run quote unavailable for hop %s -> %s: %w", hopIn, hopOut, quoteErr)
		}
		if quotedHopAmount == nil || quotedHopAmount.Sign() <= 0 {
			return nil, fmt.Errorf("rpc dry-run quote returned invalid hop amount")
		}
		createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_hop_success",
			zap.String("chain_caip2", chain.GetCAIP2ID()),
			zap.Int("hop_index", i),
			zap.String("hop_in", hopIn),
			zap.String("hop_out", hopOut),
			zap.String("amount_in_atomic", currentAmount.String()),
			zap.String("amount_out_atomic", quotedHopAmount.String()),
		)
		currentAmount = quotedHopAmount
	}

	priceSource := fmt.Sprintf("rpc-dry-run-uniswap-v3-%s", strings.ToLower(chain.Name))
	if len(routePath) > 2 {
		priceSource += "-multihop"
	}
	return &AccurateSwapQuoteResult{
		AmountOut:    currentAmount,
		PriceSource:  priceSource,
		RouteSummary: strings.Join(routePath, "->"),
	}, nil
}

func quoteHopV3WithDryRunFees(
	ctx context.Context,
	client *blockchain.EVMClient,
	quoterAddress string,
	tokenIn, tokenOut string,
	amountIn *big.Int,
	configuredActive bool,
	configuredFee uint32,
) (*big.Int, error) {
	errorByFee := make(map[uint32]string)
	// Respect on-chain configured fee tier first to avoid accidentally selecting
	// a different pool tier that yields unstable/unintended quotes.
	if configuredActive && configuredFee > 0 {
		createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_fee_probe",
			zap.String("token_in", tokenIn),
			zap.String("token_out", tokenOut),
			zap.Uint32("fee_tier", configuredFee),
			zap.String("amount_in_atomic", safeBigIntString(amountIn)),
			zap.Bool("configured_fee", true),
		)
		amountOut, err := callQuoterV3ExactInputSingle(ctx, client, quoterAddress, tokenIn, tokenOut, amountIn, configuredFee)
		if err == nil && amountOut != nil && amountOut.Sign() > 0 {
			createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_fee_probe_success",
				zap.String("token_in", tokenIn),
				zap.String("token_out", tokenOut),
				zap.Uint32("fee_tier", configuredFee),
				zap.String("amount_out_atomic", amountOut.String()),
			)
			return amountOut, nil
		}
		if err != nil {
			errorByFee[configuredFee] = err.Error()
		} else {
			errorByFee[configuredFee] = "amountOut <= 0"
		}
	}

	// Fallback probe only when configured tier is missing/failing.
	for _, fee := range commonUniswapV3FeeTiers {
		if configuredFee > 0 && fee == configuredFee {
			continue
		}
		createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_fee_probe",
			zap.String("token_in", tokenIn),
			zap.String("token_out", tokenOut),
			zap.Uint32("fee_tier", fee),
			zap.String("amount_in_atomic", safeBigIntString(amountIn)),
			zap.Bool("configured_fee", false),
		)
		amountOut, err := callQuoterV3ExactInputSingle(ctx, client, quoterAddress, tokenIn, tokenOut, amountIn, fee)
		if err == nil && amountOut != nil && amountOut.Sign() > 0 {
			createPaymentTraceDebug(ctx, "payment_quote_simulator.rpc_dry_run_fee_probe_success",
				zap.String("token_in", tokenIn),
				zap.String("token_out", tokenOut),
				zap.Uint32("fee_tier", fee),
				zap.String("amount_out_atomic", amountOut.String()),
			)
			return amountOut, nil
		}
		if err != nil {
			errorByFee[fee] = err.Error()
		} else {
			errorByFee[fee] = "amountOut <= 0"
		}
	}

	fees := make([]int, 0, len(errorByFee))
	for fee := range errorByFee {
		fees = append(fees, int(fee))
	}
	sort.Ints(fees)
	reasons := make([]string, 0, len(fees))
	for _, fee := range fees {
		reasons = append(reasons, fmt.Sprintf("%d:%s", fee, errorByFee[uint32(fee)]))
	}
	return nil, fmt.Errorf("no usable v3 fee tier found (amountIn=%s, reasons=%s)", amountIn.String(), strings.Join(reasons, "; "))
}

func parseSimulatorQuoteResponse(body []byte) (*big.Int, string, string, error) {
	if len(body) == 0 {
		return nil, "", "", fmt.Errorf("empty simulator quote response")
	}

	var raw map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, "", "", fmt.Errorf("invalid simulator quote response")
	}

	amountOut := extractBigIntFromAny(raw,
		"amount_out",
		"amountOut",
		"quoted_amount",
		"quotedAmount",
		"output_amount",
		"outputAmount",
	)
	if amountOut == nil {
		if nested, ok := raw["data"].(map[string]interface{}); ok {
			amountOut = extractBigIntFromAny(nested,
				"amount_out",
				"amountOut",
				"quoted_amount",
				"quotedAmount",
				"output_amount",
				"outputAmount",
			)
		}
	}
	if amountOut == nil {
		return nil, "", "", fmt.Errorf("invalid simulator quote response: amount_out not found")
	}

	priceSource := strings.TrimSpace(extractStringFromAny(raw, "price_source", "priceSource", "source"))
	routeSummary := strings.TrimSpace(extractStringFromAny(raw, "route", "route_summary", "routeSummary"))
	if priceSource == "" {
		priceSource = "simulator-fallback"
	}

	return amountOut, priceSource, routeSummary, nil
}

func extractBigIntFromAny(payload map[string]interface{}, keys ...string) *big.Int {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}

		switch typed := value.(type) {
		case string:
			candidate := strings.TrimSpace(typed)
			if candidate == "" {
				continue
			}
			if out, ok := new(big.Int).SetString(candidate, 10); ok && out.Sign() >= 0 {
				return out
			}
		case json.Number:
			if out, ok := new(big.Int).SetString(typed.String(), 10); ok && out.Sign() >= 0 {
				return out
			}
		}
	}
	return nil
}

func extractStringFromAny(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return ""
}

func safeBigIntString(v *big.Int) string {
	if v == nil {
		return ""
	}
	return v.String()
}
