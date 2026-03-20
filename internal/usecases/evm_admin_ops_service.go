package usecases

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
)

type evmAdminContext struct {
	sourceChainID  uuid.UUID
	destCAIP2      string
	routerAddress  string
	gatewayAddress string
	vaultAddress   string
}

type evmAdminResolveFn func(ctx context.Context, sourceChainInput, destChainInput string) (*evmAdminContext, error)
type evmAdminGetAdapterFn func(ctx context.Context, sourceChainID uuid.UUID, routerAddress, destCAIP2 string, bridgeType uint8) (string, error)
type evmAdminSendTxFn func(ctx context.Context, sourceChainID uuid.UUID, contractAddress string, parsedABI abi.ABI, method string, args ...interface{}) (string, error)
type evmAdminResolveABIFn func(ctx context.Context, sourceChainID uuid.UUID, contractType entities.SmartContractType) (abi.ABI, error)
type evmAdminReadViewFn func(ctx context.Context, sourceChainID uuid.UUID, contractAddress string, parsedABI abi.ABI, method string, args ...interface{}) ([]interface{}, error)

type evmAdminOpsService struct {
	resolveContext evmAdminResolveFn
	getAdapter     evmAdminGetAdapterFn
	sendTx         evmAdminSendTxFn
	resolveABI     evmAdminResolveABIFn
	readView       evmAdminReadViewFn
}

func newEVMAdminOpsService(
	resolveFn evmAdminResolveFn,
	getAdapterFn evmAdminGetAdapterFn,
	sendTxFn evmAdminSendTxFn,
	resolveABIFn evmAdminResolveABIFn,
	readFns ...evmAdminReadViewFn,
) *evmAdminOpsService {
	var readFn evmAdminReadViewFn
	if len(readFns) > 0 {
		readFn = readFns[0]
	}
	return &evmAdminOpsService{
		resolveContext: resolveFn,
		getAdapter:     getAdapterFn,
		sendTx:         sendTxFn,
		resolveABI:     resolveABIFn,
		readView:       readFn,
	}
}

type StargateE2EStepStatus string

const (
	StargateStepSuccess StargateE2EStepStatus = "SUCCESS"
	StargateStepSkipped StargateE2EStepStatus = "SKIPPED"
)

type StargateE2EStepResult struct {
	Name   string                 `json:"name"`
	Status StargateE2EStepStatus `json:"status"`
	TxHash string                 `json:"txHash,omitempty"`
	Reason string                 `json:"reason,omitempty"`
}

type StargateConfigureSourceInput struct {
	RegisterAdapterIfMissing bool   `json:"registerAdapterIfMissing"`
	SetDefaultBridgeType     bool   `json:"setDefaultBridgeType"`
	SenderAddress            string `json:"senderAddress"`
	DstEID                   uint32 `json:"dstEid"`
	DstPeerHex               string `json:"dstPeerHex"`
	OptionsHex               string `json:"optionsHex"`
	ComposeGasLimit          uint64 `json:"composeGasLimit"`
	RegisterDelegate         bool   `json:"registerDelegate"`
	AuthorizeVaultSpender    bool   `json:"authorizeVaultSpender"`
}

type StargateConfigureDestinationInput struct {
	ChainID                 uuid.UUID `json:"chainId"`
	ReceiverAddress         string    `json:"receiverAddress"`
	SrcEID                  uint32    `json:"srcEid"`
	SrcSenderHex            string    `json:"srcSenderHex"`
	VaultAddress            string    `json:"vaultAddress"`
	GatewayAddress          string    `json:"gatewayAddress"`
	AuthorizeVaultSpender   bool      `json:"authorizeVaultSpender"`
	AuthorizeGatewayAdapter bool      `json:"authorizeGatewayAdapter"`
}

type StargateE2EConfigureInput struct {
	SourceChainInput string                             `json:"sourceChainId"`
	DestChainInput   string                             `json:"destChainId"`
	Source           StargateConfigureSourceInput      `json:"source"`
	Destination      StargateConfigureDestinationInput `json:"destination"`
}

type StargateE2EConfigureResult struct {
	Status string                   `json:"status"`
	Steps  []StargateE2EStepResult `json:"steps"`
}

type StargateE2EStatusInput struct {
	SourceChainInput           string    `json:"sourceChainId"`
	DestChainInput             string    `json:"destChainId"`
	DestinationChainID         uuid.UUID `json:"destinationChainId"`
	DestinationReceiverAddress string    `json:"destinationReceiverAddress"`
	DestinationSrcEID          uint32    `json:"destinationSrcEid"`
	DestinationSrcSenderHex    string    `json:"destinationSrcSenderHex"`
	DestinationVaultAddress    string    `json:"destinationVaultAddress"`
	DestinationGatewayAddress  string    `json:"destinationGatewayAddress"`
}

type StargateE2EStatusChecks struct {
	SourceAdapterRegistered      bool `json:"sourceAdapterRegistered"`
	SourceDefaultBridgeStargate bool `json:"sourceDefaultBridgeStargate"`
	SourceRouteConfigured        bool `json:"sourceRouteConfigured"`
	SourcePeerMatched            bool `json:"sourcePeerMatched"`
	SourceSenderVaultAuthorized  bool `json:"sourceSenderVaultAuthorized"`
	DestinationPeerConfigured    bool `json:"destinationPeerConfigured"`
	DestinationPeerTrusted       bool `json:"destinationPeerTrusted"`
	DestinationVaultAuthorized   bool `json:"destinationVaultAuthorized"`
	DestinationGatewayAuthorized bool `json:"destinationGatewayAuthorized"`
}

type StargateE2EStatusResult struct {
	Ready  bool                     `json:"ready"`
	Checks StargateE2EStatusChecks `json:"checks"`
	Issues []string                 `json:"issues"`
}

