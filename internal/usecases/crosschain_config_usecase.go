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
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/pkg/utils"
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
	LayerZeroConfigured   bool                      `json:"layerZeroConfigured"`
	FeeQuoteHealthy       bool                      `json:"feeQuoteHealthy"`
	OverallStatus         string                    `json:"overallStatus"`
	Issues                []ContractConfigCheckItem `json:"issues"`
}

type CrosschainBridgePreflight struct {
	BridgeType   uint8           `json:"bridgeType"`
	BridgeName   string          `json:"bridgeName"`
	Ready        bool            `json:"ready"`
	Checks       map[string]bool `json:"checks"`
	ErrorCode    string          `json:"errorCode,omitempty"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
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
	adapterUsecase *OnchainAdapterUsecase
}

func NewCrosschainConfigUsecase(
	chainRepo repositories.ChainRepository,
	tokenRepo repositories.TokenRepository,
	contractRepo repositories.SmartContractRepository,
	clientFactory *blockchain.ClientFactory,
	adapterUsecase *OnchainAdapterUsecase,
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
	layerZeroConfigured := status.LayerZeroConfigured
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
	if status.DefaultBridgeType == 2 && !layerZeroConfigured {
		issues = append(issues, ContractConfigCheckItem{
			Code:    "LAYERZERO_NOT_CONFIGURED",
			Status:  "ERROR",
			Message: "layerzero adapter destination is not configured (dstEid/peer)",
		})
	}

	feeQuoteHealthy := false
	if adapterRegistered {
		feeQuoteHealthy = u.checkFeeQuoteHealth(ctx, sourceChain, destChain, status.DefaultBridgeType)
		if !feeQuoteHealthy {
			issues = append(issues, ContractConfigCheckItem{
				Code:    "FEE_QUOTE_FAILED",
				Status:  "ERROR",
				Message: "fee quote call failed for this route",
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
		LayerZeroConfigured:   layerZeroConfigured,
		FeeQuoteHealthy:       feeQuoteHealthy,
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
	buildRow := func(bridgeType uint8) CrosschainBridgePreflight {
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
			row.Checks["routeConfigured"] = status.LayerZeroConfigured
		}
		row.Checks["adapterRegistered"] = hasAdapter
		if hasAdapter && row.Checks["routeConfigured"] {
			row.Checks["feeQuoteHealthy"] = u.checkFeeQuoteHealth(ctx, sourceChain, destChain, bridgeType)
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
				row.ErrorCode = "LAYERZERO_NOT_CONFIGURED"
				row.ErrorMessage = "missing dstEid or peer"
			}
			return row
		}
		if !row.Checks["feeQuoteHealthy"] {
			row.ErrorCode = "FEE_QUOTE_FAILED"
			row.ErrorMessage = "fee quote call failed for this bridge route"
		}
		return row
	}

	// preserve fixed order [0,1,2]
	for _, bt := range []uint8{0, 1, 2} {
		bridgeRows = append(bridgeRows, buildRow(bt))
	}

	policyExecutable := false
	for _, row := range bridgeRows {
		if row.BridgeType == status.DefaultBridgeType && row.Ready {
			policyExecutable = true
			break
		}
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
			contractType = entities.ContractTypeAdapterLayerZero
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
			Step:    "setLayerZeroConfig",
			Status:  "SKIPPED",
			Message: "manual layerzero route config required (dstEid/peer/options)",
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
		return "LAYERZERO"
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
	if sourceChain == nil || destChain == nil {
		return false
	}
	router, err := u.contractRepo.GetActiveContract(ctx, sourceChain.ID, entities.ContractTypeRouter)
	if err != nil || router == nil {
		return false
	}
	rpcURL := resolveRPCURL(sourceChain)
	if rpcURL == "" {
		return false
	}
	client, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return false
	}
	defer client.Close()

	sourceTokens, _, _ := u.tokenRepo.GetTokensByChain(ctx, sourceChain.ID, utils.PaginationParams{Page: 1, Limit: 1})
	destTokens, _, _ := u.tokenRepo.GetTokensByChain(ctx, destChain.ID, utils.PaginationParams{Page: 1, Limit: 1})
	sourceToken := common.Address{}
	destToken := common.Address{}
	if len(sourceTokens) > 0 && common.IsHexAddress(sourceTokens[0].ContractAddress) {
		sourceToken = common.HexToAddress(sourceTokens[0].ContractAddress)
	}
	if len(destTokens) > 0 && common.IsHexAddress(destTokens[0].ContractAddress) {
		destToken = common.HexToAddress(destTokens[0].ContractAddress)
	}

	messageTupleType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "paymentId", Type: "bytes32"},
		{Name: "receiver", Type: "address"},
		{Name: "sourceToken", Type: "address"},
		{Name: "destToken", Type: "address"},
		{Name: "amount", Type: "uint256"},
		{Name: "destChainId", Type: "string"},
		{Name: "minAmountOut", Type: "uint256"},
	})
	if err != nil {
		return false
	}
	stringType, _ := abi.NewType("string", "", nil)
	uint8Type, _ := abi.NewType("uint8", "", nil)
	args := abi.Arguments{
		{Type: stringType},
		{Type: uint8Type},
		{Type: messageTupleType},
	}
	type bridgeMessage struct {
		PaymentId    [32]byte
		Receiver     common.Address
		SourceToken  common.Address
		DestToken    common.Address
		Amount       *big.Int
		DestChainId  string
		MinAmountOut *big.Int
	}
	msgStruct := bridgeMessage{
		PaymentId:    [32]byte{},
		Receiver:     common.Address{},
		SourceToken:  sourceToken,
		DestToken:    destToken,
		Amount:       big.NewInt(1),
		DestChainId:  destChain.GetCAIP2ID(),
		MinAmountOut: big.NewInt(0),
	}
	packedArgs, err := args.Pack(destChain.GetCAIP2ID(), bridgeType, msgStruct)
	if err != nil {
		return false
	}
	methodSig := []byte("quotePaymentFee(string,uint8,(bytes32,address,address,address,uint256,string,uint256))")
	methodID := crypto.Keccak256(methodSig)[:4]
	calldata := append(methodID, packedArgs...)

	out, callErr := client.CallView(ctx, router.ContractAddress, calldata)
	return callErr == nil && len(out) > 0
}
