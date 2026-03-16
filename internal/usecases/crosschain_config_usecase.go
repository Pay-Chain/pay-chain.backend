package usecases

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/infrastructure/blockchain"
	"payment-kita.backend/pkg/utils"
)

type CrosschainRouteStatus struct {
	RouteKey              string                    `json:"routeKey"`
	SourceChainID         string                    `json:"sourceChainId"`
	SourceChainName       string                    `json:"sourceChainName"`
	DestChainID           string                    `json:"destChainId"`
	DestChainName         string                    `json:"destChainName"`
	DefaultBridgeType     uint8                     `json:"defaultBridgeType"`
	AdapterRegistered     bool                      `json:"adapterRegistered"`
	AdapterAddress        string                    `json:"adapterAddress"`
	HyperbridgeConfigured bool                      `json:"hyperbridgeConfigured"`
	CcipConfigured        bool                      `json:"ccipConfigured"`
	StargateConfigured   bool                      `json:"stargateConfigured"`
	FeeQuoteHealthy       bool                      `json:"feeQuoteHealthy"`
	QuoteSchemaMismatch   bool                      `json:"quoteSchemaMismatch"`
	QuotePathUsed         string                    `json:"quotePathUsed,omitempty"`
	QuoteFailureReason    string                    `json:"quoteFailureReason,omitempty"`
	OverallStatus         string                    `json:"overallStatus"`
	Issues                []ContractConfigCheckItem `json:"issues"`
}

type CrosschainBridgePreflight struct {
	BridgeType          uint8           `json:"bridgeType"`
	BridgeName          string          `json:"bridgeName"`
	Ready               bool            `json:"ready"`
	Checks              map[string]bool `json:"checks"`
	QuoteSchemaMismatch bool            `json:"quoteSchemaMismatch"`
	QuotePathUsed       string          `json:"quotePathUsed,omitempty"`
	QuoteFailureReason  string          `json:"quoteFailureReason,omitempty"`
	ErrorCode           string          `json:"errorCode,omitempty"`
	ErrorMessage        string          `json:"errorMessage,omitempty"`
}

type CrosschainPreflightResult struct {
	SourceChainID     string                      `json:"sourceChainId"`
	DestChainID       string                      `json:"destChainId"`
	DefaultBridgeType uint8                       `json:"defaultBridgeType"`
	FallbackMode      string                      `json:"fallbackMode"`
	FallbackOrder     []uint8                     `json:"fallbackOrder"`
	Bridges           []CrosschainBridgePreflight `json:"bridges"`
	PolicyExecutable  bool                        `json:"policyExecutable"`
	Issues            []ContractConfigCheckItem   `json:"issues"`
}

type CrosschainOverview struct {
	Items []CrosschainRouteStatus `json:"items"`
	Meta  utils.PaginationMeta    `json:"meta"`
}

type AutoFixRequest struct {
	SourceChainID string `json:"sourceChainId" binding:"required"`
	DestChainID   string `json:"destChainId" binding:"required"`
	BridgeType    *uint8 `json:"bridgeType,omitempty"`
}

type AutoFixStep struct {
	Step    string `json:"step"`
	Status  string `json:"status"` // SUCCESS, SKIPPED, FAILED
	Message string `json:"message"`
	TxHash  string `json:"txHash,omitempty"`
}

type AutoFixResult struct {
	SourceChainID string        `json:"sourceChainId"`
	DestChainID   string        `json:"destChainId"`
	BridgeType    uint8         `json:"bridgeType"`
	Steps         []AutoFixStep `json:"steps"`
}

type CrosschainConfigUsecase struct {
	chainRepo      repositories.ChainRepository
	tokenRepo      repositories.TokenRepository
	contractRepo   repositories.SmartContractRepository
	clientFactory  *blockchain.ClientFactory
	chainResolver  *ChainResolver
	adapterUsecase CrosschainAdapterUsecase
	feeQuoteHealth func(ctx context.Context, sourceChain, destChain *entities.Chain, bridgeType uint8) bool
}

type CrosschainAdapterUsecase interface {
	GetStatus(ctx context.Context, sourceChainInput, destChainInput string) (*OnchainAdapterStatus, error)
	RegisterAdapter(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8, adapterAddress string) (string, error)
	SetDefaultBridgeType(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8) (string, error)
	SetHyperbridgeConfig(
		ctx context.Context,
		sourceChainInput, destChainInput string,
		stateMachineIDHex, destinationContractHex string,
	) (string, []string, error)
}

func NewCrosschainConfigUsecase(
	chainRepo repositories.ChainRepository,
	tokenRepo repositories.TokenRepository,
	contractRepo repositories.SmartContractRepository,
	clientFactory *blockchain.ClientFactory,
	adapterUsecase CrosschainAdapterUsecase,
) *CrosschainConfigUsecase {
	return &CrosschainConfigUsecase{
		chainRepo:      chainRepo,
		tokenRepo:      tokenRepo,
		contractRepo:   contractRepo,
		clientFactory:  clientFactory,
		chainResolver:  NewChainResolver(chainRepo),
		adapterUsecase: adapterUsecase,
	}
}