type CCIPConfigInput struct {
	SourceChainInput        string  `json:"sourceChainId"`
	DestChainInput          string  `json:"destChainId"`
	ChainSelector           *uint64 `json:"chainSelector"`
	DestinationAdapterHex   string  `json:"destinationAdapterHex"`
	DestinationGasLimit     *uint64 `json:"destinationGasLimit"`
	DestinationExtraArgsHex string  `json:"destinationExtraArgsHex"`
	DestinationFeeToken     string  `json:"destinationFeeTokenAddress"`
	DestinationReceiver     string  `json:"destinationReceiverAddress"`
	SourceChainSelector     *uint64 `json:"sourceChainSelector"`
	TrustedSenderHex        string  `json:"trustedSenderHex"`
	AllowSourceChain        *bool   `json:"allowSourceChain"`
}

type HyperbridgeTokenGatewayConfigInput struct {
	SourceChainInput     string  `json:"sourceChainId"`
	DestChainInput       string  `json:"destChainId"`
	StateMachineIDHex    string  `json:"stateMachineIdHex"`
	SettlementExecutor   string  `json:"settlementExecutorAddress"`
	NativeCost           *string `json:"nativeCost"`
	RelayerFee           *string `json:"relayerFee"`
}

func (s *evmAdminOpsService) RegisterAdapter(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	bridgeType uint8,
	adapterAddress string,
) (string, error) {
	if !common.IsHexAddress(adapterAddress) || common.HexToAddress(adapterAddress) == (common.Address{}) {
		return "", domainerrors.BadRequest("invalid adapterAddress")
	}

	resolved, err := s.resolveContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return "", err
	}

	parsedABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeRouter)
	if err != nil {
		return "", err
	}

	return s.sendTx(
		ctx,
		resolved.sourceChainID,
		resolved.routerAddress,
		parsedABI,
		"registerAdapter",
		resolved.destCAIP2,
		bridgeType,
		common.HexToAddress(adapterAddress),
	)
}

func (s *evmAdminOpsService) SetDefaultBridgeType(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	bridgeType uint8,
) (string, error) {
	resolved, err := s.resolveContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return "", err
	}

	parsedABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeGateway)
	if err != nil {
		return "", err
	}

	return s.sendTx(
		ctx,
		resolved.sourceChainID,
		resolved.gatewayAddress,
		parsedABI,
		"setDefaultBridgeType",
		resolved.destCAIP2,
		bridgeType,
	)
}

func (s *evmAdminOpsService) SetHyperbridgeConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	stateMachineIDHex, destinationContractHex string,
) (string, []string, error) {
	resolved, err := s.resolveContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return "", nil, err
	}
	adapter, err := s.getAdapter(ctx, resolved.sourceChainID, resolved.routerAddress, resolved.destCAIP2, 0)
	if err != nil {
		return "", nil, err
	}
	if !isValidAdapterAddress(adapter) {
		return "", nil, domainerrors.BadRequest("hyperbridge adapter (type 0) is not registered")
	}

	parsedABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeAdapterHyperbridge)
	if err != nil {
		return "", nil, err
	}

	var txHashes []string
	target := normalizeHexInput(stateMachineIDHex)
	if target != "" {
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, parsedABI, "setStateMachineId", resolved.destCAIP2, common.FromHex("0x"+target))
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setStateMachineId", txErr)
		}
		txHashes = append(txHashes, txHash)
	}

	dest := normalizeHexInput(destinationContractHex)
	if dest != "" {
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, parsedABI, "setDestinationContract", resolved.destCAIP2, common.FromHex("0x"+dest))
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setDestinationContract", txErr)
		}
		txHashes = append(txHashes, txHash)
	}
	if len(txHashes) == 0 {
		return "", nil, domainerrors.BadRequest("stateMachineId or destinationContract is required")
	}
	return adapter, txHashes, nil
}

func (s *evmAdminOpsService) SetHyperbridgeTokenGatewayConfig(
	ctx context.Context,
	input HyperbridgeTokenGatewayConfigInput,
) (string, []string, error) {
	resolved, err := s.resolveContext(ctx, input.SourceChainInput, input.DestChainInput)
	if err != nil {
		return "", nil, err
	}
	adapter, err := s.getAdapter(ctx, resolved.sourceChainID, resolved.routerAddress, resolved.destCAIP2, 3)
	if err != nil {
		return "", nil, err
	}
	if !isValidAdapterAddress(adapter) {
		return "", nil, domainerrors.BadRequest("hyperbridge token gateway adapter (type 3) is not registered")
	}

	parsedABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeAdapterHyperbridge)
	if err != nil {
		return "", nil, err
	}

	txHashes := make([]string, 0, 4)

	stateMachine := normalizeHexInput(input.StateMachineIDHex)
	if stateMachine != "" {
		txHash, txErr := s.sendTx(
			ctx,
			resolved.sourceChainID,
			adapter,
			parsedABI,
			"setStateMachineId",
			resolved.destCAIP2,
			common.FromHex("0x"+stateMachine),
		)
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setStateMachineId", txErr)
		}
		txHashes = append(txHashes, txHash)
	}

	settlementExecutor := strings.TrimSpace(input.SettlementExecutor)
	if settlementExecutor != "" {
		if !isValidAdapterAddress(settlementExecutor) {
			return "", nil, domainerrors.BadRequest("invalid settlementExecutorAddress")
		}
		txHash, txErr := s.sendTx(
			ctx,
			resolved.sourceChainID,
			adapter,
			parsedABI,
			"setRouteSettlementExecutor",
			resolved.destCAIP2,
			common.HexToAddress(settlementExecutor),
		)
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setRouteSettlementExecutor", txErr)
		}
		txHashes = append(txHashes, txHash)
	}

	if nativeCost, parseErr := parseOptionalBigInt(input.NativeCost); parseErr != nil {
		return "", nil, domainerrors.BadRequest("invalid nativeCost")
	} else if nativeCost != nil {
		txHash, txErr := s.sendTx(
			ctx,
			resolved.sourceChainID,
			adapter,
			parsedABI,
			"setNativeCost",
			resolved.destCAIP2,
			nativeCost,
		)
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setNativeCost", txErr)
		}
		txHashes = append(txHashes, txHash)
	}

	if relayerFee, parseErr := parseOptionalBigInt(input.RelayerFee); parseErr != nil {
		return "", nil, domainerrors.BadRequest("invalid relayerFee")
	} else if relayerFee != nil {
		txHash, txErr := s.sendTx(
			ctx,
			resolved.sourceChainID,
			adapter,
			parsedABI,
			"setRelayerFee",
			resolved.destCAIP2,
			relayerFee,
		)
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setRelayerFee", txErr)
		}
		txHashes = append(txHashes, txHash)
	}

	if len(txHashes) == 0 {
		return "", nil, domainerrors.BadRequest("at least one config field is required")
	}

	return adapter, txHashes, nil
}

