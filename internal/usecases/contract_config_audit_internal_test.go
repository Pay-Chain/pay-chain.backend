package usecases

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"

	"github.com/volatiletech/null/v8"
)

func TestRequiredFunctions(t *testing.T) {
	t.Run("gateway", func(t *testing.T) {
		require.Contains(t, requiredFunctions(entities.ContractTypeGateway), "createPayment")
	})
	t.Run("router", func(t *testing.T) {
		require.Contains(t, requiredFunctions(entities.ContractTypeRouter), "quotePaymentFee")
	})
	t.Run("vault", func(t *testing.T) {
		require.Contains(t, requiredFunctions(entities.ContractTypeVault), "pushTokens")
	})
	t.Run("token registry", func(t *testing.T) {
		require.Contains(t, requiredFunctions(entities.ContractTypeTokenRegistry), "isSupportedToken")
	})
	t.Run("token swapper", func(t *testing.T) {
		require.Contains(t, requiredFunctions(entities.ContractTypeTokenSwapper), "swap")
	})
	t.Run("ccip adapter", func(t *testing.T) {
		require.Contains(t, requiredFunctions(entities.ContractTypeAdapterCCIP), "setChainSelector")
	})
	t.Run("hyperbridge adapter", func(t *testing.T) {
		require.Contains(t, requiredFunctions(entities.ContractTypeAdapterHyperbridge), "setStateMachineId")
	})
	t.Run("layerzero adapter", func(t *testing.T) {
		require.Contains(t, requiredFunctions(entities.ContractTypeAdapterLayerZero), "setRoute")
	})
	t.Run("unknown", func(t *testing.T) {
		require.Empty(t, requiredFunctions(entities.SmartContractType("UNKNOWN")))
	})
}

func TestExtractFunctionNames(t *testing.T) {
	abiEntries := []interface{}{
		map[string]interface{}{"type": "function", "name": "setFoo"},
		map[string]interface{}{"type": "function", "name": "setFoo"},
		map[string]interface{}{"type": "event", "name": "Ignored"},
		map[string]interface{}{"type": "function", "name": " quoteFee "},
	}

	names := extractFunctionNames(abiEntries)
	require.ElementsMatch(t, []string{"setFoo", "quoteFee"}, names)
	require.Empty(t, extractFunctionNames("invalid"))
}

func TestGenerateFieldsFromFunctions(t *testing.T) {
	fields := generateFieldsFromFunctions([]string{"setRoute", "setPeer", "quoteFee", "setRoute"})
	require.Equal(t, []string{"peer", "route"}, fields)
}

func TestFindActiveContractByType(t *testing.T) {
	contracts := []*entities.SmartContract{
		{Type: entities.ContractTypeGateway, IsActive: false},
		{Type: entities.ContractTypeRouter, IsActive: true},
	}
	found := findActiveContractByType(contracts, entities.ContractTypeRouter)
	require.NotNil(t, found)
	require.Equal(t, entities.ContractTypeRouter, found.Type)
	require.Nil(t, findActiveContractByType(contracts, entities.ContractTypeVault))
}

func TestMergeSummaryAndDeriveOverallStatus(t *testing.T) {
	summary := map[string]int{"ok": 0, "warn": 0, "error": 0}
	mergeSummary(summary, []ContractConfigCheckItem{
		{Status: "OK"},
		{Status: "warn"},
		{Status: "ERROR"},
	})
	require.Equal(t, 1, summary["ok"])
	require.Equal(t, 1, summary["warn"])
	require.Equal(t, 1, summary["error"])
	require.Equal(t, "ERROR", deriveOverallStatus(summary))
	require.Equal(t, "WARN", deriveOverallStatus(map[string]int{"ok": 1, "warn": 1, "error": 0}))
	require.Equal(t, "OK", deriveOverallStatus(map[string]int{"ok": 1, "warn": 0, "error": 0}))
}

