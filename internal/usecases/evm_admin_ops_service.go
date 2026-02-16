package usecases

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

type evmAdminContext struct {
	sourceChainID  uuid.UUID
	destCAIP2      string
	routerAddress  string
	gatewayAddress string
}

type evmAdminResolveFn func(ctx context.Context, sourceChainInput, destChainInput string) (*evmAdminContext, error)
type evmAdminGetAdapterFn func(ctx context.Context, sourceChainID uuid.UUID, routerAddress, destCAIP2 string, bridgeType uint8) (string, error)
type evmAdminSendTxFn func(ctx context.Context, sourceChainID uuid.UUID, contractAddress string, parsedABI abi.ABI, method string, args ...interface{}) (string, error)

type evmAdminOpsService struct {
	resolveContext evmAdminResolveFn
	getAdapter     evmAdminGetAdapterFn
	sendTx         evmAdminSendTxFn
}

func newEVMAdminOpsService(resolveFn evmAdminResolveFn, getAdapterFn evmAdminGetAdapterFn, sendTxFn evmAdminSendTxFn) *evmAdminOpsService {
	return &evmAdminOpsService{
		resolveContext: resolveFn,
		getAdapter:     getAdapterFn,
		sendTx:         sendTxFn,
	}
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

	return s.sendTx(
		ctx,
		resolved.sourceChainID,
		resolved.routerAddress,
		payChainRouterAdminABI,
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

	return s.sendTx(
		ctx,
		resolved.sourceChainID,
		resolved.gatewayAddress,
		payChainGatewayAdminABI,
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

	var txHashes []string
	target := normalizeHexInput(stateMachineIDHex)
	if target != "" {
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, hyperbridgeSenderAdminABI, "setStateMachineId", resolved.destCAIP2, common.FromHex("0x"+target))
		if txErr != nil {
			return "", txHashes, txErr
		}
		txHashes = append(txHashes, txHash)
	}

	dest := normalizeHexInput(destinationContractHex)
	if dest != "" {
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, hyperbridgeSenderAdminABI, "setDestinationContract", resolved.destCAIP2, common.FromHex("0x"+dest))
		if txErr != nil {
			return "", txHashes, txErr
		}
		txHashes = append(txHashes, txHash)
	}
	if len(txHashes) == 0 {
		return "", nil, domainerrors.BadRequest("stateMachineId or destinationContract is required")
	}
	return adapter, txHashes, nil
}

func (s *evmAdminOpsService) SetCCIPConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	chainSelector *uint64, destinationAdapterHex string,
) (string, []string, error) {
	resolved, err := s.resolveContext(ctx, sourceChainInput, destChainInput)
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

	var txHashes []string
	if chainSelector != nil {
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, ccipSenderAdminABI, "setChainSelector", resolved.destCAIP2, *chainSelector)
		if txErr != nil {
			return "", txHashes, txErr
		}
		txHashes = append(txHashes, txHash)
	}
	dest := normalizeHexInput(destinationAdapterHex)
	if dest != "" {
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, ccipSenderAdminABI, "setDestinationAdapter", resolved.destCAIP2, common.FromHex("0x"+dest))
		if txErr != nil {
			return "", txHashes, txErr
		}
		txHashes = append(txHashes, txHash)
	}
	if len(txHashes) == 0 {
		return "", nil, domainerrors.BadRequest("chainSelector or destinationAdapter is required")
	}
	return adapter, txHashes, nil
}

func (s *evmAdminOpsService) SetLayerZeroConfig(
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
		return "", nil, domainerrors.BadRequest("layerzero adapter (type 2) is not registered")
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
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, layerZeroSenderAdminABI, "setRoute", resolved.destCAIP2, *dstEid, peer32)
		if txErr != nil {
			return "", txHashes, txErr
		}
		txHashes = append(txHashes, txHash)
	}

	trimmedOptions := strings.TrimSpace(optionsHex)
	if trimmedOptions != "" {
		if !strings.HasPrefix(trimmedOptions, "0x") {
			trimmedOptions = "0x" + trimmedOptions
		}
		txHash, txErr := s.sendTx(ctx, resolved.sourceChainID, adapter, layerZeroSenderAdminABI, "setEnforcedOptions", resolved.destCAIP2, common.FromHex(trimmedOptions))
		if txErr != nil {
			return "", txHashes, txErr
		}
		txHashes = append(txHashes, txHash)
	}

	if len(txHashes) == 0 {
		return "", nil, domainerrors.BadRequest("dstEid+peerHex or optionsHex is required")
	}
	return adapter, txHashes, nil
}

func isValidAdapterAddress(adapter string) bool {
	return common.IsHexAddress(adapter) && common.HexToAddress(adapter) != (common.Address{})
}

func normalizeHexInput(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.TrimPrefix(trimmed, "0x")
}