func (s *evmAdminOpsService) SetCCIPConfig(ctx context.Context, input CCIPConfigInput) (string, []string, error) {
	resolved, err := s.resolveContext(ctx, input.SourceChainInput, input.DestChainInput)
	if err != nil {
		return "", nil, err
	}
	adapter, err := s.getAdapter(ctx, resolved.sourceChainID, resolved.routerAddress, resolved.destCAIP2, 1)
	if err != nil {
		return "", nil, err
	}
	if !isValidAdapterAddress(adapter) {
		return "", nil, domainerrors.BadRequest("ccip adapter (type 1) is not registered")
	}

	parsedABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeAdapterCCIP)
	if err != nil {
		return "", nil, err
	}

	var txHashes []string
	dest := normalizeHexInput(input.DestinationAdapterHex)
	_, hasSetChainConfig := parsedABI.Methods["setChainConfig"]

	// Prefer modern single-call setter when available and both selector+destination are provided.
	usedSetChainConfig := false
	if input.ChainSelector != nil && dest != "" && hasSetChainConfig {
		if destAddress, parseErr := parseAdapterAddressHex("0x" + dest); parseErr == nil {
			txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, parsedABI, "setChainConfig", resolved.destCAIP2, *input.ChainSelector, destAddress)
			if txErr != nil {
				return "", txHashes, wrapAdminTxError("setChainConfig", txErr)
			}
			txHashes = append(txHashes, txHash)
			usedSetChainConfig = true
		}
	}
	if !usedSetChainConfig {
		if input.ChainSelector != nil {
			txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, parsedABI, "setChainSelector", resolved.destCAIP2, *input.ChainSelector)
			if txErr != nil {
				return "", txHashes, wrapAdminTxError("setChainSelector", txErr)
			}
			txHashes = append(txHashes, txHash)
		}
		if dest != "" {
			txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, parsedABI, "setDestinationAdapter", resolved.destCAIP2, common.FromHex("0x"+dest))
			if txErr != nil {
				return "", txHashes, wrapAdminTxError("setDestinationAdapter", txErr)
			}
			txHashes = append(txHashes, txHash)
		}
	}
	if input.DestinationGasLimit != nil {
		gasLimit := new(big.Int).SetUint64(*input.DestinationGasLimit)
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, parsedABI, "setDestinationGasLimit", resolved.destCAIP2, gasLimit)
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setDestinationGasLimit", txErr)
		}
		txHashes = append(txHashes, txHash)
	}
	extra := normalizeHexInput(input.DestinationExtraArgsHex)
	if extra != "" {
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, parsedABI, "setDestinationExtraArgs", resolved.destCAIP2, common.FromHex("0x"+extra))
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setDestinationExtraArgs", txErr)
		}
		txHashes = append(txHashes, txHash)
	}
	if strings.TrimSpace(input.DestinationFeeToken) != "" {
		if !common.IsHexAddress(input.DestinationFeeToken) {
			return "", nil, domainerrors.BadRequest("invalid destinationFeeTokenAddress")
		}
		txHash, txErr := s.sendTx(
			ctx,
			resolved.sourceChainID,
			adapter,
			parsedABI,
			"setDestinationFeeToken",
			resolved.destCAIP2,
			common.HexToAddress(input.DestinationFeeToken),
		)
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setDestinationFeeToken", txErr)
		}
		txHashes = append(txHashes, txHash)
	}

	shouldConfigureReceiver := strings.TrimSpace(input.DestinationReceiver) != "" ||
		input.SourceChainSelector != nil ||
		strings.TrimSpace(input.TrustedSenderHex) != "" ||
		input.AllowSourceChain != nil
	if shouldConfigureReceiver {
		if strings.TrimSpace(input.DestinationReceiver) == "" {
			return "", nil, domainerrors.BadRequest("destinationReceiverAddress is required for receiver trust configuration")
		}
		if !common.IsHexAddress(input.DestinationReceiver) || common.HexToAddress(input.DestinationReceiver) == (common.Address{}) {
			return "", nil, domainerrors.BadRequest("invalid destinationReceiverAddress")
		}
		if input.SourceChainSelector == nil {
			return "", nil, domainerrors.BadRequest("sourceChainSelector is required for receiver trust configuration")
		}

		destinationCtx, resolveErr := s.resolveContext(ctx, input.DestChainInput, input.SourceChainInput)
		if resolveErr != nil {
			return "", nil, resolveErr
		}
		receiverABI := FallbackCCIPReceiverAdminABI

		trustedSender := normalizeHexInput(input.TrustedSenderHex)
		if trustedSender != "" {
			txHash, txErr := s.sendTx(
				ctx,
				destinationCtx.sourceChainID,
				input.DestinationReceiver,
				receiverABI,
				"setTrustedSender",
				*input.SourceChainSelector,
				common.FromHex("0x"+trustedSender),
			)
			if txErr != nil {
				return "", txHashes, wrapAdminTxError("setTrustedSender", txErr)
			}
			txHashes = append(txHashes, txHash)
		}
		if input.AllowSourceChain != nil {
			txHash, txErr := s.sendTx(
				ctx,
				destinationCtx.sourceChainID,
				input.DestinationReceiver,
				receiverABI,
				"setSourceChainAllowed",
				*input.SourceChainSelector,
				*input.AllowSourceChain,
			)
			if txErr != nil {
				return "", txHashes, wrapAdminTxError("setSourceChainAllowed", txErr)
			}
			txHashes = append(txHashes, txHash)
		}
	}
	if len(txHashes) == 0 {
		return "", nil, domainerrors.BadRequest(
			"at least one config field is required (chainSelector, destinationAdapter, destinationGasLimit, destinationExtraArgsHex, destinationFeeTokenAddress, receiver trust fields)",
		)
	}
	return adapter, txHashes, nil
}