func (u *CrosschainConfigUsecase) Overview(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	pagination utils.PaginationParams,
) (*CrosschainOverview, error) {
	chains, err := u.chainRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var sourceChains []*entities.Chain
	var destChains []*entities.Chain

	if strings.TrimSpace(sourceChainInput) != "" {
		sourceID, _, resolveErr := u.chainResolver.ResolveFromAny(ctx, sourceChainInput)
		if resolveErr != nil {
			return nil, domainerrors.BadRequest("invalid sourceChainId")
		}
		sourceChain, getErr := u.chainRepo.GetByID(ctx, sourceID)
		if getErr != nil {
			return nil, getErr
		}
		sourceChains = append(sourceChains, sourceChain)
	} else {
		for _, ch := range chains {
			if ch != nil && ch.IsActive && ch.Type == entities.ChainTypeEVM {
				sourceChains = append(sourceChains, ch)
			}
		}
	}

	if strings.TrimSpace(destChainInput) != "" {
		destID, _, resolveErr := u.chainResolver.ResolveFromAny(ctx, destChainInput)
		if resolveErr != nil {
			return nil, domainerrors.BadRequest("invalid destChainId")
		}
		destChain, getErr := u.chainRepo.GetByID(ctx, destID)
		if getErr != nil {
			return nil, getErr
		}
		destChains = append(destChains, destChain)
	} else {
		for _, ch := range chains {
			if ch != nil && ch.IsActive {
				destChains = append(destChains, ch)
			}
		}
	}

	var routes []CrosschainRouteStatus
	for _, source := range sourceChains {
		for _, dest := range destChains {
			if source == nil || dest == nil || source.ID == dest.ID {
				continue
			}
			status, statusErr := u.RecheckRoute(ctx, source.GetCAIP2ID(), dest.GetCAIP2ID())
			if statusErr != nil {
				routes = append(routes, CrosschainRouteStatus{
					RouteKey:        source.GetCAIP2ID() + "->" + dest.GetCAIP2ID(),
					SourceChainID:   source.GetCAIP2ID(),
					SourceChainName: source.Name,
					DestChainID:     dest.GetCAIP2ID(),
					DestChainName:   dest.Name,
					OverallStatus:   "ERROR",
					Issues: []ContractConfigCheckItem{
						{Code: "RECHECK_FAILED", Status: "ERROR", Message: statusErr.Error()},
					},
				})
				continue
			}
			routes = append(routes, *status)
		}
	}

	total := int64(len(routes))
	start := pagination.CalculateOffset()
	if start > len(routes) {
		start = len(routes)
	}
	end := start + pagination.Limit
	if pagination.Limit <= 0 || end > len(routes) {
		end = len(routes)
	}

	return &CrosschainOverview{
		Items: routes[start:end],
		Meta:  utils.CalculateMeta(total, pagination.Page, pagination.Limit),
	}, nil
}

func (u *CrosschainConfigUsecase) RecheckRoute(ctx context.Context, sourceChainInput, destChainInput string) (*CrosschainRouteStatus, error) {
	sourceID, sourceCAIP2, err := u.chainResolver.ResolveFromAny(ctx, sourceChainInput)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid sourceChainId")
	}
	destID, destCAIP2, err := u.chainResolver.ResolveFromAny(ctx, destChainInput)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid destChainId")
	}
	sourceChain, err := u.chainRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	destChain, err := u.chainRepo.GetByID(ctx, destID)
	if err != nil {
		return nil, err
	}

	status, err := u.adapterUsecase.GetStatus(ctx, sourceCAIP2, destCAIP2)
	if err != nil {
		return nil, err
	}

	issues := make([]ContractConfigCheckItem, 0)
	adapterRegistered := status.HasAdapterDefault && status.AdapterDefaultType != "" && status.AdapterDefaultType != "0x0000000000000000000000000000000000000000"
	if !adapterRegistered {
		issues = append(issues, ContractConfigCheckItem{
			Code:    "ADAPTER_NOT_REGISTERED",
			Status:  "ERROR",
			Message: "adapter is not registered for route default bridge type",
		})
	}

	ccipConfigured := status.CCIPChainSelector != 0 && status.CCIPDestinationAdapter != "" && status.CCIPDestinationAdapter != "0x"
	hyperConfigured := status.HyperbridgeConfigured
	stargateConfigured := status.StargateConfigured
	if status.DefaultBridgeType == 0 && !hyperConfigured {
		issues = append(issues, ContractConfigCheckItem{
			Code:    "HYPERBRIDGE_NOT_CONFIGURED",
			Status:  "ERROR",
			Message: "hyperbridge adapter destination is not configured (state machine/destination contract)",
		})
	}
	if status.DefaultBridgeType == 1 && !ccipConfigured {
		issues = append(issues, ContractConfigCheckItem{
			Code:    "CCIP_NOT_CONFIGURED",
			Status:  "ERROR",
			Message: "ccip adapter destination is not configured (chain selector/destination adapter)",
		})
	}
	if status.DefaultBridgeType == 2 && !stargateConfigured {
		issues = append(issues, ContractConfigCheckItem{
			Code:    "STARGATE_NOT_CONFIGURED",
			Status:  "ERROR",
			Message: "stargate adapter destination is not configured (dstEid/peer)",
		})
	}

	feeQuoteHealthy := false
	quotePathUsed := ""
	feeQuoteReason := ""
	quoteSchemaMismatch := false
	if adapterRegistered {
		feeQuoteHealthy, quotePathUsed, feeQuoteReason = u.evaluateFeeQuoteHealthDetailed(ctx, sourceChain, destChain, status.DefaultBridgeType)
		quoteSchemaMismatch = isQuoteSchemaMismatchReason(feeQuoteReason) || strings.Contains(strings.ToLower(strings.TrimSpace(feeQuoteReason)), "quote_failed_schema_mismatch")
		if !feeQuoteHealthy {
			message := "fee quote call failed for this route"
			if strings.TrimSpace(feeQuoteReason) != "" {
				message = fmt.Sprintf("%s: %s", message, strings.TrimSpace(feeQuoteReason))
			}
			issues = append(issues, ContractConfigCheckItem{
				Code:    "FEE_QUOTE_FAILED",
				Status:  "ERROR",
				Message: message,
			})
		}
	}

	overall := "READY"
	if len(issues) > 0 {
		overall = "ERROR"
	}

	return &CrosschainRouteStatus{
		RouteKey:              sourceCAIP2 + "->" + destCAIP2,
		SourceChainID:         sourceCAIP2,
		SourceChainName:       sourceChain.Name,
		DestChainID:           destCAIP2,
		DestChainName:         destChain.Name,
		DefaultBridgeType:     status.DefaultBridgeType,
		AdapterRegistered:     adapterRegistered,
		AdapterAddress:        status.AdapterDefaultType,
		HyperbridgeConfigured: hyperConfigured,
		CcipConfigured:        ccipConfigured,
		StargateConfigured:   stargateConfigured,
		FeeQuoteHealthy:       feeQuoteHealthy,
		QuoteSchemaMismatch:   quoteSchemaMismatch,
		QuotePathUsed:         quotePathUsed,
		QuoteFailureReason:    strings.TrimSpace(feeQuoteReason),
		OverallStatus:         overall,
		Issues:                issues,
	}, nil
}

