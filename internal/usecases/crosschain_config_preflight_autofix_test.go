package usecases_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
	uc "pay-chain.backend/internal/usecases"
)

type rpcReqPreflight struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type rpcRespPreflight struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
}

type rpcStatusConfig struct {
	DefaultBridgeType uint8
	AdaptersByType    map[uint8]string
}

func mustParseABIPreflight(t *testing.T, raw string) abi.ABI {
	t.Helper()
	parsed, err := abi.JSON(strings.NewReader(raw))
	require.NoError(t, err)
	return parsed
}

func encodeOutPreflight(t *testing.T, method abi.Method, values ...interface{}) string {
	t.Helper()
	out, err := method.Outputs.Pack(values...)
	require.NoError(t, err)
	return "0x" + hex.EncodeToString(out)
}

func buildStatusRPCServer(t *testing.T, cfg rpcStatusConfig) *httptest.Server {
	t.Helper()

	gatewayABI := mustParseABIPreflight(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
	]`)
	routerABI := mustParseABIPreflight(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"uint8","name":"","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"uint8","name":"","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`)
	layerZeroABI := mustParseABIPreflight(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"isRouteConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"dstEids","outputs":[{"internalType":"uint32","name":"","type":"uint32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"enforcedOptions","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
	]`)
	hyperbridgeABI := mustParseABIPreflight(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
	]`)

	defaultMethodID := "0x" + hex.EncodeToString(gatewayABI.Methods["defaultBridgeTypes"].ID)
	hasMethodID := "0x" + hex.EncodeToString(routerABI.Methods["hasAdapter"].ID)
	getMethodID := "0x" + hex.EncodeToString(routerABI.Methods["getAdapter"].ID)
	lzConfiguredMethodID := "0x" + hex.EncodeToString(layerZeroABI.Methods["isRouteConfigured"].ID)
	lzDstEidMethodID := "0x" + hex.EncodeToString(layerZeroABI.Methods["dstEids"].ID)
	lzPeerMethodID := "0x" + hex.EncodeToString(layerZeroABI.Methods["peers"].ID)
	lzOptionsMethodID := "0x" + hex.EncodeToString(layerZeroABI.Methods["enforcedOptions"].ID)
	hyperConfiguredMethodID := "0x" + hex.EncodeToString(hyperbridgeABI.Methods["isChainConfigured"].ID)
	hyperStateMachineMethodID := "0x" + hex.EncodeToString(hyperbridgeABI.Methods["stateMachineIds"].ID)
	hyperDestinationMethodID := "0x" + hex.EncodeToString(hyperbridgeABI.Methods["destinationContracts"].ID)

	defaultBridgeTypeOut := encodeOutPreflight(t, gatewayABI.Methods["defaultBridgeTypes"], cfg.DefaultBridgeType)

	adapterFor := func(bridgeType uint8) common.Address {
		if cfg.AdaptersByType == nil {
			return common.Address{}
		}
		addr := strings.TrimSpace(cfg.AdaptersByType[bridgeType])
		if !common.IsHexAddress(addr) {
			return common.Address{}
		}
		return common.HexToAddress(addr)
	}

	return newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req rpcReqPreflight
		_ = json.NewDecoder(r.Body).Decode(&req)

		res := rpcRespPreflight{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "eth_chainId":
			res.Result = "0x2105"
		case "eth_call":
			var params []json.RawMessage
			_ = json.Unmarshal(req.Params, &params)
			var callObj struct {
				To    string `json:"to"`
				Data  string `json:"data"`
				Input string `json:"input"`
			}
			if len(params) > 0 {
				_ = json.Unmarshal(params[0], &callObj)
			}
			data := callObj.Data
			if data == "" {
				data = callObj.Input
			}
			methodID := ""
			if len(data) >= 10 {
				methodID = data[:10]
			}
			switch methodID {
			case defaultMethodID:
				res.Result = defaultBridgeTypeOut
			case hasMethodID:
				unpacked, err := routerABI.Methods["hasAdapter"].Inputs.Unpack(common.FromHex(data)[4:])
				if err != nil || len(unpacked) < 2 {
					res.Result = encodeOutPreflight(t, routerABI.Methods["hasAdapter"], false)
					break
				}
				bridgeType, _ := unpacked[1].(uint8)
				res.Result = encodeOutPreflight(t, routerABI.Methods["hasAdapter"], adapterFor(bridgeType) != (common.Address{}))
			case getMethodID:
				unpacked, err := routerABI.Methods["getAdapter"].Inputs.Unpack(common.FromHex(data)[4:])
				if err != nil || len(unpacked) < 2 {
					res.Result = encodeOutPreflight(t, routerABI.Methods["getAdapter"], common.Address{})
					break
				}
				bridgeType, _ := unpacked[1].(uint8)
				res.Result = encodeOutPreflight(t, routerABI.Methods["getAdapter"], adapterFor(bridgeType))
			case lzConfiguredMethodID:
				res.Result = encodeOutPreflight(t, layerZeroABI.Methods["isRouteConfigured"], true)
			case lzDstEidMethodID:
				res.Result = encodeOutPreflight(t, layerZeroABI.Methods["dstEids"], uint32(40161))
			case lzPeerMethodID:
				res.Result = encodeOutPreflight(t, layerZeroABI.Methods["peers"], [32]byte{1})
			case lzOptionsMethodID:
				res.Result = encodeOutPreflight(t, layerZeroABI.Methods["enforcedOptions"], []byte{0x01})
			case hyperConfiguredMethodID:
				res.Result = encodeOutPreflight(t, hyperbridgeABI.Methods["isChainConfigured"], true)
			case hyperStateMachineMethodID:
				res.Result = encodeOutPreflight(t, hyperbridgeABI.Methods["stateMachineIds"], []byte("EVM-42161"))
			case hyperDestinationMethodID:
				res.Result = encodeOutPreflight(t, hyperbridgeABI.Methods["destinationContracts"], common.LeftPadBytes(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes(), 32))
			default:
				res.Result = "0x"
			}
		default:
			res.Result = "0x0"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
}