func (s *evmAdminOpsService) SetStargateConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	dstEid *uint32, peerHex, optionsHex string,
) (string, []string, error) {
	resolved, err := s.resolveContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return "", nil, err
	}
	adapter, err := s.getAdapter(ctx, resolved.sourceChainID, resolved.routerAddress, resolved.destCAIP2, 2)
	if err != nil {
		return "", nil, err
	}
	if !isValidAdapterAddress(adapter) {
		return "", nil, domainerrors.BadRequest("stargate adapter (type 2) is not registered")
	}

	parsedABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeAdapterStargate)
	if err != nil {
		return "", nil, err
	}

	var txHashes []string
	trimmedPeer := strings.TrimSpace(peerHex)
	if dstEid != nil || trimmedPeer != "" {
		if dstEid == nil || trimmedPeer == "" {
			return "", nil, domainerrors.BadRequest("dstEid and peerHex are required to set route")
		}
		peer32, parseErr := parseHexToBytes32(trimmedPeer)
		if parseErr != nil {
			return "", nil, domainerrors.BadRequest("invalid peerHex")
		}
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, parsedABI, "setRoute", resolved.destCAIP2, *dstEid, peer32)
		if txErr != nil {
			return "", txHashes, wrapAdminTxError("setRoute", txErr)
		}
		txHashes = append(txHashes, txHash)
	}

	trimmedOptions := strings.TrimSpace(optionsHex)
	if trimmedOptions != "" {
		if !strings.HasPrefix(trimmedOptions, "0x") {
			trimmedOptions = "0x" + trimmedOptions
		}
		optionsMethod := stargateOptionsSetterMethod(parsedABI)
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, parsedABI, optionsMethod, resolved.destCAIP2, common.FromHex(trimmedOptions))
		if txErr != nil {
			return "", txHashes, wrapAdminTxError(optionsMethod, txErr)
		}
		txHashes = append(txHashes, txHash)
	}

	if len(txHashes) == 0 {
		return "", nil, domainerrors.BadRequest("dstEid+peerHex or optionsHex is required")
	}
	return adapter, txHashes, nil
}

