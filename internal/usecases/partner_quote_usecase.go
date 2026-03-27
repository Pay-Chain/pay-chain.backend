package usecases

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	domainentities "payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/pkg/utils"
)

const (
	partnerQuoteTTL      = 5 * time.Minute
	partnerQuoteSlippage = 100
)

type CreatePartnerQuoteInput struct {
	MerchantID      uuid.UUID
	InvoiceCurrency string
	InvoiceToken    string
	InvoiceAmount   string
	SelectedChain   string
	SelectedToken   string
	DestWallet      string
}

type CreatePartnerQuoteOutput struct {
	QuoteID             string    `json:"quote_id"`
	InvoiceCurrency     string    `json:"invoice_currency"`
	InvoiceAmount       string    `json:"invoice_amount"`
	SelectedChain       string    `json:"selected_chain"`
	SelectedToken       string    `json:"selected_token"`
	SelectedTokenSymbol string    `json:"selected_token_symbol"`
	QuotedAmount        string    `json:"quoted_amount"`
	QuoteDecimals       int       `json:"quote_decimals"`
	QuoteRate           string    `json:"quote_rate"`
	PriceSource         string    `json:"price_source"`
	Route               string    `json:"route"`
	SlippageBps         int       `json:"slippage_bps"`
	RateTimestamp       time.Time `json:"rate_timestamp"`
	QuoteExpiresAt      time.Time `json:"quote_expires_at"`
}

type PreviewRequiredInputForOutputInput struct {
	MerchantID         uuid.UUID
	SelectedChain      string
	InputToken         string
	OutputToken        string
	TargetOutputAmount string
}

type PreviewRequiredInputForOutputOutput struct {
	RequiredInputAmount string `json:"required_input_amount"`
	PriceSource         string `json:"price_source"`
	Route               string `json:"route"`
}

type PartnerQuoteUsecase struct {
	quoteRepo               repositories.PaymentQuoteRepository
	tokenRepo               repositories.TokenRepository
	chainRepo               repositories.ChainRepository
	chainResolver           *ChainResolver
	routeSupportFn          func(context.Context, uuid.UUID, string, string) (*TokenRouteSupportStatus, error)
	swapQuoteFn             func(context.Context, uuid.UUID, string, string, *big.Int) (*big.Int, error)
	accurateQuoteFn         func(context.Context, uuid.UUID, string, string, *big.Int) (*AccurateSwapQuoteResult, error)
	accurateRequiredInputFn func(context.Context, uuid.UUID, string, string, *big.Int) (*AccurateSwapRequiredInputResult, error)
	simulatorQuoteFn        func(context.Context, uuid.UUID, string, string, *big.Int) (*AccurateSwapQuoteResult, error)
}

func NewPartnerQuoteUsecase(
	quoteRepo repositories.PaymentQuoteRepository,
	tokenRepo repositories.TokenRepository,
	chainRepo repositories.ChainRepository,
	paymentUsecase *PaymentUsecase,
) *PartnerQuoteUsecase {
	uc := &PartnerQuoteUsecase{
		quoteRepo:     quoteRepo,
		tokenRepo:     tokenRepo,
		chainRepo:     chainRepo,
		chainResolver: NewChainResolver(chainRepo),
	}
	if paymentUsecase != nil {
		uc.routeSupportFn = paymentUsecase.CheckRouteSupportDetailed
		uc.swapQuoteFn = paymentUsecase.getSwapQuote
		uc.accurateQuoteFn = paymentUsecase.getAccuratePartnerQuote
		uc.accurateRequiredInputFn = paymentUsecase.getAccuratePartnerRequiredInput
		uc.simulatorQuoteFn = paymentUsecase.getSimulatorBackedPartnerQuote
	}
	return uc
}

func (u *PartnerQuoteUsecase) RouteSupportFnForTest(fn func(context.Context, uuid.UUID, string, string) (*TokenRouteSupportStatus, error)) {
	u.routeSupportFn = fn
}

func (u *PartnerQuoteUsecase) SwapQuoteFnForTest(fn func(context.Context, uuid.UUID, string, string, *big.Int) (*big.Int, error)) {
	u.swapQuoteFn = fn
}

func (u *PartnerQuoteUsecase) SimulatorQuoteFnForTest(fn func(context.Context, uuid.UUID, string, string, *big.Int) (*AccurateSwapQuoteResult, error)) {
	u.simulatorQuoteFn = fn
}