func TestBuildContractReport(t *testing.T) {
	u := &ContractConfigAuditUsecase{}
	contract := &entities.SmartContract{
		ID:              uuid.New(),
		Name:            "Gateway X",
		Type:            entities.ContractTypeGateway,
		ContractAddress: "",
		StartBlock:      0,
		IsActive:        true,
		ABI: []interface{}{
			map[string]interface{}{"type": "function", "name": "createPayment"},
			map[string]interface{}{"type": "function", "name": "setDefaultBridgeType"},
		},
	}

	report := u.buildContractReport(contract)
	require.Equal(t, "Gateway X", report.Name)
	require.NotEmpty(t, report.MissingFunctions)
	require.NotEmpty(t, report.Checks)

	pool := &entities.SmartContract{
		ID:              uuid.New(),
		Name:            "Pool",
		Type:            entities.ContractTypePool,
		ContractAddress: "0xabc",
		IsActive:        true,
		Token0Address:   null.String{Valid: false},
		Token1Address:   null.String{Valid: false},
	}
	poolReport := u.buildContractReport(pool)
	codes := make([]string, 0, len(poolReport.Checks))
	for _, c := range poolReport.Checks {
		codes = append(codes, c.Code)
	}
	require.Contains(t, codes, "POOL_TOKEN_PAIR_MISSING")
}

func TestRunEVMOnchainChecks_RPCMissing(t *testing.T) {
	u := &ContractConfigAuditUsecase{}
	source := &entities.Chain{ID: uuid.New(), ChainID: "8453", Type: entities.ChainTypeEVM}

	checks := u.runEVMOnchainChecks(context.Background(), source, nil, "eip155:42161")
	require.Len(t, checks, 1)
	require.Equal(t, "RPC_MISSING", checks[0].Code)
}

func TestRunEVMOnchainChecks_RPCConnectFailed(t *testing.T) {
	u := &ContractConfigAuditUsecase{
		clientFactory: blockchain.NewClientFactory(),
	}
	source := &entities.Chain{
		ID:      uuid.New(),
		ChainID: "8453",
		Type:    entities.ChainTypeEVM,
		RPCURL:  "http://127.0.0.1:0",
	}

	checks := u.runEVMOnchainChecks(context.Background(), source, nil, "eip155:42161")
	require.NotEmpty(t, checks)
	require.Equal(t, "RPC_CONNECT_FAILED", checks[0].Code)
}

func TestRunEVMOnchainChecks_GatewayRouterMissing(t *testing.T) {
	srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
		if req["method"] == "eth_chainId" {
			res["result"] = "0x2105"
		} else {
			res["result"] = "0x"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	u := &ContractConfigAuditUsecase{
		clientFactory: blockchain.NewClientFactory(),
	}
	source := &entities.Chain{
		ID:      uuid.New(),
		ChainID: "8453",
		Type:    entities.ChainTypeEVM,
		RPCURL:  srv.URL,
	}

	checks := u.runEVMOnchainChecks(context.Background(), source, []*entities.SmartContract{}, "eip155:42161")
	require.Len(t, checks, 2)
	require.Equal(t, "GATEWAY_MISSING", checks[0].Code)
	require.Equal(t, "ROUTER_MISSING", checks[1].Code)
}

func TestRunEVMOnchainChecks_DefaultBridgeReadFailed(t *testing.T) {
	srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
		if req["method"] == "eth_chainId" {
			res["result"] = "0x2105"
		} else {
			// return empty output to trigger decode error in defaultBridgeTypes
			res["result"] = "0x"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	sourceID := uuid.New()
	u := &ContractConfigAuditUsecase{
		clientFactory: blockchain.NewClientFactory(),
	}
	source := &entities.Chain{
		ID:      sourceID,
		ChainID: "8453",
		Type:    entities.ChainTypeEVM,
		RPCURL:  srv.URL,
	}
	contracts := []*entities.SmartContract{
		{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true},
		{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true},
	}

	checks := u.runEVMOnchainChecks(context.Background(), source, contracts, "eip155:42161")
	require.NotEmpty(t, checks)
	require.Equal(t, "DEFAULT_BRIDGE_READ_FAILED", checks[len(checks)-1].Code)
}