func (s *evmAdminOpsService) ConfigureStargateE2E(
	ctx context.Context,
	input StargateE2EConfigureInput,
) (*StargateE2EConfigureResult, error) {
	resolved, err := s.resolveContext(ctx, input.SourceChainInput, input.DestChainInput)
	if err != nil {
		return nil, err
	}

	routerABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeRouter)
	if err != nil {
		return nil, err
	}
	gatewayABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeGateway)
	if err != nil {
		return nil, err
	}
	senderABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeAdapterStargate)
	if err != nil {
		return nil, err
	}

	var sourceVaultABI abi.ABI
	if resolved.vaultAddress != "" {
		sourceVaultABI, err = s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeVault)
		if err != nil {
			return nil, err
		}
	}

	var destinationReceiverABI, destinationVaultABI, destinationGatewayABI abi.ABI
	if input.Destination.ChainID != uuid.Nil {
		destinationReceiverABI, err = s.resolveABI(ctx, input.Destination.ChainID, entities.ContractTypeReceiverStargate)
		if err != nil {
			return nil, err
		}
		if input.Destination.AuthorizeVaultSpender && input.Destination.VaultAddress != "" {
			destinationVaultABI, err = s.resolveABI(ctx, input.Destination.ChainID, entities.ContractTypeVault)
			if err != nil {
				return nil, err
			}
		}
		if input.Destination.AuthorizeGatewayAdapter && input.Destination.GatewayAddress != "" {
			destinationGatewayABI, err = s.resolveABI(ctx, input.Destination.ChainID, entities.ContractTypeGateway)
			if err != nil {
				return nil, err
			}
		}
	}

	steps := make([]StargateE2EStepResult, 0, 12)
	addSuccess := func(name, txHash string) {
		steps = append(steps, StargateE2EStepResult{Name: name, Status: StargateStepSuccess, TxHash: txHash})
	}
	addSkipped := func(name, reason string) {
		steps = append(steps, StargateE2EStepResult{Name: name, Status: StargateStepSkipped, Reason: reason})
	}

	currentSender, err := s.getAdapter(ctx, resolved.sourceChainID, resolved.routerAddress, resolved.destCAIP2, 2)
	if err != nil {
		return nil, err
	}

	sourceSender := strings.TrimSpace(input.Source.SenderAddress)
	if sourceSender != "" && (!common.IsHexAddress(sourceSender) || common.HexToAddress(sourceSender) == (common.Address{})) {
		return nil, domainerrors.BadRequest("invalid source sender address")
	}

	if sourceSender == "" && isValidAdapterAddress(currentSender) {
		sourceSender = currentSender
	}

	if !isValidAdapterAddress(currentSender) {
		if !input.Source.RegisterAdapterIfMissing {
			return nil, domainerrors.BadRequest("stargate adapter (type 2) is not registered and registerAdapterIfMissing is false")
		}
		if !isValidAdapterAddress(sourceSender) {
			return nil, domainerrors.BadRequest("senderAddress is required when adapter type 2 is missing")
		}
		txHash, txErr := s.sendTx(
			ctx,
			resolved.sourceChainID,
			resolved.routerAddress,
			routerABI,
			"registerAdapter",
			resolved.destCAIP2,
			uint8(2),
			common.HexToAddress(sourceSender),
		)
		if txErr != nil {
			return nil, wrapAdminTxError("registerAdapter", txErr)
		}
		addSuccess("registerAdapter", txHash)
	} else if input.Source.RegisterAdapterIfMissing && isValidAdapterAddress(sourceSender) && !strings.EqualFold(currentSender, sourceSender) {
		txHash, txErr := s.sendTx(
			ctx,
			resolved.sourceChainID,
			resolved.routerAddress,
			routerABI,
			"registerAdapter",
			resolved.destCAIP2,
			uint8(2),
			common.HexToAddress(sourceSender),
		)
		if txErr != nil {
			return nil, wrapAdminTxError("registerAdapter", txErr)
		}
		addSuccess("registerAdapter", txHash)
	} else {
		addSkipped("registerAdapter", "already-configured")
	}

	if !isValidAdapterAddress(sourceSender) {
		return nil, domainerrors.BadRequest("cannot resolve active stargate sender adapter")
	}

	if input.Source.SetDefaultBridgeType {
		defaultBridge, readErr := s.readUint8(ctx, resolved.sourceChainID, resolved.gatewayAddress, gatewayABI, "defaultBridgeTypes", resolved.destCAIP2)
		if readErr == nil && defaultBridge == 2 {
			addSkipped("setDefaultBridgeType", "already-configured")
		} else {
			txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, resolved.gatewayAddress, gatewayABI, "setDefaultBridgeType", resolved.destCAIP2, uint8(2))
			if txErr != nil {
				return nil, wrapAdminTxError("setDefaultBridgeType", txErr)
			}
			addSuccess("setDefaultBridgeType", txHash)
		}
	}

	if input.Source.DstEID == 0 {
		return nil, domainerrors.BadRequest("source dstEid is required")
	}
	dstPeer, parsePeerErr := parseHexToBytes32(input.Source.DstPeerHex)
	if parsePeerErr != nil {
		return nil, domainerrors.BadRequest("invalid source dstPeerHex")
	}

	currentDstEID, dstReadErr := s.readUint32(ctx, resolved.sourceChainID, sourceSender, senderABI, "dstEids", resolved.destCAIP2)
	currentPeer, peerReadErr := s.readBytes32(ctx, resolved.sourceChainID, sourceSender, senderABI, "peers", resolved.destCAIP2)
	if dstReadErr == nil && peerReadErr == nil && currentDstEID == input.Source.DstEID && currentPeer == dstPeer {
		addSkipped("setRoute", "already-configured")
	} else {
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, sourceSender, senderABI, "setRoute", resolved.destCAIP2, input.Source.DstEID, dstPeer)
		if txErr != nil {
			return nil, wrapAdminTxError("setRoute", txErr)
		}
		addSuccess("setRoute", txHash)
	}

	options := strings.TrimSpace(input.Source.OptionsHex)
	if options != "" {
		if !strings.HasPrefix(options, "0x") {
			options = "0x" + options
		}
		optionsMethod := stargateOptionsSetterMethod(senderABI)
		optionsGetter := stargateOptionsGetterMethod(senderABI)
		currentOptions, optsReadErr := s.readBytes(ctx, resolved.sourceChainID, sourceSender, senderABI, optionsGetter, resolved.destCAIP2)
		if optsReadErr == nil && strings.EqualFold("0x"+common.Bytes2Hex(currentOptions), options) {
			addSkipped(optionsMethod, "already-configured")
		} else {
			txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, sourceSender, senderABI, optionsMethod, resolved.destCAIP2, common.FromHex(options))
			if txErr != nil {
				return nil, wrapAdminTxError(optionsMethod, txErr)
			}
			addSuccess(optionsMethod, txHash)
		}
	} else {
		addSkipped(stargateOptionsSetterMethod(senderABI), "no-options")
	}

	if input.Source.ComposeGasLimit > 0 {
		currentComposeGas, gasReadErr := s.readUint64Compatible(ctx, resolved.sourceChainID, sourceSender, senderABI, "destinationComposeGasLimits", resolved.destCAIP2)
		if gasReadErr == nil && currentComposeGas == input.Source.ComposeGasLimit {
			addSkipped("setDestinationComposeGasLimit", "already-configured")
		} else if _, ok := senderABI.Methods["setDestinationComposeGasLimit"]; ok {
			txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, sourceSender, senderABI, "setDestinationComposeGasLimit", resolved.destCAIP2, input.Source.ComposeGasLimit)
			if txErr != nil {
				return nil, wrapAdminTxError("setDestinationComposeGasLimit", txErr)
			}
			addSuccess("setDestinationComposeGasLimit", txHash)
		}
	}

	if input.Source.RegisterDelegate {
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, sourceSender, senderABI, "registerDelegate")
		if txErr != nil {
			return nil, wrapAdminTxError("registerDelegate", txErr)
		}
		addSuccess("registerDelegate", txHash)
	}

	if input.Source.AuthorizeVaultSpender {
		if resolved.vaultAddress == "" {
			addSkipped("sourceVault.setAuthorizedSpender", "source-vault-unavailable")
		} else {
			authorized, authReadErr := s.readBool(ctx, resolved.sourceChainID, resolved.vaultAddress, sourceVaultABI, "authorizedSpenders", common.HexToAddress(sourceSender))
			if authReadErr == nil && authorized {
				addSkipped("sourceVault.setAuthorizedSpender", "already-configured")
			} else {
				txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, resolved.vaultAddress, sourceVaultABI, "setAuthorizedSpender", common.HexToAddress(sourceSender), true)
				if txErr != nil {
					return nil, wrapAdminTxError("source setAuthorizedSpender", txErr)
				}
				addSuccess("sourceVault.setAuthorizedSpender", txHash)
			}
		}
	}

	if input.Destination.ReceiverAddress != "" || input.Destination.AuthorizeVaultSpender || input.Destination.AuthorizeGatewayAdapter {
		if input.Destination.ChainID == uuid.Nil {
			return nil, domainerrors.BadRequest("destination chainId is required for destination side configuration")
		}
		if !common.IsHexAddress(input.Destination.ReceiverAddress) || common.HexToAddress(input.Destination.ReceiverAddress) == (common.Address{}) {
			return nil, domainerrors.BadRequest("invalid destination receiver address")
		}
		if input.Destination.SrcEID == 0 {
			return nil, domainerrors.BadRequest("destination srcEid is required")
		}
		srcSender, parseSrcErr := parseHexToBytes32(input.Destination.SrcSenderHex)
		if parseSrcErr != nil {
			return nil, domainerrors.BadRequest("invalid destination srcSenderHex")
		}

		peerConfigured, trusted, _, pathReadErr := s.readPathState(ctx, input.Destination.ChainID, input.Destination.ReceiverAddress, destinationReceiverABI, input.Destination.SrcEID, srcSender)
		if pathReadErr == nil && peerConfigured && trusted {
			addSkipped("destinationReceiver.setPeer", "already-configured")
		} else {
			txHash, txErr := s.sendTx(ctx, input.Destination.ChainID, input.Destination.ReceiverAddress, destinationReceiverABI, "setPeer", input.Destination.SrcEID, srcSender)
			if txErr != nil {
				return nil, wrapAdminTxError("destination setPeer", txErr)
			}
			addSuccess("destinationReceiver.setPeer", txHash)
		}

		if input.Destination.AuthorizeVaultSpender {
			if !common.IsHexAddress(input.Destination.VaultAddress) || common.HexToAddress(input.Destination.VaultAddress) == (common.Address{}) {
				return nil, domainerrors.BadRequest("invalid destination vault address")
			}
			authorized, authReadErr := s.readBool(
				ctx,
				input.Destination.ChainID,
				input.Destination.VaultAddress,
				destinationVaultABI,
				"authorizedSpenders",
				common.HexToAddress(input.Destination.ReceiverAddress),
			)
			if authReadErr == nil && authorized {
				addSkipped("destinationVault.setAuthorizedSpender", "already-configured")
			} else {
				txHash, txErr := s.sendTx(
					ctx,
					input.Destination.ChainID,
					input.Destination.VaultAddress,
					destinationVaultABI,
					"setAuthorizedSpender",
					common.HexToAddress(input.Destination.ReceiverAddress),
					true,
				)
				if txErr != nil {
					return nil, wrapAdminTxError("destination setAuthorizedSpender", txErr)
				}
				addSuccess("destinationVault.setAuthorizedSpender", txHash)
			}
		}

		if input.Destination.AuthorizeGatewayAdapter {
			if !common.IsHexAddress(input.Destination.GatewayAddress) || common.HexToAddress(input.Destination.GatewayAddress) == (common.Address{}) {
				return nil, domainerrors.BadRequest("invalid destination gateway address")
			}
			authorized, authReadErr := s.readBool(
				ctx,
				input.Destination.ChainID,
				input.Destination.GatewayAddress,
				destinationGatewayABI,
				"isAuthorizedAdapter",
				common.HexToAddress(input.Destination.ReceiverAddress),
			)
			if authReadErr == nil && authorized {
				addSkipped("destinationGateway.setAuthorizedAdapter", "already-configured")
			} else {
				txHash, txErr := s.sendTx(
					ctx,
					input.Destination.ChainID,
					input.Destination.GatewayAddress,
					destinationGatewayABI,
					"setAuthorizedAdapter",
					common.HexToAddress(input.Destination.ReceiverAddress),
					true,
				)
				if txErr != nil {
					return nil, wrapAdminTxError("destination setAuthorizedAdapter", txErr)
				}
				addSuccess("destinationGateway.setAuthorizedAdapter", txHash)
			}
		}
	}

	return &StargateE2EConfigureResult{
		Status: "SUCCESS",
		Steps:  steps,
	}, nil
}