func (u *CrosschainConfigUsecase) Preflight(ctx context.Context, sourceChainInput, destChainInput string) (*CrosschainPreflightResult, error) {
	sourceID, _, err := u.chainResolver.ResolveFromAny(ctx, sourceChainInput)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid sourceChainId")
	}
	destID, _, err := u.chainResolver.ResolveFromAny(ctx, destChainInput)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid destChainId")
	}
	sourceChain, err := u.chainRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	destChain, err := u.chainRepo.GetByID(ctx, destID)
	if err != nil {
		return nil, err
	}

	route, err := u.RecheckRoute(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return nil, err
	}
	status, err := u.adapterUsecase.GetStatus(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return nil, err
	}

	bridgeRows := make([]CrosschainBridgePreflight, 0, 3)
	policyExecutable := false

	// preserve fixed order [0,1,2]
	for _, bt := range []uint8{0, 1, 2} {
		row := u.buildPreflightRow(ctx, sourceChain, destChain, status, bt)
		if row.Ready && row.BridgeType == status.DefaultBridgeType {
			policyExecutable = true
		}
		bridgeRows = append(bridgeRows, row)
	}

	return &CrosschainPreflightResult{
		SourceChainID:     route.SourceChainID,
		DestChainID:       route.DestChainID,
		DefaultBridgeType: status.DefaultBridgeType,
		FallbackMode:      string(entities.BridgeFallbackModeStrict),
		FallbackOrder:     []uint8{status.DefaultBridgeType},
		Bridges:           bridgeRows,
		PolicyExecutable:  policyExecutable,
		Issues:            route.Issues,
	}, nil
}

func (u *CrosschainConfigUsecase) buildPreflightRow(
	ctx context.Context,
	sourceChain, destChain *entities.Chain,
	status *OnchainAdapterStatus,
	bridgeType uint8,
) CrosschainBridgePreflight {
	row := CrosschainBridgePreflight{
		BridgeType: bridgeType,
		BridgeName: bridgeName(bridgeType),
		Checks: map[string]bool{
			"adapterRegistered": false,
			"routeConfigured":   false,
			"feeQuoteHealthy":   false,
		},
	}

	var hasAdapter bool
	switch bridgeType {
	case 0:
		hasAdapter = status.HasAdapterType0 && status.AdapterType0 != "" && status.AdapterType0 != "0x0000000000000000000000000000000000000000"
		row.Checks["routeConfigured"] = status.HyperbridgeConfigured
	case 1:
		hasAdapter = status.HasAdapterType1 && status.AdapterType1 != "" && status.AdapterType1 != "0x0000000000000000000000000000000000000000"
		row.Checks["routeConfigured"] = status.CCIPChainSelector != 0 && status.CCIPDestinationAdapter != "" && status.CCIPDestinationAdapter != "0x"
	case 2:
		hasAdapter = status.HasAdapterType2 && status.AdapterType2 != "" && status.AdapterType2 != "0x0000000000000000000000000000000000000000"
		row.Checks["routeConfigured"] = status.StargateConfigured
	}
	row.Checks["adapterRegistered"] = hasAdapter
	feeQuoteReason := ""
	if hasAdapter && row.Checks["routeConfigured"] {
		row.Checks["feeQuoteHealthy"], row.QuotePathUsed, feeQuoteReason = u.evaluateFeeQuoteHealthDetailed(ctx, sourceChain, destChain, bridgeType)
		row.QuoteFailureReason = strings.TrimSpace(feeQuoteReason)
		row.QuoteSchemaMismatch = isQuoteSchemaMismatchReason(feeQuoteReason) || strings.Contains(strings.ToLower(strings.TrimSpace(feeQuoteReason)), "quote_failed_schema_mismatch")
	}

	if row.Checks["adapterRegistered"] && row.Checks["routeConfigured"] && row.Checks["feeQuoteHealthy"] {
		row.Ready = true
		return row
	}

	if !row.Checks["adapterRegistered"] {
		row.ErrorCode = "ADAPTER_NOT_REGISTERED"
		row.ErrorMessage = "adapter is not registered for this bridge type"
		return row
	}
	if !row.Checks["routeConfigured"] {
		switch bridgeType {
		case 0:
			row.ErrorCode = "HYPERBRIDGE_NOT_CONFIGURED"
			row.ErrorMessage = "missing state machine ID or destination contract"
		case 1:
			row.ErrorCode = "CCIP_NOT_CONFIGURED"
			row.ErrorMessage = "missing chain selector or destination adapter"
		case 2:
			row.ErrorCode = "STARGATE_NOT_CONFIGURED"
			row.ErrorMessage = "missing dstEid or peer"
		}
		return row
	}
	if !row.Checks["feeQuoteHealthy"] {
		row.ErrorCode = "FEE_QUOTE_FAILED"
		row.ErrorMessage = "fee quote call failed for this bridge route"
		if strings.TrimSpace(feeQuoteReason) != "" {
			row.ErrorMessage = fmt.Sprintf("%s: %s", row.ErrorMessage, strings.TrimSpace(feeQuoteReason))
		}
	}
	return row
}

