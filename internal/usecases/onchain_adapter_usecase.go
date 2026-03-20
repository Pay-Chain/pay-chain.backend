package usecases

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/infrastructure/blockchain"
	"payment-kita.backend/pkg/logger"
)

var (
	FallbackPaymentKitaGatewayABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"setDefaultBridgeType","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"address","name":"adapter","type":"address"},{"internalType":"bool","name":"authorized","type":"bool"}],"name":"setAuthorizedAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"isAuthorizedAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"quoteTotalAmount","outputs":[{"internalType":"uint256","name":"totalAmount","type":"uint256"},{"internalType":"uint256","name":"platformFee","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"FIXED_BASE_FEE","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"FEE_RATE_BPS","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
	]`)
	FallbackPaymentKitaVaultAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"bool","name":"authorized","type":"bool"}],"name":"setAuthorizedSpender","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"authorizedSpenders","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`)
	FallbackPaymentKitaRouterAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"address","name":"adapter","type":"address"}],"name":"registerAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"}
	]`)
	FallbackHyperbridgeAdapterABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"stateMachineId","type":"bytes"}],"name":"setStateMachineId","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"destination","type":"bytes"}],"name":"setDestinationContract","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"address","name":"_router","type":"address"}],"name":"setSwapRouter","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"swapRouter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"address","name":"settlementExecutor","type":"address"}],"name":"setRouteSettlementExecutor","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"nativeCost","type":"uint256"}],"name":"setNativeCost","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"relayerFee","type":"uint256"}],"name":"setRelayerFee","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"settlementExecutors","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"nativeCosts","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"relayerFees","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"isRouteConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`)
	// Deprecated: use FallbackHyperbridgeAdapterABI
	FallbackHyperbridgeSenderAdminABI             = FallbackHyperbridgeAdapterABI
	FallbackHyperbridgeTokenGatewaySenderAdminABI = FallbackHyperbridgeAdapterABI
	FallbackCCIPSenderAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"uint64","name":"selector","type":"uint64"}],"name":"setChainSelector","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"uint64","name":"selector","type":"uint64"},{"internalType":"address","name":"destAdapter","type":"address"}],"name":"setChainConfig","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"adapter","type":"bytes"}],"name":"setDestinationAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"uint256","name":"gasLimit","type":"uint256"}],"name":"setDestinationGasLimit","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"extraArgs","type":"bytes"}],"name":"setDestinationExtraArgs","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"address","name":"feeToken","type":"address"}],"name":"setDestinationFeeToken","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationGasLimits","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationExtraArgs","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationFeeTokens","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`)
	FallbackCCIPReceiverAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"uint64","name":"chainSelector","type":"uint64"},{"internalType":"bytes","name":"sender","type":"bytes"}],"name":"setTrustedSender","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"uint64","name":"chainSelector","type":"uint64"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setSourceChainAllowed","outputs":[],"stateMutability":"nonpayable","type":"function"}
	]`)
	FallbackStargateSenderAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setRoute","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"bytes","name":"extraOptions","type":"bytes"}],"name":"setDestinationExtraOptions","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint128","name":"gasLimit","type":"uint128"}],"name":"setDestinationComposeGasLimit","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"bytes","name":"options","type":"bytes"}],"name":"setEnforcedOptions","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[],"name":"registerDelegate","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"dstEids","outputs":[{"internalType":"uint32","name":"","type":"uint32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"destinationExtraOptions","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"destinationComposeGasLimits","outputs":[{"internalType":"uint128","name":"","type":"uint128"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"enforcedOptions","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"isRouteConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`)
	FallbackStargateReceiverAdminABI = mustParseABI(`[
		{"inputs":[{"internalType":"uint32","name":"eid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setPeer","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"uint32","name":"_srcEid","type":"uint32"},{"internalType":"bytes32","name":"_sender","type":"bytes32"}],"name":"getPathState","outputs":[{"internalType":"bool","name":"peerConfigured","type":"bool"},{"internalType":"bool","name":"trusted","type":"bool"},{"internalType":"bytes32","name":"configuredPeer","type":"bytes32"},{"internalType":"uint64","name":"expectedNonce","type":"uint64"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"uint32","name":"","type":"uint32"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"}
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
			logger.Error(ctx, "failed to connect to RPC", zap.String("rpc_url", rpcURL), zap.Error(err))
			return "", domainerrors.NewError("failed to connect to blockchain RPC: "+err.Error(), err)
		}
		defer client.Close()

		privateKeyHex := strings.TrimPrefix(ownerPrivateKey, "0x")
		privateKey, err := crypto.HexToECDSA(privateKeyHex)
		if err != nil {
			return "", domainerrors.BadRequest("invalid owner private key format")
		}

		chainID, err := client.ChainID(ctx)
		if err != nil {
			logger.Error(ctx, "failed to get chain ID", zap.Error(err))
			return "", domainerrors.NewError("failed to get chainID from RPC: "+err.Error(), err)
		}
		if chainID == nil {
			return "", domainerrors.NewError("chain id is nil from RPC", nil)
		}
		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
		if err != nil {
			return "", domainerrors.NewError("failed to create transactor: "+err.Error(), err)
		}
		auth.Context = ctx

		txHash, err := performContractTransact(client, contractAddress, parsedABI, auth, method, args...)
		if err != nil {
			logger.Error(ctx, "on-chain transaction failed", zap.String("method", method), zap.Error(err))
			return "", domainerrors.NewError("on-chain transaction failed: "+err.Error(), err)
		}
		return txHash, nil
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
	HasAdapterType3                bool   `json:"hasAdapterType3"`
	AdapterType0                   string `json:"adapterType0"`
	AdapterType1                   string `json:"adapterType1"`
	AdapterType2                   string `json:"adapterType2"`
	AdapterType3                   string `json:"adapterType3"`
	HasAdapterDefault              bool   `json:"hasAdapterDefault"`
	AdapterDefaultType             string `json:"adapterDefaultType"`
	HyperbridgeConfigured          bool   `json:"hyperbridgeConfigured"`
	HyperbridgeStateMachineID      string `json:"hyperbridgeStateMachineId"`
	HyperbridgeDestinationContract string `json:"hyperbridgeDestinationContract"`
	HyperbridgeTokenGatewayConfigured         bool   `json:"hyperbridgeTokenGatewayConfigured"`
	HyperbridgeTokenGatewayStateMachineID     string `json:"hyperbridgeTokenGatewayStateMachineId"`
	HyperbridgeTokenGatewaySettlementExecutor string `json:"hyperbridgeTokenGatewaySettlementExecutor"`
	HyperbridgeTokenGatewayNativeCost         string `json:"hyperbridgeTokenGatewayNativeCost"`
	HyperbridgeTokenGatewayRelayerFee         string `json:"hyperbridgeTokenGatewayRelayerFee"`
	CCIPChainSelector              uint64 `json:"ccipChainSelector"`
	CCIPDestinationAdapter         string `json:"ccipDestinationAdapter"`
	CCIPDestinationGasLimit        string `json:"ccipDestinationGasLimit"`
	CCIPDestinationExtraArgsHex    string `json:"ccipDestinationExtraArgsHex"`
	CCIPDestinationFeeTokenAddress string `json:"ccipDestinationFeeTokenAddress"`
	StargateConfigured            bool   `json:"stargateConfigured"`
	StargateDstEID                uint32 `json:"stargateDstEid"`
	StargatePeer                  string `json:"stargatePeer"`
	StargateOptionsHex            string `json:"stargateOptionsHex"`
	StargateComposeGasLimit       string `json:"stargateComposeGasLimit"`
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
				vaultAddress:   "",
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
			return u.callGetAdapter(ctx, evmClient, routerAddress, FallbackPaymentKitaRouterAdminABI, destCAIP2, bridgeType)
		},
		func(ctx context.Context, sourceChainID uuid.UUID, contractAddress string, parsedABI abi.ABI, method string, args ...interface{}) (string, error) {
			return u.sendTx(ctx, sourceChainID, contractAddress, parsedABI, method, args...)
		},
		u.ResolveABIWithFallback,
		func(
			ctx context.Context,
			sourceChainID uuid.UUID,
			contractAddress string,
			parsedABI abi.ABI,
			method string,
			args ...interface{},
		) ([]interface{}, error) {
			if u.clientFactory == nil {
				return nil, domainerrors.BadRequest("evm client factory is not configured")
			}
			chain, err := u.chainRepo.GetByID(ctx, sourceChainID)
			if err != nil {
				return nil, err
			}
			rpcURL := resolveRPCURL(chain)
			if rpcURL == "" {
				return nil, domainerrors.BadRequest("no active rpc url for source chain")
			}
			evmClient, err := u.clientFactory.GetEVMClient(rpcURL)
			if err != nil {
				return nil, err
			}
			defer evmClient.Close()

			data, err := parsedABI.Pack(method, args...)
			if err != nil {
				return nil, err
			}
			out, err := evmClient.CallView(ctx, contractAddress, data)
			if err != nil {
				return nil, err
			}
			return parsedABI.Unpack(method, out)
		},
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
	has3, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 3)
	adapter0, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 0)
	adapter1, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 1)
	adapter2, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 2)
	adapter3, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, 3)
	hasDefault, _ := u.callHasAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, defaultType)
	adapterDefault, _ := u.callGetAdapter(ctx, evmClient, router.ContractAddress, routerABI, destCAIP2, defaultType)
	hyperConfigured := false
	hyperStateMachine := ""
	hyperDestination := ""
	hyperTokenConfigured := false
	hyperTokenStateMachine := ""
	hyperTokenSettlementExecutor := ""
	hyperTokenNativeCost := ""
	hyperTokenRelayerFee := ""
	ccipSelector := uint64(0)
	ccipDestination := ""
	ccipGasLimit := ""
	ccipExtraArgsHex := ""
	ccipFeeToken := ""
	stargateConfigured := false
	stargateDstEid := uint32(0)
	stargatePeer := ""
	stargateOptionsHex := ""
	stargateComposeGasLimit := ""

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
		if gasLimit, gErr := u.callCCIPDestinationGasLimit(ctx, evmClient, adapter1, ccipABI, destCAIP2); gErr == nil {
			ccipGasLimit = gasLimit.String()
		}
		if extraArgs, xErr := u.callCCIPDestinationExtraArgs(ctx, evmClient, adapter1, ccipABI, destCAIP2); xErr == nil {
			ccipExtraArgsHex = "0x" + common.Bytes2Hex(extraArgs)
		}
		if feeToken, fErr := u.callCCIPDestinationFeeToken(ctx, evmClient, adapter1, ccipABI, destCAIP2); fErr == nil {
			ccipFeeToken = feeToken.Hex()
		}
	}
	if has2 && adapter2 != "" && adapter2 != "0x0000000000000000000000000000000000000000" {
		lzABI, _ := u.ResolveABIWithFallback(ctx, sourceChainID, entities.ContractTypeAdapterStargate)
		if configured, cfgErr := u.callStargateConfigured(ctx, evmClient, adapter2, lzABI, destCAIP2); cfgErr == nil {
			stargateConfigured = configured
		}
		if dstEid, dErr := u.callStargateDstEid(ctx, evmClient, adapter2, lzABI, destCAIP2); dErr == nil {
			stargateDstEid = dstEid
		}
		if peer, pErr := u.callStargatePeer(ctx, evmClient, adapter2, lzABI, destCAIP2); pErr == nil {
			stargatePeer = peer.Hex()
		}
		if opts, oErr := u.callStargateOptions(ctx, evmClient, adapter2, lzABI, destCAIP2); oErr == nil && len(opts) > 0 {
			stargateOptionsHex = "0x" + common.Bytes2Hex(opts)
		}
		if gasLimit, gErr := u.callStargateComposeGasLimit(ctx, evmClient, adapter2, lzABI, destCAIP2); gErr == nil && gasLimit != nil && gasLimit.Sign() > 0 {
			stargateComposeGasLimit = gasLimit.String()
		}
	}
	if has3 && adapter3 != "" && adapter3 != "0x0000000000000000000000000000000000000000" {
		hyperTokenABI, _ := u.ResolveABIWithFallback(ctx, sourceChainID, entities.ContractTypeAdapterHyperbridge)
		if configured, cfgErr := u.callTokenGatewayConfigured(ctx, evmClient, adapter3, hyperTokenABI, destCAIP2); cfgErr == nil {
			hyperTokenConfigured = configured
		}
		if sm, smErr := u.callHyperbridgeBytes(ctx, evmClient, adapter3, hyperTokenABI, "stateMachineIds", destCAIP2); smErr == nil && len(sm) > 0 {
			hyperTokenStateMachine = "0x" + common.Bytes2Hex(sm)
		}
		if settlementExecutor, settlementErr := u.callTokenGatewaySettlementExecutor(ctx, evmClient, adapter3, hyperTokenABI, destCAIP2); settlementErr == nil {
			hyperTokenSettlementExecutor = settlementExecutor.Hex()
		}
		if nativeCost, nativeCostErr := u.callTokenGatewayNativeCost(ctx, evmClient, adapter3, hyperTokenABI, destCAIP2); nativeCostErr == nil && nativeCost != nil {
			hyperTokenNativeCost = nativeCost.String()
		}
		if relayerFee, relayerFeeErr := u.callTokenGatewayRelayerFee(ctx, evmClient, adapter3, hyperTokenABI, destCAIP2); relayerFeeErr == nil && relayerFee != nil {
			hyperTokenRelayerFee = relayerFee.String()
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
		HasAdapterType3:                has3,
		AdapterType0:                   adapter0,
		AdapterType1:                   adapter1,
		AdapterType2:                   adapter2,
		AdapterType3:                   adapter3,
		HasAdapterDefault:              hasDefault,
		AdapterDefaultType:             adapterDefault,
		HyperbridgeConfigured:          hyperConfigured,
		HyperbridgeStateMachineID:      hyperStateMachine,
		HyperbridgeDestinationContract: hyperDestination,
		HyperbridgeTokenGatewayConfigured:         hyperTokenConfigured,
		HyperbridgeTokenGatewayStateMachineID:     hyperTokenStateMachine,
		HyperbridgeTokenGatewaySettlementExecutor: hyperTokenSettlementExecutor,
		HyperbridgeTokenGatewayNativeCost:         hyperTokenNativeCost,
		HyperbridgeTokenGatewayRelayerFee:         hyperTokenRelayerFee,
		CCIPChainSelector:              ccipSelector,
		CCIPDestinationAdapter:         ccipDestination,
		CCIPDestinationGasLimit:        ccipGasLimit,
		CCIPDestinationExtraArgsHex:    ccipExtraArgsHex,
		CCIPDestinationFeeTokenAddress: ccipFeeToken,
		StargateConfigured:            stargateConfigured,
		StargateDstEID:                stargateDstEid,
		StargatePeer:                  stargatePeer,
		StargateOptionsHex:            stargateOptionsHex,
		StargateComposeGasLimit:       stargateComposeGasLimit,
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

func (u *OnchainAdapterUsecase) SetHyperbridgeTokenGatewayConfig(
	ctx context.Context,
	input HyperbridgeTokenGatewayConfigInput,
) (string, []string, error) {
	return u.adminOps.SetHyperbridgeTokenGatewayConfig(ctx, input)
}

func (u *OnchainAdapterUsecase) SetCCIPConfig(
	ctx context.Context,
	input CCIPConfigInput,
) (string, []string, error) {
	return u.adminOps.SetCCIPConfig(ctx, input)
}

func (u *OnchainAdapterUsecase) SetStargateConfig(
	ctx context.Context,
	sourceChainInput, destChainInput string,
	dstEid *uint32, peerHex, optionsHex string,
) (string, []string, error) {
	return u.adminOps.SetStargateConfig(ctx, sourceChainInput, destChainInput, dstEid, peerHex, optionsHex)
}

func (u *OnchainAdapterUsecase) ConfigureStargateE2E(
	ctx context.Context,
	input StargateE2EConfigureInput,
) (*StargateE2EConfigureResult, error) {
	if strings.TrimSpace(input.SourceChainInput) == "" || strings.TrimSpace(input.DestChainInput) == "" {
		return nil, domainerrors.BadRequest("sourceChainId and destChainId are required")
	}
	_, destCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.DestChainInput)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid destChainId")
	}
	destChainID, _, err := u.chainResolver.ResolveFromAny(ctx, destCAIP2)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid destination execution chain")
	}
	input.Destination.ChainID = destChainID

	if strings.TrimSpace(input.Source.SenderAddress) == "" {
		sourceChainID, _, srcErr := u.chainResolver.ResolveFromAny(ctx, input.SourceChainInput)
		if srcErr == nil {
			if sender, senderErr := u.contractRepo.GetActiveContract(ctx, sourceChainID, entities.ContractTypeAdapterStargate); senderErr == nil && sender != nil {
				input.Source.SenderAddress = sender.ContractAddress
			}
		}
	}
	if strings.TrimSpace(input.Destination.ReceiverAddress) == "" {
		if receiver, receiverErr := u.contractRepo.GetActiveContract(ctx, input.Destination.ChainID, entities.ContractTypeReceiverStargate); receiverErr == nil && receiver != nil {
			input.Destination.ReceiverAddress = receiver.ContractAddress
		}
	}
	if input.Destination.AuthorizeVaultSpender && strings.TrimSpace(input.Destination.VaultAddress) == "" {
		if vault, vaultErr := u.contractRepo.GetActiveContract(ctx, input.Destination.ChainID, entities.ContractTypeVault); vaultErr == nil && vault != nil {
			input.Destination.VaultAddress = vault.ContractAddress
		}
	}
	if input.Destination.AuthorizeGatewayAdapter && strings.TrimSpace(input.Destination.GatewayAddress) == "" {
		if gateway, gwErr := u.contractRepo.GetActiveContract(ctx, input.Destination.ChainID, entities.ContractTypeGateway); gwErr == nil && gateway != nil {
			input.Destination.GatewayAddress = gateway.ContractAddress
		}
	}

	return u.adminOps.ConfigureStargateE2E(ctx, input)
}

func (u *OnchainAdapterUsecase) GetStargateE2EStatus(
	ctx context.Context,
	input StargateE2EStatusInput,
) (*StargateE2EStatusResult, error) {
	if strings.TrimSpace(input.SourceChainInput) == "" || strings.TrimSpace(input.DestChainInput) == "" {
		return nil, domainerrors.BadRequest("sourceChainId and destChainId are required")
	}
	destChainID, _, err := u.chainResolver.ResolveFromAny(ctx, input.DestChainInput)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid destChainId")
	}
	input.DestinationChainID = destChainID

	if strings.TrimSpace(input.DestinationReceiverAddress) == "" {
		if receiver, receiverErr := u.contractRepo.GetActiveContract(ctx, destChainID, entities.ContractTypeReceiverStargate); receiverErr == nil && receiver != nil {
			input.DestinationReceiverAddress = receiver.ContractAddress
		}
	}
	if strings.TrimSpace(input.DestinationVaultAddress) == "" {
		if vault, vaultErr := u.contractRepo.GetActiveContract(ctx, destChainID, entities.ContractTypeVault); vaultErr == nil && vault != nil {
			input.DestinationVaultAddress = vault.ContractAddress
		}
	}
	if strings.TrimSpace(input.DestinationGatewayAddress) == "" {
		if gateway, gwErr := u.contractRepo.GetActiveContract(ctx, destChainID, entities.ContractTypeGateway); gwErr == nil && gateway != nil {
			input.DestinationGatewayAddress = gateway.ContractAddress
		}
	}

	return u.adminOps.GetStargateE2EStatus(ctx, input)
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

	const maxAttempts = 4
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		txHash, err := executeOnchainTx(ctx, rpcURL, u.ownerPrivateKey, contractAddress, parsedABI, method, args...)
		if err == nil {
			return txHash, nil
		}
		lastErr = err
		if !isRetriableNonceError(err) || attempt == maxAttempts {
			break
		}

		// Backoff a bit so RPC mempool/pending nonce can converge.
		wait := time.Duration(attempt*250) * time.Millisecond
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(wait):
		}
	}
	return "", lastErr
}

func isRetriableNonceError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "nonce too low") ||
		strings.Contains(msg, "replacement transaction underpriced")
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

func (u *OnchainAdapterUsecase) callCCIPDestinationGasLimit(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (*big.Int, error) {
	return callTypedView[*big.Int](ctx, client, adapterAddress, parsedABI, "destinationGasLimits", destCAIP2)
}

func (u *OnchainAdapterUsecase) callCCIPDestinationExtraArgs(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) ([]byte, error) {
	return callTypedView[[]byte](ctx, client, adapterAddress, parsedABI, "destinationExtraArgs", destCAIP2)
}

func (u *OnchainAdapterUsecase) callCCIPDestinationFeeToken(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (common.Address, error) {
	return callTypedView[common.Address](ctx, client, adapterAddress, parsedABI, "destinationFeeTokens", destCAIP2)
}

func (u *OnchainAdapterUsecase) callStargateConfigured(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (bool, error) {
	return callTypedView[bool](ctx, client, adapterAddress, parsedABI, "isRouteConfigured", destCAIP2)
}

func (u *OnchainAdapterUsecase) callTokenGatewayConfigured(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (bool, error) {
	return callTypedView[bool](ctx, client, adapterAddress, parsedABI, "isRouteConfigured", destCAIP2)
}

func (u *OnchainAdapterUsecase) callTokenGatewaySettlementExecutor(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (common.Address, error) {
	return callTypedView[common.Address](ctx, client, adapterAddress, parsedABI, "settlementExecutors", destCAIP2)
}

func (u *OnchainAdapterUsecase) callTokenGatewayNativeCost(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (*big.Int, error) {
	return callTypedView[*big.Int](ctx, client, adapterAddress, parsedABI, "nativeCosts", destCAIP2)
}

func (u *OnchainAdapterUsecase) callTokenGatewayRelayerFee(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (*big.Int, error) {
	return callTypedView[*big.Int](ctx, client, adapterAddress, parsedABI, "relayerFees", destCAIP2)
}

func (u *OnchainAdapterUsecase) callStargateDstEid(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (uint32, error) {
	return callTypedView[uint32](ctx, client, adapterAddress, parsedABI, "dstEids", destCAIP2)
}

func (u *OnchainAdapterUsecase) callStargatePeer(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (common.Hash, error) {
	value, err := callTypedView[[32]byte](ctx, client, adapterAddress, parsedABI, "peers", destCAIP2)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(value[:]), nil
}

func (u *OnchainAdapterUsecase) callStargateOptions(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) ([]byte, error) {
	if _, ok := parsedABI.Methods["destinationExtraOptions"]; ok {
		return callTypedView[[]byte](ctx, client, adapterAddress, parsedABI, "destinationExtraOptions", destCAIP2)
	}
	return callTypedView[[]byte](ctx, client, adapterAddress, parsedABI, "enforcedOptions", destCAIP2)
}

func (u *OnchainAdapterUsecase) callStargateComposeGasLimit(ctx context.Context, client *blockchain.EVMClient, adapterAddress string, parsedABI abi.ABI, destCAIP2 string) (*big.Int, error) {
	if _, ok := parsedABI.Methods["destinationComposeGasLimits"]; ok {
		return callTypedView[*big.Int](ctx, client, adapterAddress, parsedABI, "destinationComposeGasLimits", destCAIP2)
	}
	return nil, fmt.Errorf("destinationComposeGasLimits not supported by ABI")
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

func resolveRPCURLs(chain *entities.Chain) []string {
	if chain == nil {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(chain.RPCs)+1)
	add := func(v string) {
		url := strings.TrimSpace(v)
		if url == "" {
			return
		}
		if _, ok := seen[url]; ok {
			return
		}
		seen[url] = struct{}{}
		out = append(out, url)
	}

	// Keep backward compatibility: prefer legacy primary RPC first.
	add(chain.RPCURL)
	for _, rpc := range chain.RPCs {
		if rpc.IsActive {
			add(rpc.URL)
		}
	}
	for _, rpc := range chain.RPCs {
		add(rpc.URL)
	}
	return out
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