func (s *evmAdminOpsService) GetStargateE2EStatus(
	ctx context.Context,
	input StargateE2EStatusInput,
) (*StargateE2EStatusResult, error) {
	resolved, err := s.resolveContext(ctx, input.SourceChainInput, input.DestChainInput)
	if err != nil {
		return nil, err
	}
	result := &StargateE2EStatusResult{
		Checks: StargateE2EStatusChecks{},
		Issues: make([]string, 0, 8),
	}

	gatewayABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeGateway)
	if err != nil {
		return nil, err
	}
	senderABI, err := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeAdapterStargate)
	if err != nil {
		return nil, err
	}

	senderAddress, adapterErr := s.getAdapter(ctx, resolved.sourceChainID, resolved.routerAddress, resolved.destCAIP2, 2)
	if adapterErr == nil && isValidAdapterAddress(senderAddress) {
		result.Checks.SourceAdapterRegistered = true
	} else {
		result.Issues = append(result.Issues, "source adapter stargate (type 2) is not registered")
	}

	defaultBridge, defaultErr := s.readUint8(ctx, resolved.sourceChainID, resolved.gatewayAddress, gatewayABI, "defaultBridgeTypes", resolved.destCAIP2)
	if defaultErr == nil && defaultBridge == 2 {
		result.Checks.SourceDefaultBridgeStargate = true
	} else {
		result.Issues = append(result.Issues, "source gateway default bridge is not Stargate (2)")
	}

	if isValidAdapterAddress(senderAddress) {
		configured, cfgErr := s.readBool(ctx, resolved.sourceChainID, senderAddress, senderABI, "isRouteConfigured", resolved.destCAIP2)
		if cfgErr == nil && configured {
			result.Checks.SourceRouteConfigured = true
		} else {
			result.Issues = append(result.Issues, "source sender route is not configured")
		}

		dstEid, dstErr := s.readUint32(ctx, resolved.sourceChainID, senderAddress, senderABI, "dstEids", resolved.destCAIP2)
		dstPeer, peerErr := s.readBytes32(ctx, resolved.sourceChainID, senderAddress, senderABI, "peers", resolved.destCAIP2)
		if dstErr == nil && peerErr == nil && dstEid != 0 && dstPeer != ([32]byte{}) {
			result.Checks.SourcePeerMatched = true
		} else {
			result.Issues = append(result.Issues, "source sender peer/dstEid is not fully configured")
		}

		if resolved.vaultAddress != "" {
			vaultABI, vaultErr := s.resolveABI(ctx, resolved.sourceChainID, entities.ContractTypeVault)
			if vaultErr == nil {
				auth, readErr := s.readBool(ctx, resolved.sourceChainID, resolved.vaultAddress, vaultABI, "authorizedSpenders", common.HexToAddress(senderAddress))
				if readErr == nil && auth {
					result.Checks.SourceSenderVaultAuthorized = true
				}
			}
		}
	}
	if !result.Checks.SourceSenderVaultAuthorized {
		result.Issues = append(result.Issues, "source vault sender authorization is missing")
	}

	if input.DestinationChainID != uuid.Nil &&
		common.IsHexAddress(input.DestinationReceiverAddress) &&
		input.DestinationSrcEID != 0 &&
		strings.TrimSpace(input.DestinationSrcSenderHex) != "" {
		receiverABI, receiverErr := s.resolveABI(ctx, input.DestinationChainID, entities.ContractTypeReceiverStargate)
		if receiverErr == nil {
			srcSender, parseErr := parseHexToBytes32(input.DestinationSrcSenderHex)
			if parseErr == nil {
				peerConfigured, trusted, _, pathErr := s.readPathState(ctx, input.DestinationChainID, input.DestinationReceiverAddress, receiverABI, input.DestinationSrcEID, srcSender)
				if pathErr == nil && peerConfigured {
					result.Checks.DestinationPeerConfigured = true
				}
				if pathErr == nil && trusted {
					result.Checks.DestinationPeerTrusted = true
				}
			}
		}
	}
	if !result.Checks.DestinationPeerConfigured {
		result.Issues = append(result.Issues, "destination receiver peer is not configured")
	}
	if !result.Checks.DestinationPeerTrusted {
		result.Issues = append(result.Issues, "destination receiver path is not trusted for source sender")
	}

	if input.DestinationChainID != uuid.Nil && common.IsHexAddress(input.DestinationVaultAddress) && common.IsHexAddress(input.DestinationReceiverAddress) {
		vaultABI, vaultErr := s.resolveABI(ctx, input.DestinationChainID, entities.ContractTypeVault)
		if vaultErr == nil {
			auth, readErr := s.readBool(ctx, input.DestinationChainID, input.DestinationVaultAddress, vaultABI, "authorizedSpenders", common.HexToAddress(input.DestinationReceiverAddress))
			if readErr == nil && auth {
				result.Checks.DestinationVaultAuthorized = true
			}
		}
	}
	if !result.Checks.DestinationVaultAuthorized {
		result.Issues = append(result.Issues, "destination vault receiver authorization is missing")
	}

	if input.DestinationChainID != uuid.Nil && common.IsHexAddress(input.DestinationGatewayAddress) && common.IsHexAddress(input.DestinationReceiverAddress) {
		dstGatewayABI, gwErr := s.resolveABI(ctx, input.DestinationChainID, entities.ContractTypeGateway)
		if gwErr == nil {
			auth, readErr := s.readBool(ctx, input.DestinationChainID, input.DestinationGatewayAddress, dstGatewayABI, "isAuthorizedAdapter", common.HexToAddress(input.DestinationReceiverAddress))
			if readErr == nil && auth {
				result.Checks.DestinationGatewayAuthorized = true
			}
		}
	}
	if !result.Checks.DestinationGatewayAuthorized {
		result.Issues = append(result.Issues, "destination gateway receiver authorization is missing")
	}

	result.Ready = len(result.Issues) == 0
	return result, nil
}