func (u *CrosschainConfigUsecase) AutoFix(ctx context.Context, req *AutoFixRequest) (*AutoFixResult, error) {
	status, err := u.adapterUsecase.GetStatus(ctx, req.SourceChainID, req.DestChainID)
	if err != nil {
		return nil, err
	}
	bridgeType := status.DefaultBridgeType
	if req.BridgeType != nil {
		bridgeType = *req.BridgeType
	}

	result := &AutoFixResult{
		SourceChainID: req.SourceChainID,
		DestChainID:   req.DestChainID,
		BridgeType:    bridgeType,
		Steps:         []AutoFixStep{},
	}

	adapterAddress := ""
	if bridgeType == 0 {
		adapterAddress = status.AdapterType0
	} else if bridgeType == 1 {
		adapterAddress = status.AdapterType1
	} else {
		adapterAddress = status.AdapterType2
	}

	if adapterAddress == "" || adapterAddress == "0x0000000000000000000000000000000000000000" {
		sourceUUID, _, resolveErr := u.chainResolver.ResolveFromAny(ctx, req.SourceChainID)
		if resolveErr != nil {
			return nil, domainerrors.BadRequest("invalid sourceChainId")
		}
		contractType := entities.ContractTypeAdapterHyperbridge
		if bridgeType == 1 {
			contractType = entities.ContractTypeAdapterCCIP
		} else if bridgeType == 2 {
			contractType = entities.ContractTypeAdapterStargate
		}
		adapterContract, getErr := u.contractRepo.GetActiveContract(ctx, sourceUUID, contractType)
		if getErr != nil || adapterContract == nil {
			result.Steps = append(result.Steps, AutoFixStep{
				Step:    "registerAdapter",
				Status:  "FAILED",
				Message: "active adapter contract not found on source chain",
			})
			return result, nil
		}
		txHash, regErr := u.adapterUsecase.RegisterAdapter(ctx, req.SourceChainID, req.DestChainID, bridgeType, adapterContract.ContractAddress)
		if regErr != nil {
			result.Steps = append(result.Steps, AutoFixStep{
				Step:    "registerAdapter",
				Status:  "FAILED",
				Message: regErr.Error(),
			})
			return result, nil
		}
		result.Steps = append(result.Steps, AutoFixStep{
			Step:    "registerAdapter",
			Status:  "SUCCESS",
			Message: "adapter registered",
			TxHash:  txHash,
		})
		adapterAddress = adapterContract.ContractAddress
	} else {
		result.Steps = append(result.Steps, AutoFixStep{
			Step:    "registerAdapter",
			Status:  "SKIPPED",
			Message: "adapter already registered",
		})
	}

	if status.DefaultBridgeType != bridgeType {
		txHash, setErr := u.adapterUsecase.SetDefaultBridgeType(ctx, req.SourceChainID, req.DestChainID, bridgeType)
		if setErr != nil {
			result.Steps = append(result.Steps, AutoFixStep{
				Step:    "setDefaultBridge",
				Status:  "FAILED",
				Message: setErr.Error(),
			})
			return result, nil
		}
		result.Steps = append(result.Steps, AutoFixStep{
			Step:    "setDefaultBridge",
			Status:  "SUCCESS",
			Message: "default bridge updated",
			TxHash:  txHash,
		})
	} else {
		result.Steps = append(result.Steps, AutoFixStep{
			Step:    "setDefaultBridge",
			Status:  "SKIPPED",
			Message: "default bridge already set",
		})
	}

	if bridgeType == 0 {
		stateMachineHex := deriveEvmStateMachineHex(req.DestChainID)
		destHex, deriveErr := u.deriveDestinationContractHex(ctx, req.DestChainID, entities.ContractTypeAdapterHyperbridge)
		if deriveErr != nil {
			result.Steps = append(result.Steps, AutoFixStep{
				Step:    "setHyperbridgeDestination",
				Status:  "FAILED",
				Message: deriveErr.Error(),
			})
			return result, nil
		}

		_, txHashes, setErr := u.adapterUsecase.SetHyperbridgeConfig(ctx, req.SourceChainID, req.DestChainID, stateMachineHex, destHex)
		if setErr != nil {
			result.Steps = append(result.Steps, AutoFixStep{
				Step:    "setHyperbridgeConfig",
				Status:  "FAILED",
				Message: setErr.Error(),
			})
			return result, nil
		}
		result.Steps = append(result.Steps, AutoFixStep{
			Step:    "setHyperbridgeConfig",
			Status:  "SUCCESS",
			Message: "hyperbridge route configured",
			TxHash:  strings.Join(txHashes, ","),
		})
	}
	if bridgeType == 2 {
		result.Steps = append(result.Steps, AutoFixStep{
			Step:    "setStargateConfig",
			Status:  "SKIPPED",
			Message: "manual stargate route config required (dstEid/peer/options)",
		})
	}

	return result, nil
}

func bridgeName(bridgeType uint8) string {
	switch bridgeType {
	case 0:
		return "HYPERBRIDGE"
	case 1:
		return "CCIP"
	case 2:
		return "STARGATE"
	case 3:
		return "HYPERBRIDGE_TOKEN_GATEWAY"
	default:
		return "UNKNOWN"
	}
}

func (u *CrosschainConfigUsecase) deriveDestinationContractHex(ctx context.Context, destChainInput string, contractType entities.SmartContractType) (string, error) {
	destUUID, _, err := u.chainResolver.ResolveFromAny(ctx, destChainInput)
	if err != nil {
		return "", domainerrors.BadRequest("invalid destChainId")
	}
	contract, err := u.contractRepo.GetActiveContract(ctx, destUUID, contractType)
	if err != nil || contract == nil {
		return "", fmt.Errorf("active destination contract (%s) not found", contractType)
	}
	return addressToPaddedBytesHex(contract.ContractAddress)
}

