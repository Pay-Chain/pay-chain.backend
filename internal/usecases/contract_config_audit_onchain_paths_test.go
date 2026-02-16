package usecases

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

type auditRPCReq struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      interface{} `json:"id"`
}

func encodeMethodOutput(t *testing.T, rawABI, method string, values ...interface{}) string {
	t.Helper()
	parsed, err := abi.JSON(stringsReader(rawABI))
	require.NoError(t, err)
	out, err := parsed.Methods[method].Outputs.Pack(values...)
	require.NoError(t, err)
	return "0x" + hex.EncodeToString(out)
}

func TestRunEVMOnchainChecks_HyperbridgeConfiguredPath(t *testing.T) {
	defaultBridgeABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}]`
	hasAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`
	getAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
	hyperConfiguredABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"}]`

	adapterAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	callResults := []string{
		encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(0)),
		encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true),
		encodeMethodOutput(t, getAdapterABI, "getAdapter", adapterAddr),
		encodeMethodOutput(t, hyperConfiguredABI, "isChainConfigured", true),
	}
	callIdx := 0

	srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req auditRPCReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]interface{}{"jsonrpc": "2.0", "id": req.ID}
		if req.Method == "eth_chainId" {
			res["result"] = "0x2105"
		} else if req.Method == "eth_call" {
			if callIdx < len(callResults) {
				res["result"] = callResults[callIdx]
				callIdx++
			} else {
				res["result"] = "0x"
			}
		} else {
			res["result"] = "0x0"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	sourceID := uuid.New()
	usecase := &ContractConfigAuditUsecase{clientFactory: blockchain.NewClientFactory()}
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: srv.URL}
	contracts := []*entities.SmartContract{
		{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true},
		{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true},
	}

	checks := usecase.runEVMOnchainChecks(context.Background(), source, contracts, "eip155:42161")
	codes := map[string]bool{}
	for _, c := range checks {
		codes[c.Code] = true
	}
	require.True(t, codes["DEFAULT_BRIDGE_TYPE"])
	require.True(t, codes["ADAPTER_REGISTERED"])
	require.True(t, codes["ADAPTER_ADDRESS_FOUND"])
	require.True(t, codes["HYPERBRIDGE_CHAIN_CONFIGURED"])
}

func TestRunEVMOnchainChecks_CcipConfiguredPath(t *testing.T) {
	defaultBridgeABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}]`
	hasAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`
	getAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
	ccipSelectorABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"}]`
	ccipDestABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`

	adapterAddr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	callResults := []string{
		encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(1)),
		encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true),
		encodeMethodOutput(t, getAdapterABI, "getAdapter", adapterAddr),
		encodeMethodOutput(t, ccipSelectorABI, "chainSelectors", uint64(4949039107694359620)),
		encodeMethodOutput(t, ccipDestABI, "destinationAdapters", []byte{0x01, 0x02, 0x03}),
	}
	callIdx := 0

	srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req auditRPCReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]interface{}{"jsonrpc": "2.0", "id": req.ID}
		if req.Method == "eth_chainId" {
			res["result"] = "0x2105"
		} else if req.Method == "eth_call" {
			if callIdx < len(callResults) {
				res["result"] = callResults[callIdx]
				callIdx++
			} else {
				res["result"] = "0x"
			}
		} else {
			res["result"] = "0x0"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	sourceID := uuid.New()
	usecase := &ContractConfigAuditUsecase{clientFactory: blockchain.NewClientFactory()}
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: srv.URL}
	contracts := []*entities.SmartContract{
		{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true},
		{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true},
	}

	checks := usecase.runEVMOnchainChecks(context.Background(), source, contracts, "eip155:42161")
	codes := map[string]bool{}
	for _, c := range checks {
		codes[c.Code] = true
	}
	require.True(t, codes["DEFAULT_BRIDGE_TYPE"])
	require.True(t, codes["ADAPTER_REGISTERED"])
	require.True(t, codes["ADAPTER_ADDRESS_FOUND"])
	require.True(t, codes["CCIP_SELECTOR_CONFIGURED"])
	require.True(t, codes["CCIP_DEST_ADAPTER_CONFIGURED"])
}

