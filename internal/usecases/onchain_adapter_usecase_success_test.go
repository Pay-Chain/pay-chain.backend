package usecases_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
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

type rpcReqOA struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type rpcRespOA struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func mustParseABI(t *testing.T, raw string) abi.ABI {
	t.Helper()
	parsed, err := abi.JSON(strings.NewReader(raw))
	require.NoError(t, err)
	return parsed
}

func encodeOut(t *testing.T, method abi.Method, values ...interface{}) string {
	t.Helper()
	out, err := method.Outputs.Pack(values...)
	require.NoError(t, err)
	return "0x" + hex.EncodeToString(out)
}

func TestOnchainAdapterUsecase_GetStatus_Success(t *testing.T) {
	gatewayABI := mustParseABI(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
	]`)
	routerABI := mustParseABI(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"uint8","name":"","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"uint8","name":"","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`)
	hyperABI := mustParseABI(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
	]`)

	adapter0 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	stateMachine := []byte("eip155:42161")
	destinationContract := common.LeftPadBytes(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes(), 32)

	callResults := []string{
		encodeOut(t, gatewayABI.Methods["defaultBridgeTypes"], uint8(0)),            // default bridge
		encodeOut(t, routerABI.Methods["hasAdapter"], true),                         // has type0
		encodeOut(t, routerABI.Methods["hasAdapter"], false),                        // has type1
		encodeOut(t, routerABI.Methods["hasAdapter"], false),                        // has type2
		encodeOut(t, routerABI.Methods["getAdapter"], adapter0),                     // adapter type0
		encodeOut(t, routerABI.Methods["getAdapter"], common.Address{}),             // adapter type1
		encodeOut(t, routerABI.Methods["getAdapter"], common.Address{}),             // adapter type2
		encodeOut(t, routerABI.Methods["hasAdapter"], true),                         // has default
		encodeOut(t, routerABI.Methods["getAdapter"], adapter0),                     // adapter default
		encodeOut(t, hyperABI.Methods["isChainConfigured"], true),                   // hyper configured
		encodeOut(t, hyperABI.Methods["stateMachineIds"], stateMachine),             // hyper state machine
		encodeOut(t, hyperABI.Methods["destinationContracts"], destinationContract), // hyper destination
	}
	var (
		callMu sync.Mutex
		callIx int
	)

	srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req rpcReqOA
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := rpcRespOA{JSONRPC: "2.0", ID: req.ID}

		switch req.Method {
		case "eth_chainId":
			res.Result = "0x2105"
		case "eth_call":
			callMu.Lock()
			if callIx < len(callResults) {
				res.Result = callResults[callIx]
				callIx++
			} else {
				res.Result = "0x"
			}
			callMu.Unlock()
		default:
			res.Result = "0x0"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: srv.URL}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeAdapterHyperbridge).Return(&entities.SmartContract{
		ID:              uuid.New(),
		ChainUUID:       sourceID,
		Type:            entities.ContractTypeAdapterHyperbridge,
		ContractAddress: adapter0.Hex(),
		IsActive:        true,
	}, nil)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, blockchain.NewClientFactory(), "")
	status, err := u.GetStatus(context.Background(), "eip155:8453", "eip155:42161")
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, uint8(0), status.DefaultBridgeType)
	require.True(t, status.HasAdapterType0)
	require.False(t, status.HasAdapterType1)
	require.False(t, status.HasAdapterType2)
	require.Equal(t, adapter0.Hex(), status.AdapterType0)
	require.True(t, status.HyperbridgeConfigured)
	require.Contains(t, status.HyperbridgeStateMachineID, "0x")
	require.Contains(t, status.HyperbridgeDestinationContract, "0x")
}

func TestOnchainAdapterUsecase_GetStatus_AllBridgeAdapters(t *testing.T) {
	gatewayABI := mustParseABI(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
	]`)
	routerABI := mustParseABI(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"uint8","name":"","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"uint8","name":"","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`)
	hyperABI := mustParseABI(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
	]`)
	ccipABI := mustParseABI(t, `[
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
	]`)
	layerZeroABI := mustParseABI(t, `[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"isRouteConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"dstEids","outputs":[{"internalType":"uint32","name":"","type":"uint32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"enforcedOptions","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
	]`)

	adapter0 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	adapter1 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	adapter2 := common.HexToAddress("0x3333333333333333333333333333333333333333")
	lzPeer := [32]byte{9}
	ccipDest := common.LeftPadBytes(common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Bytes(), 32)
	hyperDest := common.LeftPadBytes(common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Bytes(), 32)

	callResults := []string{
		encodeOut(t, gatewayABI.Methods["defaultBridgeTypes"], uint8(2)),
		encodeOut(t, routerABI.Methods["hasAdapter"], true),
		encodeOut(t, routerABI.Methods["hasAdapter"], true),
		encodeOut(t, routerABI.Methods["hasAdapter"], true),
		encodeOut(t, routerABI.Methods["getAdapter"], adapter0),
		encodeOut(t, routerABI.Methods["getAdapter"], adapter1),
		encodeOut(t, routerABI.Methods["getAdapter"], adapter2),
		encodeOut(t, routerABI.Methods["hasAdapter"], true),
		encodeOut(t, routerABI.Methods["getAdapter"], adapter2),
		encodeOut(t, hyperABI.Methods["isChainConfigured"], true),
		encodeOut(t, hyperABI.Methods["stateMachineIds"], []byte("EVM-42161")),
		encodeOut(t, hyperABI.Methods["destinationContracts"], hyperDest),
		encodeOut(t, ccipABI.Methods["chainSelectors"], uint64(4949039107694359620)),
		encodeOut(t, ccipABI.Methods["destinationAdapters"], ccipDest),
		encodeOut(t, layerZeroABI.Methods["isRouteConfigured"], true),
		encodeOut(t, layerZeroABI.Methods["dstEids"], uint32(30110)),
		encodeOut(t, layerZeroABI.Methods["peers"], lzPeer),
		encodeOut(t, layerZeroABI.Methods["enforcedOptions"], []byte{0x01, 0x02}),
	}

	var (
		callMu sync.Mutex
		callIx int
	)

	srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req rpcReqOA
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := rpcRespOA{JSONRPC: "2.0", ID: req.ID}

		switch req.Method {
		case "eth_chainId":
			res.Result = "0x2105"
		case "eth_call":
			callMu.Lock()
			if callIx < len(callResults) {
				res.Result = callResults[callIx]
				callIx++
			} else {
				res.Result = "0x"
			}
			callMu.Unlock()
		default:
			res.Result = "0x0"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: srv.URL}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeAdapterHyperbridge).Return(&entities.SmartContract{
		ID:              uuid.New(),
		ChainUUID:       sourceID,
		Type:            entities.ContractTypeAdapterHyperbridge,
		ContractAddress: adapter0.Hex(),
		IsActive:        true,
	}, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeAdapterCCIP).Return(&entities.SmartContract{
		ID:              uuid.New(),
		ChainUUID:       sourceID,
		Type:            entities.ContractTypeAdapterCCIP,
		ContractAddress: adapter1.Hex(),
		IsActive:        true,
	}, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeAdapterLayerZero).Return(&entities.SmartContract{
		ID:              uuid.New(),
		ChainUUID:       sourceID,
		Type:            entities.ContractTypeAdapterLayerZero,
		ContractAddress: adapter2.Hex(),
		IsActive:        true,
	}, nil)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, blockchain.NewClientFactory(), "")
	status, err := u.GetStatus(context.Background(), "eip155:8453", "eip155:42161")
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, uint8(2), status.DefaultBridgeType)
	require.True(t, status.HyperbridgeConfigured)
	require.NotEmpty(t, status.CCIPDestinationAdapter)
	require.Equal(t, uint64(4949039107694359620), status.CCIPChainSelector)
	require.True(t, status.LayerZeroConfigured)
	require.Equal(t, uint32(30110), status.LayerZeroDstEID)
	require.NotEqual(t, "0x", status.LayerZeroPeer)
	require.NotEmpty(t, status.LayerZeroOptionsHex)
}