func addressToPaddedBytesHex(address string) (string, error) {
	if !common.IsHexAddress(address) {
		return "", fmt.Errorf("invalid hex address")
	}
	addr := common.HexToAddress(address)
	padded := common.LeftPadBytes(addr.Bytes(), 32)
	return "0x" + hex.EncodeToString(padded), nil
}

func deriveEvmStateMachineHex(caip2 string) string {
	parts := strings.Split(strings.TrimSpace(caip2), ":")
	if len(parts) == 2 && strings.EqualFold(parts[0], "eip155") && parts[1] != "" {
		raw := []byte("EVM-" + parts[1])
		return "0x" + hex.EncodeToString(raw)
	}
	return ""
}

func (u *CrosschainConfigUsecase) checkFeeQuoteHealth(ctx context.Context, sourceChain, destChain *entities.Chain, bridgeType uint8) bool {
	ok, _ := u.checkFeeQuoteHealthWithReason(ctx, sourceChain, destChain, bridgeType)
	return ok
}

func (u *CrosschainConfigUsecase) evaluateFeeQuoteHealthDetailed(
	ctx context.Context,
	sourceChain, destChain *entities.Chain,
	bridgeType uint8,
) (bool, string, string) {
	ok, reason := u.evaluateFeeQuoteHealthWithReason(ctx, sourceChain, destChain, bridgeType)
	if ok {
		return true, "safe_or_legacy", ""
	}
	path := ""
	reasonTrimmed := strings.TrimSpace(reason)
	switch {
	case strings.Contains(reasonTrimmed, "stage=quotePaymentFeeSafe"):
		path = "safe"
	case strings.Contains(reasonTrimmed, "stage=quotePaymentFee(legacy)"):
		path = "legacy"
	case strings.Contains(reasonTrimmed, "stage=quotePaymentFee"):
		path = "v2"
	}
	return false, path, reason
}

func (u *CrosschainConfigUsecase) checkFeeQuoteHealthWithReason(
	ctx context.Context,
	sourceChain, destChain *entities.Chain,
	bridgeType uint8,
) (bool, string) {
	if sourceChain == nil || destChain == nil {
		return false, "source/destination chain is missing"
	}
	router, err := u.contractRepo.GetActiveContract(ctx, sourceChain.ID, entities.ContractTypeRouter)
	if err != nil || router == nil {
		return false, "router contract is not configured"
	}
	rpcURLs := resolveRPCURLs(sourceChain)
	if len(rpcURLs) == 0 {
		return false, "active rpc is not configured"
	}
	lastReason := "execution reverted"
	rpcErrors := make([]string, 0, len(rpcURLs))
	for _, rpcURL := range rpcURLs {
		client, cErr := u.clientFactory.GetEVMClient(rpcURL)
		if cErr != nil {
			lastReason = fmt.Sprintf("rpc=%s: %s", rpcURL, cErr.Error())
			rpcErrors = append(rpcErrors, lastReason)
			continue
		}
		ok, _, reason := u.checkFeeQuoteHealthWithReasonOnClient(
			ctx,
			client,
			sourceChain,
			destChain,
			bridgeType,
			router.ContractAddress,
			rpcURL,
		)
		client.Close()
		if ok {
			return true, ""
		}
		if strings.TrimSpace(reason) != "" {
			lastReason = reason
			rpcErrors = append(rpcErrors, reason)
		}
	}
	if len(rpcErrors) > 0 {
		return false, summarizeFeeQuoteRPCFailures(rpcErrors)
	}
	return false, lastReason
}