func (u *PartnerQuoteUsecase) CreateQuote(ctx context.Context, input *CreatePartnerQuoteInput) (*CreatePartnerQuoteOutput, error) {
	return u.createQuoteCore(ctx, input, true)
}

func (u *PartnerQuoteUsecase) PreviewQuote(ctx context.Context, input *CreatePartnerQuoteInput) (*CreatePartnerQuoteOutput, error) {
	return u.createQuoteCore(ctx, input, false)
}

func (u *PartnerQuoteUsecase) PreviewRequiredInputForOutput(ctx context.Context, input *PreviewRequiredInputForOutputInput) (*PreviewRequiredInputForOutputOutput, error) {
	ctx = withQuoteRequestCache(ctx)
	if input == nil {
		return nil, domainerrors.BadRequest("input is required")
	}
	if input.MerchantID == uuid.Nil {
		return nil, domainerrors.Forbidden("merchant context required")
	}
	if strings.TrimSpace(input.SelectedChain) == "" {
		return nil, domainerrors.BadRequest("selected_chain is required")
	}
	if strings.TrimSpace(input.InputToken) == "" {
		return nil, domainerrors.BadRequest("input_token is required")
	}
	if strings.TrimSpace(input.OutputToken) == "" {
		return nil, domainerrors.BadRequest("output_token is required")
	}
	if strings.TrimSpace(input.TargetOutputAmount) == "" {
		return nil, domainerrors.BadRequest("target_output_amount is required")
	}
	if u.routeSupportFn == nil || u.accurateRequiredInputFn == nil {
		return nil, domainerrors.BadRequest("accurate exact-output quote unavailable for selected pair")
	}

	chainID, _, err := u.chainResolver.ResolveFromAny(ctx, input.SelectedChain)
	if err != nil {
		return nil, domainerrors.BadRequest(fmt.Sprintf("invalid selected_chain: %v", err))
	}
	inputToken, err := u.getCachedTokenByAddress(ctx, chainID, strings.TrimSpace(input.InputToken))
	if err != nil || inputToken == nil || !inputToken.IsActive {
		return nil, domainerrors.BadRequest("input_token not supported on selected_chain")
	}
	outputToken, err := u.getCachedTokenByAddress(ctx, chainID, strings.TrimSpace(input.OutputToken))
	if err != nil || outputToken == nil || !outputToken.IsActive {
		return nil, domainerrors.BadRequest("output_token not supported on selected_chain")
	}
	targetOut := new(big.Int)
	if _, ok := targetOut.SetString(strings.TrimSpace(input.TargetOutputAmount), 10); !ok || targetOut.Sign() <= 0 {
		return nil, domainerrors.BadRequest("target_output_amount must be a positive integer string")
	}

	routeStatus, err := u.routeSupportFn(ctx, chainID, inputToken.ContractAddress, outputToken.ContractAddress)
	if err != nil {
		return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to resolve route support: %v", err))
	}
	if routeStatus == nil || !routeStatus.Exists || !routeStatus.Executable {
		return nil, domainerrors.BadRequest("selected token pair is not supported on selected_chain")
	}

	requiredInput, reqErr := u.accurateRequiredInputFn(ctx, chainID, inputToken.ContractAddress, outputToken.ContractAddress, targetOut)
	if reqErr != nil || requiredInput == nil || requiredInput.AmountIn == nil || requiredInput.AmountIn.Sign() <= 0 {
		return nil, domainerrors.BadRequest("accurate exact-output quote unavailable for selected pair")
	}

	priceSource := strings.TrimSpace(requiredInput.PriceSource)
	if priceSource == "" {
		chain, chainErr := u.getCachedChainByID(ctx, chainID)
		if chainErr == nil && chain != nil {
			priceSource = buildPartnerPriceSource(chain, outputToken.Symbol, inputToken.Symbol, routeStatus)
		} else {
			priceSource = "uniswap-v3-exact-output"
		}
	}

	routeSummary := u.summarizeRouteForProbe(inputToken, outputToken, routeStatus)
	return &PreviewRequiredInputForOutputOutput{
		RequiredInputAmount: requiredInput.AmountIn.String(),
		PriceSource:         priceSource,
		Route:               routeSummary,
	}, nil
}