func setupCrosschainConfigUsecaseNoAdapter(t *testing.T) (*uc.CrosschainConfigUsecase, *MockSmartContractRepository, uuid.UUID, uuid.UUID) {
	t.Helper()

	srv := buildStatusRPCServer(t, rpcStatusConfig{DefaultBridgeType: 2})
	t.Cleanup(srv.Close)

	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true, RPCURL: srv.URL}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo := new(MockChainRepository)
	tokenRepo := new(MockTokenRepository)
	contractRepo := new(MockSmartContractRepository)

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	chainRepo.On("GetByID", mock.Anything, destID).Return(dest, nil)

	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)

	factory := blockchain.NewClientFactory()
	adapterUsecase := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, factory, "")
	usecase := uc.NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, factory, adapterUsecase)

	return usecase, contractRepo, sourceID, destID
}

func TestCrosschainConfigUsecase_Preflight_NoAdapters(t *testing.T) {
	u, _, _, _ := setupCrosschainConfigUsecaseNoAdapter(t)

	result, err := u.Preflight(context.Background(), "eip155:8453", "eip155:42161")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.PolicyExecutable)
	require.Equal(t, uint8(2), result.DefaultBridgeType)
	require.Len(t, result.Bridges, 3)

	for _, row := range result.Bridges {
		require.False(t, row.Ready)
		require.Equal(t, "ADAPTER_NOT_REGISTERED", row.ErrorCode)
	}

	codes := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		codes = append(codes, issue.Code)
	}
	require.Contains(t, codes, "ADAPTER_NOT_REGISTERED")
	require.Contains(t, codes, "LAYERZERO_NOT_CONFIGURED")
}