func (u *CrosschainConfigUsecase) checkFeeQuoteHealthWithReasonOnClient(
	ctx context.Context,
	client *blockchain.EVMClient,
	sourceChain, destChain *entities.Chain,
	bridgeType uint8,
	routerAddress string,
	rpcURL string,
) (bool, string, string) {
	if runtimeOK, runtimeReason := u.checkAdapterRuntimeReadiness(
		ctx,
		client,
		sourceChain,
		destChain.GetCAIP2ID(),
		routerAddress,
		bridgeType,
	); !runtimeOK {
		return false, "", runtimeReason
	}

	sourceTokens, _, _ := u.tokenRepo.GetTokensByChain(ctx, sourceChain.ID, utils.PaginationParams{Page: 1, Limit: 200})
	destTokens, _, _ := u.tokenRepo.GetTokensByChain(ctx, destChain.ID, utils.PaginationParams{Page: 1, Limit: 200})
	tokenPairs := buildFeeQuoteTokenPairs(sourceTokens, destTokens)
	if len(tokenPairs) == 0 {
		return false, "", "no valid source/destination token pair for quote"
	}

	messageTupleTypeV2, _ := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "paymentId", Type: "bytes32"},
		{Name: "receiver", Type: "address"},
		{Name: "sourceToken", Type: "address"},
		{Name: "destToken", Type: "address"},
		{Name: "amount", Type: "uint256"},
		{Name: "destChainId", Type: "string"},
		{Name: "minAmountOut", Type: "uint256"},
		{Name: "payer", Type: "address"},
	})
	messageTupleTypeV1, _ := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "paymentId", Type: "bytes32"},
		{Name: "receiver", Type: "address"},
		{Name: "sourceToken", Type: "address"},
		{Name: "destToken", Type: "address"},
		{Name: "amount", Type: "uint256"},
		{Name: "destChainId", Type: "string"},
		{Name: "minAmountOut", Type: "uint256"},
	})
	stringType, _ := abi.NewType("string", "", nil)
	uint8Type, _ := abi.NewType("uint8", "", nil)
	argsV2 := abi.Arguments{
		{Type: stringType},
		{Type: uint8Type},
		{Type: messageTupleTypeV2},
	}
	argsV1 := abi.Arguments{
		{Type: stringType},
		{Type: uint8Type},
		{Type: messageTupleTypeV1},
	}
	type bridgeMessageV2 struct {
		PaymentId    [32]byte
		Receiver     common.Address
		SourceToken  common.Address
		DestToken    common.Address
		Amount       *big.Int
		DestChainId  string
		MinAmountOut *big.Int
		Payer        common.Address
	}
	type bridgeMessageV1 struct {
		PaymentId    [32]byte
		Receiver     common.Address
		SourceToken  common.Address
		DestToken    common.Address
		Amount       *big.Int
		DestChainId  string
		MinAmountOut *big.Int
	}
	lastReason := "execution reverted"
	lastPath := ""
	for _, pair := range tokenPairs {
		amountCandidates := pair.amounts
		if len(amountCandidates) == 0 {
			amountCandidates = []*big.Int{big.NewInt(1)}
		}
		for _, sampleAmount := range amountCandidates {
			msgStructV2 := bridgeMessageV2{
				PaymentId:    [32]byte{},
				Receiver:     common.Address{},
				SourceToken:  pair.sourceToken,
				DestToken:    pair.destToken,
				Amount:       sampleAmount,
				DestChainId:  destChain.GetCAIP2ID(),
				MinAmountOut: big.NewInt(0),
				Payer:        common.Address{},
			}
			msgStructV1 := bridgeMessageV1{
				PaymentId:    [32]byte{},
				Receiver:     common.Address{},
				SourceToken:  pair.sourceToken,
				DestToken:    pair.destToken,
				Amount:       sampleAmount,
				DestChainId:  destChain.GetCAIP2ID(),
				MinAmountOut: big.NewInt(0),
			}
			packedArgsV2, _ := argsV2.Pack(destChain.GetCAIP2ID(), bridgeType, msgStructV2)
			packedArgsV1, _ := argsV1.Pack(destChain.GetCAIP2ID(), bridgeType, msgStructV1)

			// Prefer safe quote path (newer router) to get exact failure reason.
			safeMethodSig := []byte("quotePaymentFeeSafe(string,uint8,(bytes32,address,address,address,uint256,string,uint256,address))")
			safeMethodID := crypto.Keccak256(safeMethodSig)[:4]
			safeCalldata := append(safeMethodID, packedArgsV2...)
			lastPath = "safe"
			safeSchemaMismatch := false
			if safeOut, safeErr := client.CallView(ctx, routerAddress, safeCalldata); safeErr == nil {
				if len(safeOut) > 0 {
					boolType, _ := abi.NewType("bool", "", nil)
					uint256Type, _ := abi.NewType("uint256", "", nil)
					stringType, _ := abi.NewType("string", "", nil)
					safeOutputs := abi.Arguments{
						{Type: boolType},
						{Type: uint256Type},
						{Type: stringType},
					}
					if decoded, unpackErr := safeOutputs.Unpack(safeOut); unpackErr == nil && len(decoded) == 3 {
						ok, _ := decoded[0].(bool)
						reason, _ := decoded[2].(string)
						if ok {
							return true, "safe", ""
						}
						if isQuoteSchemaMismatchReason(reason) {
							lastReason = formatFeeQuoteAttemptReason(
								rpcURL,
								pair.sourceToken,
								pair.destToken,
								sampleAmount,
								"quotePaymentFeeSafe",
								"quote_failed_schema_mismatch",
							)
							continue
						}
						if reason != "execution_reverted" && strings.TrimSpace(reason) != "" {
							lastReason = formatFeeQuoteAttemptReason(
								rpcURL,
								pair.sourceToken,
								pair.destToken,
								sampleAmount,
								"quotePaymentFeeSafe",
								reason,
							)
							continue
						}
					}
				}
			} else if decoded, decodedOK := decodeRevertDataFromError(safeErr); decodedOK {
				lastReason = formatFeeQuoteAttemptReason(
					rpcURL,
					pair.sourceToken,
					pair.destToken,
					sampleAmount,
					"quotePaymentFeeSafe",
					formatDecodedRouteErrorForPreflight(decoded),
				)
				continue
			} else if isQuoteSchemaMismatchReason(safeErr.Error()) {
				safeSchemaMismatch = true
			} else {
				lastReason = formatFeeQuoteAttemptReason(
					rpcURL,
					pair.sourceToken,
					pair.destToken,
					sampleAmount,
					"quotePaymentFeeSafe",
					safeErr.Error(),
				)
				continue
			}

			methodSig := []byte("quotePaymentFee(string,uint8,(bytes32,address,address,address,uint256,string,uint256,address))")
			methodID := crypto.Keccak256(methodSig)[:4]
			calldata := append(methodID, packedArgsV2...)
			lastPath = "v2"

			out, callErr := client.CallView(ctx, routerAddress, calldata)
			if callErr != nil {
				// legacy fallback for older router tuple schema
				legacySig := []byte("quotePaymentFee(string,uint8,(bytes32,address,address,address,uint256,string,uint256))")
				legacyID := crypto.Keccak256(legacySig)[:4]
				legacyCalldata := append(legacyID, packedArgsV1...)
				lastPath = "legacy"
				out, callErr = client.CallView(ctx, routerAddress, legacyCalldata)
				if callErr != nil {
					if decoded, decodedOK := decodeRevertDataFromError(callErr); decodedOK {
						lastReason = formatFeeQuoteAttemptReason(
							rpcURL,
							pair.sourceToken,
							pair.destToken,
							sampleAmount,
							"quotePaymentFee(legacy)",
							formatDecodedRouteErrorForPreflight(decoded),
						)
						continue
					}
					if isQuoteSchemaMismatchReason(callErr.Error()) || safeSchemaMismatch {
						lastReason = formatFeeQuoteAttemptReason(
							rpcURL,
							pair.sourceToken,
							pair.destToken,
							sampleAmount,
							"quotePaymentFee(legacy)",
							"quote_failed_schema_mismatch",
						)
						continue
					}
					lastReason = formatFeeQuoteAttemptReason(
						rpcURL,
						pair.sourceToken,
						pair.destToken,
						sampleAmount,
						"quotePaymentFee(legacy)",
						callErr.Error(),
					)
					continue
				}
			}
			if len(out) == 0 {
				if safeSchemaMismatch {
					lastReason = formatFeeQuoteAttemptReason(
						rpcURL,
						pair.sourceToken,
						pair.destToken,
						sampleAmount,
						"quotePaymentFee",
						"quote_failed_schema_mismatch",
					)
					continue
				}
				lastReason = formatFeeQuoteAttemptReason(
					rpcURL,
					pair.sourceToken,
					pair.destToken,
					sampleAmount,
					"quotePaymentFee",
					"empty quote response",
				)
				continue
			}
			return true, lastPath, ""
		}
	}
	return false, lastPath, lastReason
}