func (u *PartnerQuoteUsecase) createQuoteCore(ctx context.Context, input *CreatePartnerQuoteInput, persist bool) (*CreatePartnerQuoteOutput, error) {
	ctx = withQuoteRequestCache(ctx)
	if input == nil {
		return nil, domainerrors.BadRequest("input is required")
	}
	if input.MerchantID == uuid.Nil {
		return nil, domainerrors.Forbidden("merchant context required")
	}
	if strings.TrimSpace(input.InvoiceCurrency) == "" {
		return nil, domainerrors.BadRequest("invoice_currency is required")
	}
	if strings.TrimSpace(input.InvoiceAmount) == "" {
		return nil, domainerrors.BadRequest("invoice_amount is required")
	}
	if strings.TrimSpace(input.SelectedChain) == "" {
		return nil, domainerrors.BadRequest("selected_chain is required")
	}
	if strings.TrimSpace(input.SelectedToken) == "" {
		return nil, domainerrors.BadRequest("selected_token is required")
	}
	if strings.TrimSpace(input.DestWallet) == "" {
		return nil, domainerrors.BadRequest("dest_wallet is required")
	}
	if u.routeSupportFn == nil || u.swapQuoteFn == nil {
		return nil, domainerrors.InternalServerError("partner quote engine is not configured")
	}
	if persist && u.quoteRepo == nil {
		return nil, domainerrors.InternalServerError("payment quote repository is not configured")
	}

	chainID, chainCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.SelectedChain)
	if err != nil {
		return nil, domainerrors.BadRequest(fmt.Sprintf("invalid selected_chain: %v", err))
	}
	chain, err := u.getCachedChainByID(ctx, chainID)
	if err != nil {
		return nil, domainerrors.BadRequest("selected_chain not found")
	}

	selectedToken, err := u.getCachedTokenByAddress(ctx, chainID, strings.TrimSpace(input.SelectedToken))
	if err != nil || selectedToken == nil || !selectedToken.IsActive {
		return nil, domainerrors.BadRequest("selected_token not supported on selected_chain")
	}

	var invoiceToken *domainentities.Token
	invoiceTokenAddr := strings.TrimSpace(input.InvoiceToken)
	if invoiceTokenAddr != "" {
		invoiceToken, err = u.getCachedTokenByAddress(ctx, chainID, invoiceTokenAddr)
		if err != nil || invoiceToken == nil || !invoiceToken.IsActive {
			return nil, domainerrors.BadRequest("invoice_token not supported on selected_chain")
		}
		if !strings.EqualFold(strings.TrimSpace(input.InvoiceCurrency), invoiceToken.Symbol) {
			return nil, domainerrors.BadRequest("invoice_currency does not match invoice_token symbol")
		}
	} else {
		invoiceToken, err = u.getCachedTokenBySymbol(ctx, chainID, strings.TrimSpace(input.InvoiceCurrency))
		if err != nil || invoiceToken == nil || !invoiceToken.IsActive {
			return nil, domainerrors.BadRequest("invoice_currency not supported on selected_chain")
		}
	}

	amountIn := new(big.Int)
	if _, ok := amountIn.SetString(strings.TrimSpace(input.InvoiceAmount), 10); !ok || amountIn.Sign() <= 0 {
		return nil, domainerrors.BadRequest("invoice_amount must be a positive integer string")
	}

	routeStatus, err := u.routeSupportFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress)
	if err != nil {
		return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to resolve route support: %v", err))
	}
	if routeStatus == nil || !routeStatus.Exists || !routeStatus.Executable {
		return nil, domainerrors.BadRequest("selected token pair is not supported on selected_chain")
	}

	var quotedAmount *big.Int
	priceSourceOverride := ""
	simulatorFallbackReason := ""
	if u.accurateQuoteFn != nil {
		accurateQuote, quoteErr := u.accurateQuoteFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress, amountIn)
		if quoteErr == nil && accurateQuote != nil {
			quotedAmount = accurateQuote.AmountOut
			priceSourceOverride = strings.TrimSpace(accurateQuote.PriceSource)
		} else {
			// Fallback to swapper quote for non-v3-direct or missing quoter configurations.
			quotedAmount, err = u.swapQuoteFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress, amountIn)
			if err != nil {
				// Last fallback: external simulator-backed quote.
				if u.simulatorQuoteFn != nil {
					simQuote, simErr := u.simulatorQuoteFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress, amountIn)
					if simErr == nil && simQuote != nil && simQuote.AmountOut != nil && simQuote.AmountOut.Sign() > 0 {
						quotedAmount = simQuote.AmountOut
						priceSourceOverride = strings.TrimSpace(simQuote.PriceSource)
					} else if simErr != nil {
						simulatorFallbackReason = simErr.Error()
					}
				}
				if quotedAmount == nil || quotedAmount.Sign() <= 0 {
					if quoteErr != nil {
						if simulatorFallbackReason != "" {
							return nil, domainerrors.BadRequest(fmt.Sprintf("accurate quote unavailable for selected pair: %v (simulator fallback failed: %s)", quoteErr, simulatorFallbackReason))
						}
						return nil, domainerrors.BadRequest(fmt.Sprintf("accurate quote unavailable for selected pair: %v", quoteErr))
					}
					return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to fetch on-chain quote: %v", err))
				}
			}
			if quotedAmount != nil &&
				quotedAmount.Cmp(amountIn) == 0 &&
				!strings.EqualFold(invoiceToken.ContractAddress, selectedToken.ContractAddress) {
				// Try simulator fallback before rejecting placeholder.
				if u.simulatorQuoteFn != nil {
					simQuote, simErr := u.simulatorQuoteFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress, amountIn)
					if simErr == nil && simQuote != nil && simQuote.AmountOut != nil && simQuote.AmountOut.Sign() > 0 {
						quotedAmount = simQuote.AmountOut
						priceSourceOverride = strings.TrimSpace(simQuote.PriceSource)
					} else if simErr != nil {
						simulatorFallbackReason = simErr.Error()
					}
				}
				if quotedAmount != nil &&
					quotedAmount.Cmp(amountIn) == 0 &&
					!strings.EqualFold(invoiceToken.ContractAddress, selectedToken.ContractAddress) {
					if simulatorFallbackReason != "" {
						return nil, domainerrors.BadRequest(fmt.Sprintf("accurate quote unavailable for selected pair: swapper returned 1:1 placeholder quote (simulator fallback failed: %s)", simulatorFallbackReason))
					}
					return nil, domainerrors.BadRequest("accurate quote unavailable for selected pair: swapper returned 1:1 placeholder quote")
				}
			}
		}
	} else {
		quotedAmount, err = u.swapQuoteFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress, amountIn)
		if err != nil {
			if u.simulatorQuoteFn != nil {
				simQuote, simErr := u.simulatorQuoteFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress, amountIn)
				if simErr == nil && simQuote != nil && simQuote.AmountOut != nil && simQuote.AmountOut.Sign() > 0 {
					quotedAmount = simQuote.AmountOut
					priceSourceOverride = strings.TrimSpace(simQuote.PriceSource)
				} else if simErr != nil {
					simulatorFallbackReason = simErr.Error()
				}
			}
			if quotedAmount == nil || quotedAmount.Sign() <= 0 {
				return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to fetch on-chain quote: %v", err))
			}
		}
		if quotedAmount != nil &&
			quotedAmount.Cmp(amountIn) == 0 &&
			!strings.EqualFold(invoiceToken.ContractAddress, selectedToken.ContractAddress) {
			if u.simulatorQuoteFn != nil {
				simQuote, simErr := u.simulatorQuoteFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress, amountIn)
				if simErr == nil && simQuote != nil && simQuote.AmountOut != nil && simQuote.AmountOut.Sign() > 0 {
					quotedAmount = simQuote.AmountOut
					priceSourceOverride = strings.TrimSpace(simQuote.PriceSource)
				} else if simErr != nil {
					simulatorFallbackReason = simErr.Error()
				}
			}
			if quotedAmount != nil &&
				quotedAmount.Cmp(amountIn) == 0 &&
				!strings.EqualFold(invoiceToken.ContractAddress, selectedToken.ContractAddress) {
				if simulatorFallbackReason != "" {
					return nil, domainerrors.BadRequest(fmt.Sprintf("accurate quote unavailable for selected pair: swapper returned 1:1 placeholder quote (simulator fallback failed: %s)", simulatorFallbackReason))
				}
				return nil, domainerrors.BadRequest("accurate quote unavailable for selected pair: swapper returned 1:1 placeholder quote")
			}
		}
	}
	if quotedAmount == nil || quotedAmount.Sign() <= 0 {
		return nil, domainerrors.InternalServerError("invalid on-chain quote amount")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(partnerQuoteTTL)
	routeSummary := u.summarizeRouteForProbe(invoiceToken, selectedToken, routeStatus)
	if persist {
		routeSummary = u.summarizeRoute(ctx, chainID, invoiceToken, selectedToken, routeStatus)
	}
	priceSource := buildPartnerPriceSource(chain, selectedToken.Symbol, invoiceToken.Symbol, routeStatus)
	if priceSourceOverride != "" {
		priceSource = priceSourceOverride
	}
	quoteRate := formatNormalizedTokenRatio(quotedAmount, selectedToken.Decimals, amountIn, invoiceToken.Decimals, 18)
	output := &CreatePartnerQuoteOutput{
		InvoiceCurrency:     strings.TrimSpace(input.InvoiceCurrency),
		InvoiceAmount:       amountIn.String(),
		SelectedChain:       chainCAIP2,
		SelectedToken:       selectedToken.ContractAddress,
		SelectedTokenSymbol: selectedToken.Symbol,
		QuotedAmount:        quotedAmount.String(),
		QuoteDecimals:       selectedToken.Decimals,
		QuoteRate:           quoteRate,
		PriceSource:         priceSource,
		Route:               routeSummary,
		SlippageBps:         partnerQuoteSlippage,
		RateTimestamp:       now,
		QuoteExpiresAt:      expiresAt,
	}
	if !persist {
		return output, nil
	}

	quote := &domainentities.PaymentQuote{
		ID:                    utils.GenerateUUIDv7(),
		MerchantID:            input.MerchantID,
		InvoiceCurrency:       output.InvoiceCurrency,
		InvoiceAmount:         output.InvoiceAmount,
		SelectedChainID:       output.SelectedChain,
		SelectedTokenAddress:  output.SelectedToken,
		SelectedTokenSymbol:   output.SelectedTokenSymbol,
		SelectedTokenDecimals: output.QuoteDecimals,
		QuotedAmount:          output.QuotedAmount,
		QuoteRate:             output.QuoteRate,
		PriceSource:           output.PriceSource,
		Route:                 output.Route,
		SlippageBps:           output.SlippageBps,
		RateTimestamp:         output.RateTimestamp,
		ExpiresAt:             output.QuoteExpiresAt,
		Status:                domainentities.PaymentQuoteStatusActive,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := u.quoteRepo.Create(ctx, quote); err != nil {
		return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to persist quote: %v", err))
	}
	output.QuoteID = quote.ID.String()
	return output, nil
}

func (u *PartnerQuoteUsecase) getCachedChainByID(ctx context.Context, chainID uuid.UUID) (*domainentities.Chain, error) {
	cache := getQuoteRequestCache(ctx)
	key := chainByIDCacheKey(chainID)
	if cache != nil {
		cache.mu.RLock()
		if cached, ok := cache.chainsByID[key]; ok {
			cache.mu.RUnlock()
			return cached, nil
		}
		cache.mu.RUnlock()
	}

	chain, err := u.chainRepo.GetByID(ctx, chainID)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		cache.mu.Lock()
		cache.chainsByID[key] = chain
		cache.mu.Unlock()
	}
	return chain, nil
}

