package usecases

import (
		"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"

	"github.com/volatiletech/null/v8"
)

func TestRequiredFunctions(t *testing.T) {
	t.Run("gateway", func(t *testing.T) {
		require.Contains(t, requiredFunctions(entities.ContractTypeGateway), "createPayment")
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