func TestCrosschainConfigUsecase_AutoFix_NoActiveAdapterContract(t *testing.T) {
	u, contractRepo, sourceID, _ := setupCrosschainConfigUsecaseNoAdapter(t)

	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeAdapterLayerZero).
		Return((*entities.SmartContract)(nil), errors.New("not found"))

	bridgeType := uint8(2)
	result, err := u.AutoFix(context.Background(), &uc.AutoFixRequest{
		SourceChainID: "eip155:8453",
		DestChainID:   "eip155:42161",
		BridgeType:    &bridgeType,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, uint8(2), result.BridgeType)
	require.NotEmpty(t, result.Steps)
	require.Equal(t, "registerAdapter", result.Steps[0].Step)
	require.Equal(t, "FAILED", result.Steps[0].Status)
	require.Contains(t, result.Steps[0].Message, "active adapter contract not found")
}

func TestCrosschainConfigUsecase_AutoFix_LayerZero_AllSkipped(t *testing.T) {
	srv := buildStatusRPCServer(t, rpcStatusConfig{
		DefaultBridgeType: 2,
		AdaptersByType: map[uint8]string{
			2: "0x5555555555555555555555555555555555555555",
		},
	})
	defer srv.Close()

	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true, RPCURL: srv.URL}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo := new(MockChainRepository)
	tokenRepo := new(MockTokenRepository)
	contractRepo := new(MockSmartContractRepository)

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	chainRepo.On("GetByID", mock.Anything, destID).Return(dest, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)

	factory := blockchain.NewClientFactory()
	adapterUsecase := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, factory, "")
	u := uc.NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, factory, adapterUsecase)

	bridgeType := uint8(2)
	result, err := u.AutoFix(context.Background(), &uc.AutoFixRequest{
		SourceChainID: "eip155:8453",
		DestChainID:   "eip155:42161",
		BridgeType:    &bridgeType,
	})
	require.NoError(t, err)
	require.Len(t, result.Steps, 3)
	require.Equal(t, "SKIPPED", result.Steps[0].Status)
	require.Equal(t, "SKIPPED", result.Steps[1].Status)
	require.Equal(t, "SKIPPED", result.Steps[2].Status)
	require.Equal(t, "setLayerZeroConfig", result.Steps[2].Step)
}

func TestCrosschainConfigUsecase_AutoFix_Hyperbridge_DestinationMissing(t *testing.T) {
	srv := buildStatusRPCServer(t, rpcStatusConfig{
		DefaultBridgeType: 0,
		AdaptersByType: map[uint8]string{
			0: "0x4444444444444444444444444444444444444444",
		},
	})
	defer srv.Close()

	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true, RPCURL: srv.URL}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo := new(MockChainRepository)
	tokenRepo := new(MockTokenRepository)
	contractRepo := new(MockSmartContractRepository)

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	chainRepo.On("GetByID", mock.Anything, destID).Return(dest, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)
	contractRepo.On("GetActiveContract", mock.Anything, destID, entities.ContractTypeAdapterHyperbridge).
		Return((*entities.SmartContract)(nil), errors.New("not found"))

	factory := blockchain.NewClientFactory()
	adapterUsecase := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, factory, "")
	u := uc.NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, factory, adapterUsecase)

	result, err := u.AutoFix(context.Background(), &uc.AutoFixRequest{
		SourceChainID: "eip155:8453",
		DestChainID:   "eip155:42161",
	})
	require.NoError(t, err)
	require.Len(t, result.Steps, 3)
	require.Equal(t, "registerAdapter", result.Steps[0].Step)
	require.Equal(t, "SKIPPED", result.Steps[0].Status)
	require.Equal(t, "setDefaultBridge", result.Steps[1].Step)
	require.Equal(t, "SKIPPED", result.Steps[1].Status)
	require.Equal(t, "setHyperbridgeDestination", result.Steps[2].Step)
	require.Equal(t, "FAILED", result.Steps[2].Status)
}