func (u *PartnerQuoteUsecase) getCachedTokenByAddress(ctx context.Context, chainID uuid.UUID, tokenAddress string) (*domainentities.Token, error) {
	cache := getQuoteRequestCache(ctx)
	key := tokenByAddressCacheKey(chainID, tokenAddress)
	if cache != nil {
		cache.mu.RLock()
		if cached, ok := cache.tokensByAddress[key]; ok {
			cache.mu.RUnlock()
			return cached, nil
		}
		cache.mu.RUnlock()
	}

	token, err := u.tokenRepo.GetByAddress(ctx, tokenAddress, chainID)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		cache.mu.Lock()
		cache.tokensByAddress[key] = token
		cache.mu.Unlock()
	}
	return token, nil
}

func (u *PartnerQuoteUsecase) getCachedTokenBySymbol(ctx context.Context, chainID uuid.UUID, symbol string) (*domainentities.Token, error) {
	cache := getQuoteRequestCache(ctx)
	key := tokenBySymbolCacheKey(chainID, symbol)
	if cache != nil {
		cache.mu.RLock()
		if cached, ok := cache.tokensBySymbol[key]; ok {
			cache.mu.RUnlock()
			return cached, nil
		}
		cache.mu.RUnlock()
	}

	token, err := u.tokenRepo.GetBySymbol(ctx, symbol, chainID)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		cache.mu.Lock()
		cache.tokensBySymbol[key] = token
		cache.mu.Unlock()
	}
	return token, nil
}