type feeQuoteTokenPair struct {
	sourceToken common.Address
	destToken   common.Address
	amounts     []*big.Int
}

func buildFeeQuoteTokenPairs(sourceTokens []*entities.Token, destTokens []*entities.Token) []feeQuoteTokenPair {
	type sourceTokenCandidate struct {
		addr     common.Address
		symbol   string
		decimals int
	}

	toAddr := func(token *entities.Token) (common.Address, bool) {
		if token == nil || !common.IsHexAddress(token.ContractAddress) {
			return common.Address{}, false
		}
		addr := common.HexToAddress(token.ContractAddress)
		if addr == (common.Address{}) {
			return common.Address{}, false
		}
		return addr, true
	}

	buildAmountCandidates := func(decimals int) []*big.Int {
		one := big.NewInt(1)
		candidates := []*big.Int{new(big.Int).Set(one)}
		if decimals > 0 && decimals <= 36 {
			scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
			if scale.Sign() > 0 {
				candidates = append(candidates, scale)
			}
		}
		// Deduplicate while preserving order.
		seen := make(map[string]struct{}, len(candidates))
		out := make([]*big.Int, 0, len(candidates))
		for _, c := range candidates {
			if c == nil || c.Sign() <= 0 {
				continue
			}
			key := c.String()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, c)
		}
		return out
	}

	sourceCandidates := make([]sourceTokenCandidate, 0, len(sourceTokens))
	destBySymbol := make(map[string][]common.Address)
	for _, token := range destTokens {
		symbol := strings.TrimSpace(strings.ToUpper(token.Symbol))
		if symbol == "" {
			continue
		}
		if addr, ok := toAddr(token); ok {
			destBySymbol[symbol] = append(destBySymbol[symbol], addr)
		}
	}
	for _, token := range sourceTokens {
		symbol := strings.TrimSpace(strings.ToUpper(token.Symbol))
		if symbol == "" {
			continue
		}
		if addr, ok := toAddr(token); ok {
			sourceCandidates = append(sourceCandidates, sourceTokenCandidate{
				addr:     addr,
				symbol:   symbol,
				decimals: token.Decimals,
			})
		}
	}

	pairs := make([]feeQuoteTokenPair, 0, 8)
	seen := make(map[string]struct{})
	addPair := func(src, dst common.Address, amounts []*big.Int) {
		key := strings.ToLower(src.Hex()) + "->" + strings.ToLower(dst.Hex())
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		pairs = append(pairs, feeQuoteTokenPair{sourceToken: src, destToken: dst, amounts: amounts})
	}

	for _, src := range sourceCandidates {
		for _, dstAddr := range destBySymbol[src.symbol] {
			addPair(src.addr, dstAddr, buildAmountCandidates(src.decimals))
		}
	}

	// Fallback: first valid pair if symbols don't match.
	if len(pairs) == 0 {
		var srcFallback common.Address
		var dstFallback common.Address
		for _, token := range sourceTokens {
			if addr, ok := toAddr(token); ok {
				srcFallback = addr
				break
			}
		}
		for _, token := range destTokens {
			if addr, ok := toAddr(token); ok {
				dstFallback = addr
				break
			}
		}
		if srcFallback != (common.Address{}) && dstFallback != (common.Address{}) {
			addPair(srcFallback, dstFallback, []*big.Int{big.NewInt(1)})
		}
	}

	return pairs
}

func formatDecodedRouteErrorForPreflight(decoded RouteErrorDecoded) string {
	name := strings.TrimSpace(decoded.Name)
	msg := strings.TrimSpace(decoded.Message)
	selector := strings.TrimSpace(decoded.Selector)
	if isQuoteSchemaMismatchReason(msg) {
		return "quote_failed_schema_mismatch"
	}
	if msg != "" && msg != "execution_reverted" && msg != "no route error recorded" {
		return msg
	}
	if name != "" {
		switch name {
		case "NativeFeeQuoteUnavailable":
			return "native fee quote unavailable"
		case "FeeQuoteFailed":
			return "fee quote failed"
		case "RouteNotConfigured":
			return "route not configured"
		case "ChainSelectorMissing":
			return "ccip chain selector missing"
		case "DestinationAdapterMissing":
			return "destination adapter missing"
		case "StateMachineIdNotSet":
			return "hyperbridge state machine id not set"
		case "DestinationNotSet":
			return "hyperbridge destination not set"
		case "InsufficientNativeFee":
			return "insufficient native fee"
		}
		return name
	}
	if selector != "" {
		return "execution_reverted (" + selector + ")"
	}
	return "execution_reverted"
}