func (s *evmAdminOpsService) readBool(
	ctx context.Context,
	chainID uuid.UUID,
	contractAddress string,
	parsedABI abi.ABI,
	method string,
	args ...interface{},
) (bool, error) {
	values, err := s.readValues(ctx, chainID, contractAddress, parsedABI, method, args...)
	if err != nil {
		return false, err
	}
	value, ok := values[0].(bool)
	if !ok {
		return false, fmt.Errorf("invalid %s return type", method)
	}
	return value, nil
}

func (s *evmAdminOpsService) readUint8(
	ctx context.Context,
	chainID uuid.UUID,
	contractAddress string,
	parsedABI abi.ABI,
	method string,
	args ...interface{},
) (uint8, error) {
	values, err := s.readValues(ctx, chainID, contractAddress, parsedABI, method, args...)
	if err != nil {
		return 0, err
	}
	value, ok := values[0].(uint8)
	if !ok {
		return 0, fmt.Errorf("invalid %s return type", method)
	}
	return value, nil
}

func (s *evmAdminOpsService) readUint32(
	ctx context.Context,
	chainID uuid.UUID,
	contractAddress string,
	parsedABI abi.ABI,
	method string,
	args ...interface{},
) (uint32, error) {
	values, err := s.readValues(ctx, chainID, contractAddress, parsedABI, method, args...)
	if err != nil {
		return 0, err
	}
	value, ok := values[0].(uint32)
	if !ok {
		return 0, fmt.Errorf("invalid %s return type", method)
	}
	return value, nil
}