func (u *PartnerQuoteUsecase) summarizeRouteForProbe(
	invoiceToken *domainentities.Token,
	selectedToken *domainentities.Token,
	status *TokenRouteSupportStatus,
) string {
	if status == nil || len(status.Path) == 0 {
		return fmt.Sprintf("%s->%s", invoiceToken.Symbol, selectedToken.Symbol)
	}
	parts := make([]string, 0, len(status.Path))
	for _, addr := range status.Path {
		if strings.EqualFold(addr, invoiceToken.ContractAddress) {
			parts = append(parts, invoiceToken.Symbol)
			continue
		}
		if strings.EqualFold(addr, selectedToken.ContractAddress) {
			parts = append(parts, selectedToken.Symbol)
			continue
		}
		parts = append(parts, addr)
	}
	return strings.Join(parts, "->")
}

func (u *PartnerQuoteUsecase) summarizeRoute(
	ctx context.Context,
	chainID uuid.UUID,
	invoiceToken *domainentities.Token,
	selectedToken *domainentities.Token,
	status *TokenRouteSupportStatus,
) string {
	if status == nil || len(status.Path) == 0 {
		return fmt.Sprintf("%s->%s", invoiceToken.Symbol, selectedToken.Symbol)
	}

	parts := make([]string, len(status.Path))
	type lookupTask struct {
		index int
		addr  string
	}
	lookups := make([]lookupTask, 0, len(status.Path))

	for idx, addr := range status.Path {
		if strings.EqualFold(addr, invoiceToken.ContractAddress) {
			parts[idx] = invoiceToken.Symbol
			continue
		}
		if strings.EqualFold(addr, selectedToken.ContractAddress) {
			parts[idx] = selectedToken.Symbol
			continue
		}
		parts[idx] = addr
		lookups = append(lookups, lookupTask{index: idx, addr: addr})
	}

	if len(lookups) > 0 {
		var wg sync.WaitGroup
		for _, task := range lookups {
			task := task
			wg.Add(1)
			go func() {
				defer wg.Done()
				token, err := u.getCachedTokenByAddress(ctx, chainID, task.addr)
				if err == nil && token != nil && strings.TrimSpace(token.Symbol) != "" {
					parts[task.index] = token.Symbol
				}
			}()
		}
		wg.Wait()
	}
	return strings.Join(parts, "->")
}