func formatFeeQuoteAttemptReason(
	rpcURL string,
	sourceToken common.Address,
	destToken common.Address,
	amount *big.Int,
	stage string,
	reason string,
) string {
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = "execution reverted"
	}
	amountText := "0"
	if amount != nil {
		amountText = amount.String()
	}
	return fmt.Sprintf(
		"rpc=%s stage=%s sourceToken=%s destToken=%s amount=%s reason=%s",
		rpcURL,
		stage,
		sourceToken.Hex(),
		destToken.Hex(),
		amountText,
		trimmedReason,
	)
}

func summarizeFeeQuoteRPCFailures(errors []string) string {
	if len(errors) == 0 {
		return "execution reverted"
	}
	const maxItems = 3
	items := errors
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	return strings.Join(items, " | ")
}

func (u *CrosschainConfigUsecase) checkAdapterRuntimeReadiness(
	ctx context.Context,
	client *blockchain.EVMClient,
	sourceChain *entities.Chain,
	destCAIP2 string,
	routerAddress string,
	bridgeType uint8,
) (bool, string) {
	// Stargate sender does not pull tokens from Vault.
	if bridgeType == 2 {
		return true, ""
	}
	if bridgeType != 0 && bridgeType != 1 {
		return true, ""
	}

	gateway, err := u.contractRepo.GetActiveContract(ctx, sourceChain.ID, entities.ContractTypeGateway)
	if err != nil || gateway == nil || !common.IsHexAddress(gateway.ContractAddress) {
		return false, "gateway contract is not configured"
	}
	if !common.IsHexAddress(routerAddress) {
		return false, "router contract address is invalid"
	}

	gatewayVault, err := u.callAddressView(ctx, client, gateway.ContractAddress, "vault()")
	if err != nil || gatewayVault == (common.Address{}) {
		return false, "failed to resolve gateway vault"
	}

	adapter, err := u.callRouterAdapter(ctx, client, routerAddress, destCAIP2, bridgeType)
	if err != nil {
		return false, "failed to resolve bridge adapter"
	}
	if adapter == (common.Address{}) {
		return false, "adapter is not registered for this bridge type"
	}

	adapterVault, err := u.callAddressView(ctx, client, adapter.Hex(), "vault()")
	if err != nil || adapterVault == (common.Address{}) {
		return false, "failed to resolve adapter vault"
	}
	if !strings.EqualFold(gatewayVault.Hex(), adapterVault.Hex()) {
		return false, fmt.Sprintf("adapter vault mismatch (gateway=%s adapter=%s)", gatewayVault.Hex(), adapterVault.Hex())
	}

	authorized, err := u.callVaultAuthorizedSpender(ctx, client, gatewayVault.Hex(), adapter)
	if err != nil {
		return false, "failed to verify vault authorization"
	}
	if !authorized {
		return false, "adapter is not authorized spender on vault"
	}

	return true, ""
}

func (u *CrosschainConfigUsecase) callRouterAdapter(
	ctx context.Context,
	client *blockchain.EVMClient,
	routerAddress, destCAIP2 string,
	bridgeType uint8,
) (common.Address, error) {
	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return common.Address{}, err
	}
	uint8Type, err := abi.NewType("uint8", "", nil)
	if err != nil {
		return common.Address{}, err
	}
	args := abi.Arguments{
		{Type: stringType},
		{Type: uint8Type},
	}
	packedArgs, err := args.Pack(destCAIP2, bridgeType)
	if err != nil {
		return common.Address{}, err
	}
	methodID := crypto.Keccak256([]byte("getAdapter(string,uint8)"))[:4]
	out, err := client.CallView(ctx, routerAddress, append(methodID, packedArgs...))
	if err != nil {
		return common.Address{}, err
	}
	if len(out) < 32 {
		return common.Address{}, fmt.Errorf("invalid getAdapter response")
	}
	return common.BytesToAddress(out[len(out)-20:]), nil
}

func (u *CrosschainConfigUsecase) callAddressView(
	ctx context.Context,
	client *blockchain.EVMClient,
	contractAddress, signature string,
) (common.Address, error) {
	methodID := crypto.Keccak256([]byte(signature))[:4]
	out, err := client.CallView(ctx, contractAddress, methodID)
	if err != nil {
		return common.Address{}, err
	}
	if len(out) < 32 {
		return common.Address{}, fmt.Errorf("invalid %s response", signature)
	}
	return common.BytesToAddress(out[len(out)-20:]), nil
}

func (u *CrosschainConfigUsecase) callVaultAuthorizedSpender(
	ctx context.Context,
	client *blockchain.EVMClient,
	vaultAddress string,
	spender common.Address,
) (bool, error) {
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return false, err
	}
	args := abi.Arguments{{Type: addressType}}
	packedArgs, err := args.Pack(spender)
	if err != nil {
		return false, err
	}
	methodID := crypto.Keccak256([]byte("authorizedSpenders(address)"))[:4]
	out, err := client.CallView(ctx, vaultAddress, append(methodID, packedArgs...))
	if err != nil {
		return false, err
	}
	if len(out) == 0 {
		return false, fmt.Errorf("empty authorizedSpenders response")
	}
	return new(big.Int).SetBytes(out).Sign() > 0, nil
}

func (u *CrosschainConfigUsecase) evaluateFeeQuoteHealth(ctx context.Context, sourceChain, destChain *entities.Chain, bridgeType uint8) bool {
	ok, _ := u.evaluateFeeQuoteHealthWithReason(ctx, sourceChain, destChain, bridgeType)
	return ok
}

func (u *CrosschainConfigUsecase) evaluateFeeQuoteHealthWithReason(
	ctx context.Context,
	sourceChain, destChain *entities.Chain,
	bridgeType uint8,
) (bool, string) {
	if u.feeQuoteHealth != nil {
		if u.feeQuoteHealth(ctx, sourceChain, destChain, bridgeType) {
			return true, ""
		}
		return false, "bridge route fee quote failed"
	}
	return u.checkFeeQuoteHealthWithReason(ctx, sourceChain, destChain, bridgeType)
}
