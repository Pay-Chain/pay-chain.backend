package usecases

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

var (
	FallbackPayChainGatewayABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"setDefaultBridgeType","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"quoteTotalAmount","outputs":[{"internalType":"uint256","name":"totalAmount","type":"uint256"},{"internalType":"uint256","name":"platformFee","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"FIXED_BASE_FEE","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"FEE_RATE_BPS","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
	]`)
	FallbackPayChainRouterAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"address","name":"adapter","type":"address"}],"name":"registerAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"}
	]`)
	FallbackHyperbridgeSenderAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"stateMachineId","type":"bytes"}],"name":"setStateMachineId","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"destination","type":"bytes"}],"name":"setDestinationContract","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"address","name":"_router","type":"address"}],"name":"setSwapRouter","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"swapRouter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`)
	FallbackCCIPSenderAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"uint64","name":"selector","type":"uint64"}],"name":"setChainSelector","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"adapter","type":"bytes"}],"name":"setDestinationAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
	]`)
	FallbackLayerZeroSenderAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setRoute","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"bytes","name":"options","type":"bytes"}],"name":"setEnforcedOptions","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"dstEids","outputs":[{"internalType":"uint32","name":"","type":"uint32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"enforcedOptions","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"isRouteConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`)
	performContractTransact = func(client *ethclient.Client, contractAddress string, parsedABI abi.ABI, auth *bind.TransactOpts, method string, args ...interface{}) (string, error) {
		contract := bind.NewBoundContract(common.HexToAddress(contractAddress), parsedABI, client, client, client)
		tx, err := contract.Transact(auth, method, args...)
		if err != nil {
			return "", err
		}
		return tx.Hash().Hex(), nil
	}
	executeOnchainTx = func(ctx context.Context, rpcURL string, ownerPrivateKey string, contractAddress string, parsedABI abi.ABI, method string, args ...interface{}) (string, error) {
		client, err := ethclient.DialContext(ctx, rpcURL)
		if err != nil {
			return "", err
		}
		defer client.Close()

		privateKeyHex := strings.TrimPrefix(ownerPrivateKey, "0x")
		privateKey, err := crypto.HexToECDSA(privateKeyHex)
		if err != nil {
			return "", domainerrors.BadRequest("invalid owner private key")
		}

		chainID, err := client.ChainID(ctx)
		if err != nil {
			return "", err
		}
		if chainID == nil {
			return "", fmt.Errorf("chain id is nil")
		}
		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
		if err != nil {
			return "", err
		}
		auth.Context = ctx

		return performContractTransact(client, contractAddress, parsedABI, auth, method, args...)
	}
)

type OnchainAdapterStatus struct {
	SourceChainID                  string `json:"sourceChainId"`
	DestChainID                    string `json:"destChainId"`
	GatewayAddress                 string `json:"gatewayAddress"`
	RouterAddress                  string `json:"routerAddress"`
	DefaultBridgeType              uint8  `json:"defaultBridgeType"`
	HasAdapterType0                bool   `json:"hasAdapterType0"`
	HasAdapterType1                bool   `json:"hasAdapterType1"`
	HasAdapterType2                bool   `json:"hasAdapterType2"`
	AdapterType0                   string `json:"adapterType0"`
	AdapterType1                   string `json:"adapterType1"`
	AdapterType2                   string `json:"adapterType2"`
	HasAdapterDefault              bool   `json:"hasAdapterDefault"`
	AdapterDefaultType             string `json:"adapterDefaultType"`
	HyperbridgeConfigured          bool   `json:"hyperbridgeConfigured"`
	HyperbridgeStateMachineID      string `json:"hyperbridgeStateMachineId"`
	HyperbridgeDestinationContract string `json:"hyperbridgeDestinationContract"`
	CCIPChainSelector              uint64 `json:"ccipChainSelector"`
	CCIPDestinationAdapter         string `json:"ccipDestinationAdapter"`
	LayerZeroConfigured            bool   `json:"layerZeroConfigured"`
	LayerZeroDstEID                uint32 `json:"layerZeroDstEid"`
	LayerZeroPeer                  string `json:"layerZeroPeer"`
	LayerZeroOptionsHex            string `json:"layerZeroOptionsHex"`
}

type OnchainAdapterUsecase struct {
	*ABIResolverMixin
	chainRepo       repositories.ChainRepository
	contractRepo    repositories.SmartContractRepository
	clientFactory   *blockchain.ClientFactory
	chainResolver   *ChainResolver
	ownerPrivateKey string
	adminOps        *evmAdminOpsService
}

func NewOnchainAdapterUsecase(
	chainRepo repositories.ChainRepository,
	contractRepo repositories.SmartContractRepository,
	clientFactory *blockchain.ClientFactory,
	ownerPrivateKey string,
) *OnchainAdapterUsecase {
	u := &OnchainAdapterUsecase{
		ABIResolverMixin: NewABIResolverMixin(contractRepo),
		chainRepo:        chainRepo,
		contractRepo:     contractRepo,
		clientFactory:    clientFactory,
		chainResolver:    NewChainResolver(chainRepo),
		ownerPrivateKey:  strings.TrimSpace(ownerPrivateKey),
	}

	u.adminOps = newEVMAdminOpsService(
		func(ctx context.Context, sourceChainInput, destChainInput string) (*evmAdminContext, error) {
			_, sourceChainID, destCAIP2, gateway, router, err := u.resolveEVMContextCore(ctx, sourceChainInput, destChainInput)
			if err != nil {
				return nil, err
			}
			return &evmAdminContext{
				sourceChainID:  sourceChainID,
				destCAIP2:      destCAIP2,
				routerAddress:  router.ContractAddress,
				gatewayAddress: gateway.ContractAddress,
			}, nil
		},
		func(ctx context.Context, sourceChainID uuid.UUID, routerAddress, destCAIP2 string, bridgeType uint8) (string, error) {
			if u.clientFactory == nil {
				return "", domainerrors.BadRequest("evm client factory is not configured")
			}
			chain, err := u.chainRepo.GetByID(ctx, sourceChainID)
			if err != nil {
				return "", err
			}
			rpcURL := resolveRPCURL(chain)
			if rpcURL == "" {
				return "", domainerrors.BadRequest("no active rpc url for source chain")
			}
			evmClient, err := u.clientFactory.GetEVMClient(rpcURL)
			if err != nil {
				return "", err
			}
			defer evmClient.Close()
			return u.callGetAdapter(ctx, evmClient, routerAddress, FallbackPayChainRouterAdminABI, destCAIP2, bridgeType)
		},
		func(ctx context.Context, sourceChainID uuid.UUID, contractAddress string, parsedABI abi.ABI, method string, args ...interface{}) (string, error) {
			return u.sendTx(ctx, sourceChainID, contractAddress, parsedABI, method, args...)
		},
		u.ResolveABIWithFallback,
	)

	return u
}

func (u *OnchainAdapterUsecase) GetStatus(ctx context.Context, sourceChainInput, destChainInput string) (*OnchainAdapterStatus, error) {
	sourceChain, sourceChainID, destCAIP2, gateway, router, evmClient, err := u.resolveEVMContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return nil, err
	}
	defer evmClient.Close()

	gatewayABI, _ := u.ResolveABIWithFallback(ctx, sourceChainID, entities.ContractTypeGateway)
	routerABI, _ := u.ResolveABIWithFallback(ctx, sourceChainID, entities.ContractTypeRouter)
	// We ignore errors here in happy path assuming basics work, or call* will fail if ABI mismatch
	// Actually ResolveABIWithFallback returns error only if contract not found AND fallback unavailable? No, it returns fallback if known type.
	// If unknown type, it errs. These are known types.

	defaultType, err := u.callDefaultBridgeType(ctx, evmClient, gateway.ContractAddress, gatewayABI, destCAIP2)
	if err != nil {
		return nil, err
	}
	has0, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 0)
	has1, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 1)
	has2, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 2)
	adapter0, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 0)
	adapter1, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 1)
	adapter2, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 2)
	hasDefault, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, defaultType)
	adapterDefault, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, defaultType)
	hyperConfigured := false
	hyperStateMachine := ""
	hyperDestination := ""
	ccipSelector := uint64(0)
	ccipDestination := ""
	lzConfigured := false
	lzDstEid := uint32(0)
	lzPeer := ""
	lzOptionsHex := ""

	if has0 && adapter0 != "" && adapter0 != "0x0000000000000000000000000000000000000000" {
		hyperABI, _ := u.ResolveABIWithFallback(ctx, sourceChainID, entities.ContractTypeAdapterHyperbridge)
		if configured, cfgErr := u.callHyperbridgeConfigured(ctx, evmClient, adapter0, hyperABI, destCAIP2); cfgErr == nil {
			hyperConfigured = configured
		}
		if sm, smErr := u.callHyperbridgeBytes(ctx, evmClient, adapter0, hyperABI, "stateMachineIds", destCAIP2); smErr == nil && len(sm) > 0 {
			hyperStateMachine = "0x" + common.Bytes2Hex(sm)
		}
		if dst, dstErr := u.callHyperbridgeBytes(ctx, evmClient, adapter0, hyperABI, "destinationContracts", destCAIP2); dstErr == nil && len(dst) > 0 {
			hyperDestination = "0x" + common.Bytes2Hex(dst)
		}
	}
	if has1 && adapter1 != "" && adapter1 != "0x0000000000000000000000000000000000000000" {
		ccipABI, _ := u.ResolveABIWithFallback(ctx, sourceChainID, entities.ContractTypeAdapterCCIP)
		if selector, sErr := u.callCCIPSelector(ctx, evmClient, adapter1, ccipABI, destCAIP2); sErr == nil {
			ccipSelector = selector
		}
		if dst, dErr := u.callCCIPDestinationAdapter(ctx, evmClient, adapter1, ccipABI, destCAIP2); dErr == nil {
			ccipDestination = "0x" + common.Bytes2Hex(dst)
		}
	}
	if has2 && adapter2 != "" && adapter2 != "0x0000000000000000000000000000000000000000" {
		lzABI, _ := u.ResolveABIWithFallback(ctx, sourceChainID, entities.ContractTypeAdapterLayerZero)
		if configured, cfgErr := u.callLayerZeroConfigured(ctx, evmClient, adapter2, lzABI, destCAIP2); cfgErr == nil {
			lzConfigured = configured
		}
		if dstEid, dErr := u.callLayerZeroDstEid(ctx, evmClient, adapter2, lzABI, destCAIP2); dErr == nil {
			lzDstEid = dstEid
		}
		if peer, pErr := u.callLayerZeroPeer(ctx, evmClient, adapter2, lzABI, destCAIP2); pErr == nil {
			lzPeer = peer.Hex()
		}
		if opts, oErr := u.callLayerZeroOptions(ctx, evmClient, adapter2, lzABI, destCAIP2); oErr == nil && len(opts) > 0 {
			lzOptionsHex = "0x" + common.Bytes2Hex(opts)
		}
	}

	return &OnchainAdapterStatus{
		SourceChainID:                  sourceChain.GetCAIP2ID(),
		DestChainID:                    destCAIP2,
		GatewayAddress:                 gateway.ContractAddress,
		RouterAddress:                  router.ContractAddress,
		DefaultBridgeType:              defaultType,
		HasAdapterType0:                has0,
		HasAdapterType1:                has1,
		HasAdapterType2:                has2,
		AdapterType0:                   adapter0,
		AdapterType1:                   adapter1,
		AdapterType2:                   adapter2,
		HasAdapterDefault:              hasDefault,
		AdapterDefaultType:             adapterDefault,
		HyperbridgeConfigured:          hyperConfigured,
		HyperbridgeStateMachineID:      hyperStateMachine,
		HyperbridgeDestinationContract: hyperDestination,
		CCIPChainSelector:              ccipSelector,
		CCIPDestinationAdapter:         ccipDestination,
		LayerZeroConfigured:            lzConfigured,
		LayerZeroDstEID:                lzDstEid,
		LayerZeroPeer:                  lzPeer,
		LayerZeroOptionsHex:            lzOptionsHex,
	}, nil
}

func (u *OnchainAdapterUsecase) RegisterAdapter(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8, adapterAddress string) (string, error) {
	return u.adminOps.RegisterAdapter(ctx, sourceChainInput, destChainInput, bridgeType, adapterAddress)
}

func (u *OnchainAdapterUsecase) SetDefaultBridgeType(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8) (string, error) {
	return u.adminOps.SetDefaultBridgeType(ctx, sourceChainInput, destChainInput, bridgeType)
}

func (u *OnchainAdapterUsecase) SetHyperbridgeConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	stateMachineIDHex, destinationContractHex string,
) (string, []string, error) {
	return u.adminOps.SetHyperbridgeConfig(ctx, sourceChainInput, destChainInput, stateMachineIDHex, destinationContractHex)
}

func (u *OnchainAdapterUsecase) SetCCIPConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	chainSelector *uint64, destinationAdapterHex string,
) (string, []string, error) {
	return u.adminOps.SetCCIPConfig(ctx, sourceChainInput, destChainInput, chainSelector, destinationAdapterHex)
}

func (u *OnchainAdapterUsecase) SetLayerZeroConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	dstEid *uint32, peerHex, optionsHex string,
) (string, []string, error) {
	return u.adminOps.SetLayerZeroConfig(ctx, sourceChainInput, destChainInput, dstEid, peerHex, optionsHex)
}

func (u *OnchainAdapterUsecase) GenericInteract(
	ctx context.Context,
	sourceChainInput, contractAddress, method, abiStr string,
	args []interface{},
) (interface{}, bool, error) {
	sourceChainID, _, err := u.chainResolver.ResolveFromAny(ctx, sourceChainInput)
	if err != nil {
		return nil, false, domainerrors.BadRequest("invalid sourceChainId")
	}

	if abiStr == "" {
		return nil, false, domainerrors.BadRequest("abi is required for generic interaction")
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, false, domainerrors.BadRequest("invalid abi: " + err.Error())
	}

	m, ok := parsedABI.Methods[method]
	if !ok {
		return nil, false, domainerrors.BadRequest(fmt.Sprintf("method %s not found in abi", method))
	}

	// Convert arguments to types expected by the ABI
	convertedArgs, err := convertArgs(m.Inputs, args)
	if err != nil {
		return nil, false, domainerrors.BadRequest("argument conversion failed: " + err.Error())
	}

	isWrite := m.StateMutability != "view" && m.StateMutability != "pure"

	if isWrite {
		txHash, err := u.sendTx(ctx, sourceChainID, contractAddress, parsedABI, method, convertedArgs...)
		if err != nil {
			return nil, true, err
		}
		return txHash, true, nil
	}

	// Read operation
	chain, err := u.chainRepo.GetByID(ctx, sourceChainID)
	if err != nil {
		return nil, false, err
	}
	rpcURL := resolveRPCURL(chain)
	if rpcURL == "" {
		return nil, false, domainerrors.BadRequest("no active rpc url for chain")
	}
	evmClient, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return nil, false, err
	}
	defer evmClient.Close()

	data, err := parsedABI.Pack(method, convertedArgs...)
	if err != nil {
		return nil, false, err
	}

	out, err := evmClient.CallView(ctx, contractAddress, data)
	if err != nil {
		return nil, false, err
	}

	vals, err := parsedABI.Unpack(method, out)
	if err != nil {
		return nil, false, err
	}

	var result interface{}
	if len(vals) == 1 {
		result = vals[0]
	} else {
		result = vals
	}

	return result, false, nil
}

func convertArgs(inputs abi.Arguments, args []interface{}) ([]interface{}, error) {
	if len(inputs) != len(args) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(inputs), len(args))
	}

	converted := make([]interface{}, len(args))
	for i, input := range inputs {
		val, err := convertArg(input.Type, args[i])
		if err != nil {
			return nil, fmt.Errorf("arg %d (%s): %v", i, input.Name, err)
		}
		converted[i] = val
	}
	return converted, nil
}

func convertArg(t abi.Type, val interface{}) (interface{}, error) {
	switch t.T {
	case abi.AddressTy:
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for address")
		}
		if !common.IsHexAddress(s) {
			return nil, fmt.Errorf("invalid address hex")
		}
		return common.HexToAddress(s), nil
	case abi.FixedBytesTy:
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected hex string for fixed bytes")
		}
		b := common.FromHex(s)
		if len(b) > t.Size {
			return nil, fmt.Errorf("bytes length %d exceeds %d", len(b), t.Size)
		}
		// Pad to correct size
		padded := make([]byte, t.Size)
		copy(padded[len(padded)-len(b):], b)

		// We need to return the correct fixed size array type for go-ethereum
		// This is tricky because it depends on t.Size.
		// For common sizes (bytes32), we can handle them explicitly.
		if t.Size == 32 {
			var res [32]byte
			copy(res[:], padded)
			return res, nil
		}
		return padded, nil // Fallback for others, might fail Pack
	case abi.BytesTy:
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected hex string for bytes")
		}
		return common.FromHex(s), nil
	case abi.IntTy, abi.UintTy:
		// JSON numbers are float64 by default in Go
		var bigVal *big.Int
		switch v := val.(type) {
		case float64:
			bigVal = new(big.Int).SetInt64(int64(v))
		case string:
			var ok bool
			bigVal, ok = new(big.Int).SetString(v, 0)
			if !ok {
				return nil, fmt.Errorf("invalid number string")
			}
		default:
			return nil, fmt.Errorf("invalid number type %T", val)
		}

		// Convert to specific bitsize if needed (for Pack)
		switch t.Size {
		case 8:
			if t.T == abi.UintTy {
				return uint8(bigVal.Uint64()), nil
			}
			return int8(bigVal.Int64()), nil
		case 16:
			if t.T == abi.UintTy {
				return uint16(bigVal.Uint64()), nil
			}
			return int16(bigVal.Int64()), nil
		case 32:
			if t.T == abi.UintTy {
				return uint32(bigVal.Uint64()), nil
			}
			return int32(bigVal.Int64()), nil
		case 64:
			if t.T == abi.UintTy {
				return bigVal.Uint64(), nil
			}
			return bigVal.Int64(), nil
		default:
			return bigVal, nil
		}
	case abi.BoolTy:
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected boolean")
		}
		return b, nil
	case abi.StringTy:
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string")
		}
		return s, nil
	default:
		return val, nil // Fallback
	}
}

func (u *OnchainAdapterUsecase) resolveEVMContext(
	ctx context.Context,
	sourceChainInput, destChainInput string,
) (*entities.Chain, uuid.UUID, string, *entities.SmartContract, *entities.SmartContract, *blockchain.EVMClient, error) {
	sourceChain, sourceChainID, destCAIP2, gateway, router, err := u.resolveEVMContextCore(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, nil, err
	}

	rpcURL := resolveRPCURL(sourceChain)
	if rpcURL == "" {
		return nil, uuid.Nil, "", nil, nil, nil, domainerrors.BadRequest("no active rpc url for source chain")
	}
	evmClient, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, nil, err
	}
	return sourceChain, sourceChainID, destCAIP2, gateway, router, evmClient, nil
}

func (u *OnchainAdapterUsecase) resolveEVMContextCore(
	ctx context.Context,
	sourceChainInput, destChainInput string,
) (*entities.Chain, uuid.UUID, string, *entities.SmartContract, *entities.SmartContract, error) {
	sourceChainID, _, err := u.chainResolver.ResolveFromAny(ctx, sourceChainInput)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, domainerrors.BadRequest("invalid sourceChainId")
	}
	_, destCAIP2, err := u.chainResolver.ResolveFromAny(ctx, destChainInput)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, domainerrors.BadRequest("invalid destChainId")
	}

	sourceChain, err := u.chainRepo.GetByID(ctx, sourceChainID)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, err
	}
	if sourceChain.Type != entities.ChainTypeEVM {
		return nil, uuid.Nil, "", nil, nil, domainerrors.BadRequest("only EVM source chain is supported")
	}

	gateway, err := u.contractRepo.GetActiveContract(ctx, sourceChain.ID, entities.ContractTypeGateway)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, domainerrors.BadRequest("active gateway contract not found on source chain")
	}
	router, err := u.contractRepo.GetActiveContract(ctx, sourceChain.ID, entities.ContractTypeRouter)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, domainerrors.BadRequest("active router contract not found on source chain")
	}
	return sourceChain, sourceChain.ID, destCAIP2, gateway, router, nil
}

func (u *OnchainAdapterUsecase) sendTx(
	ctx context.Context,
	sourceChainID uuid.UUID,
	contractAddress string,
	parsedABI abi.ABI,
	method string,
	args ...interface{},
) (string, error) {
	if u.ownerPrivateKey == "" {
		return "", domainerrors.BadRequest("owner private key is not configured")
	}
	chain, err := u.chainRepo.GetByID(ctx, sourceChainID)
	if err != nil {
		return "", err
	}
	rpcURL := resolveRPCURL(chain)
	if rpcURL == "" {
		return "", domainerrors.BadRequest("no active rpc url for source chain")
	}
	return executeOnchainTx(ctx, rpcURL, u.ownerPrivateKey, contractAddress, parsedABI, method, args...)
}

func (u *OnchainAdapterUsecase) callDefaultBridgeType(ctx context.Context, client *blockchain.EVMClient, gatewayAddress string, parsedABI abi.ABI, destCAIP2 string) (uint8, error) {
	return callTypedView[uint8](ctx, client, gatewayAddress, parsedABI, "defaultBridgeTypes", destCAIP2)
}

func (u *OnchainAdapterUsecase) callHasAdapter(ctx context.Context, client *blockchain.EVMClient, routerAddress string, parsedABI abi.ABI, destCAIP2 string, bridgeType uint8) (bool, error) {
	return callTypedView[bool](ctx, client, routerAddress, parsedABI, "hasAdapter", destCAIP2, bridgeType)
}

func (u *OnchainAdapterUsecase) callGetAdapter(ctx context.Context, client *blockchain.EVMClient, routerAddress string, parsedABI abi.ABI, destCAIP2 string, bridgeType uint8) (string, error) {
	value, err := callTypedView[common.Address](ctx, client, routerAddress, parsedABI, "getAdapter", destCAIP2, bridgeType)
	if err != nil {
		return "", err
	}
	return value.Hex(), nil
}

func (u *OnchainAdapterUsecase) callHyperbridgeConfigured(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (bool, error) {
	return callTypedView[bool](ctx, client, adapterAddress, parsedABI, "isChainConfigured", destCAIP2)
}

func (u *OnchainAdapterUsecase) callHyperbridgeBytes(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, method, destCAIP2 string) ([]byte, error) {
	return callTypedView[[]byte](ctx, client, adapterAddress, parsedABI, method, destCAIP2)
}

func (u *OnchainAdapterUsecase) callCCIPSelector(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (uint64, error) {
	return callTypedView[uint64](ctx, client, adapterAddress, parsedABI, "chainSelectors", destCAIP2)
}

func (u *OnchainAdapterUsecase) callCCIPDestinationAdapter(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) ([]byte, error) {
	return callTypedView[[]byte](ctx, client, adapterAddress, parsedABI, "destinationAdapters", destCAIP2)
}

func (u *OnchainAdapterUsecase) callLayerZeroConfigured(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (bool, error) {
	return callTypedView[bool](ctx, client, adapterAddress, parsedABI, "isRouteConfigured", destCAIP2)
}

func (u *OnchainAdapterUsecase) callLayerZeroDstEid(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (uint32, error) {
	return callTypedView[uint32](ctx, client, adapterAddress, parsedABI, "dstEids", destCAIP2)
}

func (u *OnchainAdapterUsecase) callLayerZeroPeer(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (common.Hash, error) {
	value, err := callTypedView[[32]byte](ctx, client, adapterAddress, parsedABI, "peers", destCAIP2)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(value[:]), nil
}

func (u *OnchainAdapterUsecase) callLayerZeroOptions(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) ([]byte, error) {
	return callTypedView[[]byte](ctx, client, adapterAddress, parsedABI, "enforcedOptions", destCAIP2)
}

func callTypedView[T any](
	ctx context.Context,
	client *blockchain.EVMClient,
	contractAddress string,
	parsedABI abi.ABI,
	method string,
	args ...interface{},
) (T, error) {
	var zero T

	data, err := parsedABI.Pack(method, args...)
	if err != nil {
		return zero, err
	}
	out, err := client.CallView(ctx, contractAddress, data)
	if err != nil {
		return zero, err
	}
	vals, err := parsedABI.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return zero, fmt.Errorf("failed to decode %s", method)
	}
	value, ok := vals[0].(T)
	if !ok {
		return zero, fmt.Errorf("invalid %s return type", method)
	}
	return value, nil
}

func mustParseABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}

func resolveRPCURL(chain *entities.Chain) string {
	if chain == nil {
		return ""
	}
	if strings.TrimSpace(chain.RPCURL) != "" {
		return chain.RPCURL
	}
	for _, rpc := range chain.RPCs {
		if rpc.IsActive && strings.TrimSpace(rpc.URL) != "" {
			return rpc.URL
		}
	}
	for _, rpc := range chain.RPCs {
		if strings.TrimSpace(rpc.URL) != "" {
			return rpc.URL
		}
	}
	return ""
}

func parseHexToBytes32(v string) ([32]byte, error) {
	var out [32]byte
	raw := strings.TrimSpace(v)
	if !strings.HasPrefix(raw, "0x") {
		raw = "0x" + raw
	}
	b := common.FromHex(raw)
	if len(b) == 20 {
		b = common.LeftPadBytes(b, 32)
	}
	if len(b) != 32 {
		return out, fmt.Errorf("invalid bytes32 length")
	}
	copy(out[:], b)
	return out, nil
}