func (s *evmAdminOpsService) readBytes32(
	ctx context.Context,
	chainID uuid.UUID,
	contractAddress string,
	parsedABI abi.ABI,
	method string,
	args ...interface{},
) ([32]byte, error) {
	var zero [32]byte
	values, err := s.readValues(ctx, chainID, contractAddress, parsedABI, method, args...)
	if err != nil {
		return zero, err
	}
	value, ok := values[0].([32]byte)
	if !ok {
		return zero, fmt.Errorf("invalid %s return type", method)
	}
	return value, nil
}

func (s *evmAdminOpsService) readBytes(
	ctx context.Context,
	chainID uuid.UUID,
	contractAddress string,
	parsedABI abi.ABI,
	method string,
	args ...interface{},
) ([]byte, error) {
	values, err := s.readValues(ctx, chainID, contractAddress, parsedABI, method, args...)
	if err != nil {
		return nil, err
	}
	value, ok := values[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid %s return type", method)
	}
	return value, nil
}

func (s *evmAdminOpsService) readUint64Compatible(
	ctx context.Context,
	chainID uuid.UUID,
	contractAddress string,
	parsedABI abi.ABI,
	method string,
	args ...interface{},
) (uint64, error) {
	values, err := s.readValues(ctx, chainID, contractAddress, parsedABI, method, args...)
	if err != nil {
		return 0, err
	}
	switch value := values[0].(type) {
	case uint64:
		return value, nil
	case *big.Int:
		if value == nil || !value.IsUint64() {
			return 0, fmt.Errorf("invalid %s return type", method)
		}
		return value.Uint64(), nil
	default:
		return 0, fmt.Errorf("invalid %s return type", method)
	}
}

func stargateOptionsSetterMethod(parsedABI abi.ABI) string {
	if _, ok := parsedABI.Methods["setDestinationExtraOptions"]; ok {
		return "setDestinationExtraOptions"
	}
	return "setEnforcedOptions"
}

func stargateOptionsGetterMethod(parsedABI abi.ABI) string {
	if _, ok := parsedABI.Methods["destinationExtraOptions"]; ok {
		return "destinationExtraOptions"
	}
	return "enforcedOptions"
}

func (s *evmAdminOpsService) readPathState(
	ctx context.Context,
	chainID uuid.UUID,
	contractAddress string,
	parsedABI abi.ABI,
	srcEID uint32,
	sender [32]byte,
) (bool, bool, [32]byte, error) {
	var zero [32]byte
	values, err := s.readValues(ctx, chainID, contractAddress, parsedABI, "getPathState", srcEID, sender)
	if err == nil && len(values) >= 3 {
		peerConfigured, ok1 := values[0].(bool)
		trusted, ok2 := values[1].(bool)
		configuredPeer, ok3 := values[2].([32]byte)
		if ok1 && ok2 && ok3 {
			return peerConfigured, trusted, configuredPeer, nil
		}
	}
	peer, fallbackErr := s.readBytes32(ctx, chainID, contractAddress, parsedABI, "peers", srcEID)
	if fallbackErr != nil {
		return false, false, zero, fallbackErr
	}
	peerConfigured := peer != zero
	trusted := peerConfigured && peer == sender
	return peerConfigured, trusted, peer, nil
}

func (s *evmAdminOpsService) readValues(
	ctx context.Context,
	chainID uuid.UUID,
	contractAddress string,
	parsedABI abi.ABI,
	method string,
	args ...interface{},
) ([]interface{}, error) {
	if s.readView == nil {
		return nil, fmt.Errorf("readView is not configured")
	}
	return s.readView(ctx, chainID, contractAddress, parsedABI, method, args...)
}

func isValidAdapterAddress(adapter string) bool {
	return common.IsHexAddress(adapter) && common.HexToAddress(adapter) != (common.Address{})
}

func normalizeHexInput(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.TrimPrefix(trimmed, "0x")
}

func parseAdapterAddressHex(value string) (common.Address, error) {
	raw := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if raw == "" {
		return common.Address{}, fmt.Errorf("empty hex value")
	}
	b := common.FromHex("0x" + raw)
	if len(b) == 20 {
		addr := common.BytesToAddress(b)
		if addr == (common.Address{}) {
			return common.Address{}, fmt.Errorf("zero address")
		}
		return addr, nil
	}
	if len(b) == 32 {
		addr := common.BytesToAddress(b[12:])
		if addr == (common.Address{}) {
			return common.Address{}, fmt.Errorf("zero address")
		}
		return addr, nil
	}
	return common.Address{}, fmt.Errorf("invalid adapter hex length")
}

func parseOptionalBigInt(raw *string) (*big.Int, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	if len(value) > 2 && strings.EqualFold(value[:2], "0x") {
		n := new(big.Int)
		if _, ok := n.SetString(value[2:], 16); ok {
			return n, nil
		}
		return nil, fmt.Errorf("invalid hex integer")
	}
	n := new(big.Int)
	if _, ok := n.SetString(value, 10); ok {
		return n, nil
	}
	return nil, fmt.Errorf("invalid integer")
}

func wrapAdminTxError(method string, err error) error {
	if err == nil {
		return nil
	}
	return domainerrors.BadRequest(fmt.Sprintf("%s failed: %s", method, err.Error()))
}