func buildPartnerPriceSource(chain *domainentities.Chain, selectedTokenSymbol string, invoiceTokenSymbol string, status *TokenRouteSupportStatus) string {
	prefix := "token-swapper"
	if status != nil {
		if strings.TrimSpace(status.UniversalV4) != "" && status.UniversalV4 != "0x0000000000000000000000000000000000000000" {
			prefix = "uniswap-v4"
		} else if strings.TrimSpace(status.SwapRouterV3) != "" && status.SwapRouterV3 != "0x0000000000000000000000000000000000000000" {
			prefix = "uniswap-v3"
		}
	}
	chainName := normalizePriceSourcePart(chain.Name)
	if chainName == "" {
		chainName = normalizePriceSourcePart(chain.GetCAIP2ID())
	}
	return fmt.Sprintf("%s-%s-%s-%s", prefix, chainName, normalizePriceSourcePart(selectedTokenSymbol), normalizePriceSourcePart(invoiceTokenSymbol))
}

func normalizePriceSourcePart(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "-")
	v = strings.ReplaceAll(v, "_", "-")
	return v
}

func formatDecimalRatio(numerator *big.Int, denominator *big.Int, precision int) string {
	if denominator == nil || denominator.Sign() == 0 {
		return "0"
	}
	if numerator == nil {
		return "0"
	}
	ratio := new(big.Rat).SetFrac(numerator, denominator)
	return trimTrailingZeros(ratio.FloatString(precision))
}

func formatNormalizedTokenRatio(outputAmount *big.Int, outputDecimals int, inputAmount *big.Int, inputDecimals int, precision int) string {
	if inputAmount == nil || inputAmount.Sign() == 0 {
		return "0"
	}
	if outputAmount == nil {
		return "0"
	}

	numerator := new(big.Rat).SetInt(outputAmount)
	denominator := new(big.Rat).SetInt(inputAmount)

	if inputDecimals > 0 {
		numerator.Mul(numerator, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(inputDecimals)), nil)))
	}
	if outputDecimals > 0 {
		denominator.Mul(denominator, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(outputDecimals)), nil)))
	}

	return trimTrailingZeros(new(big.Rat).Quo(numerator, denominator).FloatString(precision))
}

func trimTrailingZeros(v string) string {
	if !strings.Contains(v, ".") {
		return v
	}
	v = strings.TrimRight(v, "0")
	v = strings.TrimRight(v, ".")
	if v == "" {
		return "0"
	}
	return v
}
