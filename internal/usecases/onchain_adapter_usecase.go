package usecases

import (
	"context"
	"fmt"
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
	payChainGatewayAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"setDefaultBridgeType","outputs":[],"stateMutability":"nonpayable","type":"function"}
	]`)
	payChainRouterAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"address","name":"adapter","type":"address"}],"name":"registerAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"}
	]`)
	hyperbridgeSenderAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"stateMachineId","type":"bytes"}],"name":"setStateMachineId","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"destination","type":"bytes"}],"name":"setDestinationContract","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"}
	]`)
	ccipSenderAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"uint64","name":"selector","type":"uint64"}],"name":"setChainSelector","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"adapter","type":"bytes"}],"name":"setDestinationAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
	]`)
	layerZeroSenderAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setRoute","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"bytes","name":"options","type":"bytes"}],"name":"setEnforcedOptions","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"dstEids","outputs":[{"internalType":"uint32","name":"","type":"uint32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"enforcedOptions","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"isRouteConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`)
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
	chainRepo       repositories.ChainRepository
	contractRepo    repositories.SmartContractRepository
	clientFactory   *blockchain.ClientFactory
	chainResolver   *ChainResolver
	ownerPrivateKey string
}

func NewOnchainAdapterUsecase(
	chainRepo repositories.ChainRepository,
	contractRepo repositories.SmartContractRepository,
	clientFactory *blockchain.ClientFactory,
	ownerPrivateKey string,
) *OnchainAdapterUsecase {
	return &OnchainAdapterUsecase{
		chainRepo:       chainRepo,
		contractRepo:    contractRepo,
		clientFactory:   clientFactory,
		chainResolver:   NewChainResolver(chainRepo),
		ownerPrivateKey: strings.TrimSpace(ownerPrivateKey),
	}
}

func (u *OnchainAdapterUsecase) GetStatus(ctx context.Context, sourceChainInput, destChainInput string) (*OnchainAdapterStatus, error) {
	sourceChain, _, destCAIP2, gateway, router, evmClient, err := u.resolveEVMContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return nil, err
	}
	defer evmClient.Close()

	defaultType, err := u.callDefaultBridgeType(ctx, evmClient, gateway.ContractAddress, destCAIP2)
	if err != nil {
		return nil, err
	}
	has0, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, 0)
	has1, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, 1)
	has2, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, 2)
	adapter0, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, 0)
	adapter1, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, 1)
	adapter2, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, 2)
	hasDefault, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, defaultType)
	adapterDefault, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, defaultType)
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
		if configured, cfgErr := u.callHyperbridgeConfigured(ctx, evmClient, adapter0, destCAIP2); cfgErr == nil {
			hyperConfigured = configured
		}
		if sm, smErr := u.callHyperbridgeBytes(ctx, evmClient, adapter0, "stateMachineIds", destCAIP2); smErr == nil {
			hyperStateMachine = "0x" + common.Bytes2Hex(sm)
		}
		if dst, dstErr := u.callHyperbridgeBytes(ctx, evmClient, adapter0, "destinationContracts", destCAIP2); dstErr == nil {
			hyperDestination = "0x" + common.Bytes2Hex(dst)
		}
	}
	if has1 && adapter1 != "" && adapter1 != "0x0000000000000000000000000000000000000000" {
		if selector, sErr := u.callCCIPSelector(ctx, evmClient, adapter1, destCAIP2); sErr == nil {
			ccipSelector = selector
		}
		if dst, dErr := u.callCCIPDestinationAdapter(ctx, evmClient, adapter1, destCAIP2); dErr == nil {
			ccipDestination = "0x" + common.Bytes2Hex(dst)
		}
	}
	if has2 && adapter2 != "" && adapter2 != "0x0000000000000000000000000000000000000000" {
		if configured, cfgErr := u.callLayerZeroConfigured(ctx, evmClient, adapter2, destCAIP2); cfgErr == nil {
			lzConfigured = configured
		}
		if dstEid, dErr := u.callLayerZeroDstEid(ctx, evmClient, adapter2, destCAIP2); dErr == nil {
			lzDstEid = dstEid
		}
		if peer, pErr := u.callLayerZeroPeer(ctx, evmClient, adapter2, destCAIP2); pErr == nil {
			lzPeer = peer.Hex()
		}
		if opts, oErr := u.callLayerZeroOptions(ctx, evmClient, adapter2, destCAIP2); oErr == nil && len(opts) > 0 {
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
	if !common.IsHexAddress(adapterAddress) || common.HexToAddress(adapterAddress) == (common.Address{}) {
		return "", domainerrors.BadRequest("invalid adapterAddress")
	}

	_, sourceChainID, destCAIP2, _, router, _, err := u.resolveEVMContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return "", err
	}

	txHash, err := u.sendTx(
		ctx,
		sourceChainID,
		router.ContractAddress,
		payChainRouterAdminABI,
		"registerAdapter",
		destCAIP2,
		bridgeType,
		common.HexToAddress(adapterAddress),
	)
	if err != nil {
		return "", err
	}
	return txHash, nil
}

func (u *OnchainAdapterUsecase) SetDefaultBridgeType(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8) (string, error) {
	_, sourceChainID, destCAIP2, gateway, _, _, err := u.resolveEVMContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return "", err
	}

	txHash, err := u.sendTx(
		ctx,
		sourceChainID,
		gateway.ContractAddress,
		payChainGatewayAdminABI,
		"setDefaultBridgeType",
		destCAIP2,
		bridgeType,
	)
	if err != nil {
		return "", err
	}
	return txHash, nil
}

func (u *OnchainAdapterUsecase) SetHyperbridgeConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	stateMachineIDHex, destinationContractHex string,
) (string, []string, error) {
	_, sourceChainID, destCAIP2, _, router, evmClient, err := u.resolveEVMContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return "", nil, err
	}
	defer evmClient.Close()

	adapter, err := u.callGetAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, 0)
	if err != nil {
		return "", nil, err
	}
	if adapter == "" || adapter == "0x0000000000000000000000000000000000000000" {
		return "", nil, domainerrors.BadRequest("hyperbridge adapter (type 0) is not registered")
	}

	var txHashes []string
	target := strings.TrimPrefix(strings.TrimSpace(stateMachineIDHex), "0x")
	if target != "" {
		txHash, txErr := u.sendTx(ctx, sourceChainID, adapter, hyperbridgeSenderAdminABI, "setStateMachineId", destCAIP2, common.FromHex("0x"+target))
		if txErr != nil {
			return "", txHashes, txErr
		}
		txHashes = append(txHashes, txHash)
	}

	dest := strings.TrimPrefix(strings.TrimSpace(destinationContractHex), "0x")
	if dest != "" {
		txHash, txErr := u.sendTx(ctx, sourceChainID, adapter, hyperbridgeSenderAdminABI, "setDestinationContract", destCAIP2, common.FromHex("0x"+dest))
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

func (u *OnchainAdapterUsecase) SetCCIPConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	chainSelector *uint64, destinationAdapterHex string,
) (string, []string, error) {
	_, sourceChainID, destCAIP2, _, router, evmClient, err := u.resolveEVMContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return "", nil, err
	}
	defer evmClient.Close()

	adapter, err := u.callGetAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, 1)
	if err != nil {
		return "", nil, err
	}
	if adapter == "" || adapter == "0x0000000000000000000000000000000000000000" {
		return "", nil, domainerrors.BadRequest("ccip adapter (type 1) is not registered")
	}

	var txHashes []string
	if chainSelector != nil {
		txHash, txErr := u.sendTx(ctx, sourceChainID, adapter, ccipSenderAdminABI, "setChainSelector", destCAIP2, *chainSelector)
		if txErr != nil {
			return "", txHashes, txErr
		}
		txHashes = append(txHashes, txHash)
	}
	dest := strings.TrimPrefix(strings.TrimSpace(destinationAdapterHex), "0x")
	if dest != "" {
		txHash, txErr := u.sendTx(ctx, sourceChainID, adapter, ccipSenderAdminABI, "setDestinationAdapter", destCAIP2, common.FromHex("0x"+dest))
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

func (u *OnchainAdapterUsecase) SetLayerZeroConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	dstEid *uint32, peerHex, optionsHex string,
) (string, []string, error) {
	_, sourceChainID, destCAIP2, _, router, evmClient, err := u.resolveEVMContext(ctx, sourceChainInput, destChainInput)
	if err != nil {
		return "", nil, err
	}
	defer evmClient.Close()

	adapter, err := u.callGetAdapter(ctx, evmClient, router.ContractAddress, destCAIP2, 2)
	if err != nil {
		return "", nil, err
	}
	if adapter == "" || adapter == "0x0000000000000000000000000000000000000000" {
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
		txHash, txErr := u.sendTx(ctx, sourceChainID, adapter, layerZeroSenderAdminABI, "setRoute", destCAIP2, *dstEid, peer32)
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
		txHash, txErr := u.sendTx(ctx, sourceChainID, adapter, layerZeroSenderAdminABI, "setEnforcedOptions", destCAIP2, common.FromHex(trimmedOptions))
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

func (u *OnchainAdapterUsecase) resolveEVMContext(
	ctx context.Context,
	sourceChainInput, destChainInput string,
) (*entities.Chain, uuid.UUID, string, *entities.SmartContract, *entities.SmartContract, *blockchain.EVMClient, error) {
	sourceChainID, _, err := u.chainResolver.ResolveFromAny(ctx, sourceChainInput)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, nil, domainerrors.BadRequest("invalid sourceChainId")
	}
	_, destCAIP2, err := u.chainResolver.ResolveFromAny(ctx, destChainInput)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, nil, domainerrors.BadRequest("invalid destChainId")
	}

	sourceChain, err := u.chainRepo.GetByID(ctx, sourceChainID)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, nil, err
	}
	if sourceChain.Type != entities.ChainTypeEVM {
		return nil, uuid.Nil, "", nil, nil, nil, domainerrors.BadRequest("only EVM source chain is supported")
	}

	gateway, err := u.contractRepo.GetActiveContract(ctx, sourceChain.ID, entities.ContractTypeGateway)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, nil, domainerrors.BadRequest("active gateway contract not found on source chain")
	}
	router, err := u.contractRepo.GetActiveContract(ctx, sourceChain.ID, entities.ContractTypeRouter)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, nil, domainerrors.BadRequest("active router contract not found on source chain")
	}

	rpcURL := resolveRPCURL(sourceChain)
	if rpcURL == "" {
		return nil, uuid.Nil, "", nil, nil, nil, domainerrors.BadRequest("no active rpc url for source chain")
	}
	evmClient, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return nil, uuid.Nil, "", nil, nil, nil, err
	}
	return sourceChain, sourceChain.ID, destCAIP2, gateway, router, evmClient, nil
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

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return "", err
	}
	defer client.Close()

	privateKeyHex := strings.TrimPrefix(u.ownerPrivateKey, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", domainerrors.BadRequest("invalid owner private key")
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return "", err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return "", err
	}
	auth.Context = ctx

	contract := bind.NewBoundContract(common.HexToAddress(contractAddress), parsedABI, client, client, client)
	tx, err := contract.Transact(auth, method, args...)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

func (u *OnchainAdapterUsecase) callDefaultBridgeType(ctx context.Context, client *blockchain.EVMClient, gatewayAddress, destCAIP2 string) (uint8, error) {
	data, err := payChainGatewayAdminABI.Pack("defaultBridgeTypes", destCAIP2)
	if err != nil {
		return 0, err
	}
	out, err := client.CallView(ctx, gatewayAddress, data)
	if err != nil {
		return 0, err
	}
	vals, err := payChainGatewayAdminABI.Unpack("defaultBridgeTypes", out)
	if err != nil || len(vals) == 0 {
		return 0, fmt.Errorf("failed to decode defaultBridgeTypes")
	}
	value, ok := vals[0].(uint8)
	if !ok {
		return 0, fmt.Errorf("invalid defaultBridgeTypes return type")
	}
	return value, nil
}

func (u *OnchainAdapterUsecase) callHasAdapter(ctx context.Context, client *blockchain.EVMClient, routerAddress, destCAIP2 string, bridgeType uint8) (bool, error) {
	data, err := payChainRouterAdminABI.Pack("hasAdapter", destCAIP2, bridgeType)
	if err != nil {
		return false, err
	}
	out, err := client.CallView(ctx, routerAddress, data)
	if err != nil {
		return false, err
	}
	vals, err := payChainRouterAdminABI.Unpack("hasAdapter", out)
	if err != nil || len(vals) == 0 {
		return false, fmt.Errorf("failed to decode hasAdapter")
	}
	value, ok := vals[0].(bool)
	if !ok {
		return false, fmt.Errorf("invalid hasAdapter return type")
	}
	return value, nil
}

func (u *OnchainAdapterUsecase) callGetAdapter(ctx context.Context, client *blockchain.EVMClient, routerAddress, destCAIP2 string, bridgeType uint8) (string, error) {
	data, err := payChainRouterAdminABI.Pack("getAdapter", destCAIP2, bridgeType)
	if err != nil {
		return "", err
	}
	out, err := client.CallView(ctx, routerAddress, data)
	if err != nil {
		return "", err
	}
	vals, err := payChainRouterAdminABI.Unpack("getAdapter", out)
	if err != nil || len(vals) == 0 {
		return "", fmt.Errorf("failed to decode getAdapter")
	}
	value, ok := vals[0].(common.Address)
	if !ok {
		return "", fmt.Errorf("invalid getAdapter return type")
	}
	return value.Hex(), nil
}

func (u *OnchainAdapterUsecase) callHyperbridgeConfigured(ctx context.Context, client *blockchain.EVMClient, adapterAddress, destCAIP2 string) (bool, error) {
	data, err := hyperbridgeSenderAdminABI.Pack("isChainConfigured", destCAIP2)
	if err != nil {
		return false, err
	}
	out, err := client.CallView(ctx, adapterAddress, data)
	if err != nil {
		return false, err
	}
	vals, err := hyperbridgeSenderAdminABI.Unpack("isChainConfigured", out)
	if err != nil || len(vals) == 0 {
		return false, fmt.Errorf("failed to decode isChainConfigured")
	}
	value, ok := vals[0].(bool)
	if !ok {
		return false, fmt.Errorf("invalid isChainConfigured return type")
	}
	return value, nil
}

func (u *OnchainAdapterUsecase) callHyperbridgeBytes(ctx context.Context, client *blockchain.EVMClient, adapterAddress, method, destCAIP2 string) ([]byte, error) {
	data, err := hyperbridgeSenderAdminABI.Pack(method, destCAIP2)
	if err != nil {
		return nil, err
	}
	out, err := client.CallView(ctx, adapterAddress, data)
	if err != nil {
		return nil, err
	}
	vals, err := hyperbridgeSenderAdminABI.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return nil, fmt.Errorf("failed to decode %s", method)
	}
	value, ok := vals[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid %s return type", method)
	}
	return value, nil
}

func (u *OnchainAdapterUsecase) callCCIPSelector(ctx context.Context, client *blockchain.EVMClient, adapterAddress, destCAIP2 string) (uint64, error) {
	data, err := ccipSenderAdminABI.Pack("chainSelectors", destCAIP2)
	if err != nil {
		return 0, err
	}
	out, err := client.CallView(ctx, adapterAddress, data)
	if err != nil {
		return 0, err
	}
	vals, err := ccipSenderAdminABI.Unpack("chainSelectors", out)
	if err != nil || len(vals) == 0 {
		return 0, fmt.Errorf("failed to decode chainSelectors")
	}
	value, ok := vals[0].(uint64)
	if !ok {
		return 0, fmt.Errorf("invalid chainSelectors return type")
	}
	return value, nil
}

func (u *OnchainAdapterUsecase) callCCIPDestinationAdapter(ctx context.Context, client *blockchain.EVMClient, adapterAddress, destCAIP2 string) ([]byte, error) {
	data, err := ccipSenderAdminABI.Pack("destinationAdapters", destCAIP2)
	if err != nil {
		return nil, err
	}
	out, err := client.CallView(ctx, adapterAddress, data)
	if err != nil {
		return nil, err
	}
	vals, err := ccipSenderAdminABI.Unpack("destinationAdapters", out)
	if err != nil || len(vals) == 0 {
		return nil, fmt.Errorf("failed to decode destinationAdapters")
	}
	value, ok := vals[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid destinationAdapters return type")
	}
	return value, nil
}

func (u *OnchainAdapterUsecase) callLayerZeroConfigured(ctx context.Context, client *blockchain.EVMClient, adapterAddress, destCAIP2 string) (bool, error) {
	data, err := layerZeroSenderAdminABI.Pack("isRouteConfigured", destCAIP2)
	if err != nil {
		return false, err
	}
	out, err := client.CallView(ctx, adapterAddress, data)
	if err != nil {
		return false, err
	}
	vals, err := layerZeroSenderAdminABI.Unpack("isRouteConfigured", out)
	if err != nil || len(vals) == 0 {
		return false, fmt.Errorf("failed to decode isRouteConfigured")
	}
	value, ok := vals[0].(bool)
	if !ok {
		return false, fmt.Errorf("invalid isRouteConfigured return type")
	}
	return value, nil
}

func (u *OnchainAdapterUsecase) callLayerZeroDstEid(ctx context.Context, client *blockchain.EVMClient, adapterAddress, destCAIP2 string) (uint32, error) {
	data, err := layerZeroSenderAdminABI.Pack("dstEids", destCAIP2)
	if err != nil {
		return 0, err
	}
	out, err := client.CallView(ctx, adapterAddress, data)
	if err != nil {
		return 0, err
	}
	vals, err := layerZeroSenderAdminABI.Unpack("dstEids", out)
	if err != nil || len(vals) == 0 {
		return 0, fmt.Errorf("failed to decode dstEids")
	}
	value, ok := vals[0].(uint32)
	if !ok {
		return 0, fmt.Errorf("invalid dstEids return type")
	}
	return value, nil
}

func (u *OnchainAdapterUsecase) callLayerZeroPeer(ctx context.Context, client *blockchain.EVMClient, adapterAddress, destCAIP2 string) (common.Hash, error) {
	data, err := layerZeroSenderAdminABI.Pack("peers", destCAIP2)
	if err != nil {
		return common.Hash{}, err
	}
	out, err := client.CallView(ctx, adapterAddress, data)
	if err != nil {
		return common.Hash{}, err
	}
	vals, err := layerZeroSenderAdminABI.Unpack("peers", out)
	if err != nil || len(vals) == 0 {
		return common.Hash{}, fmt.Errorf("failed to decode peers")
	}
	value, ok := vals[0].([32]byte)
	if !ok {
		return common.Hash{}, fmt.Errorf("invalid peers return type")
	}
	return common.BytesToHash(value[:]), nil
}

func (u *OnchainAdapterUsecase) callLayerZeroOptions(ctx context.Context, client *blockchain.EVMClient, adapterAddress, destCAIP2 string) ([]byte, error) {
	data, err := layerZeroSenderAdminABI.Pack("enforcedOptions", destCAIP2)
	if err != nil {
		return nil, err
	}
	out, err := client.CallView(ctx, adapterAddress, data)
	if err != nil {
		return nil, err
	}
	vals, err := layerZeroSenderAdminABI.Unpack("enforcedOptions", out)
	if err != nil || len(vals) == 0 {
		return nil, fmt.Errorf("failed to decode enforcedOptions")
	}
	value, ok := vals[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid enforcedOptions return type")
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
