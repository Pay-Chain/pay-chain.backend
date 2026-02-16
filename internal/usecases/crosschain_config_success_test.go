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
	"pay-chain.backend/pkg/utils"
)

type rpcReqCC struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type rpcRespCC struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
}

func mustParseABIcc(t *testing.T, raw string) abi.ABI {
	t.Helper()
	parsed, err := abi.JSON(strings.NewReader(raw))
	require.NoError(t, err)
	return parsed
}

func encOutCC(t *testing.T, method abi.Method, values ...interface{}) string {
	t.Helper()
	out, err := method.Outputs.Pack(values...)
	require.NoError(t, err)
	return "0x" + hex.EncodeToString(out)
}

func TestCrosschainConfigUsecase_RecheckRoute_StatusAndFeeQuoteIssue(t *testing.T) {
	gatewayABI := mustParseABIcc(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
	]`)
	routerABI := mustParseABIcc(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"uint8","name":"","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"uint8","name":"","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`)
	hyperABI := mustParseABIcc(t, `[
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
	]`)

	adapter0 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	stateMachine := []byte("eip155:42161")
	destinationContract := common.LeftPadBytes(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes(), 32)

	callResults := []string{
		encOutCC(t, gatewayABI.Methods["defaultBridgeTypes"], uint8(0)),
		encOutCC(t, routerABI.Methods["hasAdapter"], true),
		encOutCC(t, routerABI.Methods["hasAdapter"], false),
		encOutCC(t, routerABI.Methods["hasAdapter"], false),
		encOutCC(t, routerABI.Methods["getAdapter"], adapter0),
		encOutCC(t, routerABI.Methods["getAdapter"], common.Address{}),
		encOutCC(t, routerABI.Methods["getAdapter"], common.Address{}),
		encOutCC(t, routerABI.Methods["hasAdapter"], true),
		encOutCC(t, routerABI.Methods["getAdapter"], adapter0),
		encOutCC(t, hyperABI.Methods["isChainConfigured"], true),
		encOutCC(t, hyperABI.Methods["stateMachineIds"], stateMachine),
		encOutCC(t, hyperABI.Methods["destinationContracts"], destinationContract),
		"0x", // quotePaymentFee fallback to unhealthy
	}

	var (
		mu sync.Mutex
		ix int
	)
	srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req rpcReqCC
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := rpcRespCC{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "eth_chainId":
			res.Result = "0x2105"
		case "eth_call":
			mu.Lock()
			if ix < len(callResults) {
				res.Result = callResults[ix]
				ix++
			} else {
				res.Result = "0x"
			}
			mu.Unlock()
		default:
			res.Result = "0x0"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
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
	chainRepo.On("GetAll", mock.Anything).Return([]*entities.Chain{source, dest}, nil)

	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)

	tokenRepo.On("GetTokensByChain", mock.Anything, sourceID, mock.Anything).Return([]*entities.Token{}, int64(0), nil)
	tokenRepo.On("GetTokensByChain", mock.Anything, destID, mock.Anything).Return([]*entities.Token{}, int64(0), nil)

	factory := blockchain.NewClientFactory()
	adapterUsecase := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, factory, "")
	u := uc.NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, factory, adapterUsecase)

	status, err := u.RecheckRoute(context.Background(), "eip155:8453", "eip155:42161")
	require.NoError(t, err)
	require.NotNil(t, status)
	require.True(t, status.AdapterRegistered)
	require.True(t, status.HyperbridgeConfigured)
	require.False(t, status.FeeQuoteHealthy)
	require.Equal(t, "ERROR", status.OverallStatus)

	codes := make([]string, 0, len(status.Issues))
	for _, issue := range status.Issues {
		codes = append(codes, issue.Code)
	}
	require.Contains(t, codes, "FEE_QUOTE_FAILED")

	_, err = u.Overview(context.Background(), "eip155:8453", "eip155:42161", utils.PaginationParams{Page: 1, Limit: 20})
	require.NoError(t, err)
}
