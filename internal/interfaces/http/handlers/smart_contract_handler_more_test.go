package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestSmartContractHandler_UpdateAdditionalBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contractID := uuid.New()
	chainUUID := uuid.New()
	legacyChainUUID := uuid.New()

	repo := &smartContractRepoStub{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.SmartContract, error) {
			return &entities.SmartContract{
				ID:              contractID,
				Name:            "Gateway",
				ChainUUID:       chainUUID,
				ContractAddress: "0xabc",
				Type:            entities.ContractTypeGateway,
			}, nil
		},
		updateFn: func(_ context.Context, contract *entities.SmartContract) error {
			if contract.ChainUUID == legacyChainUUID {
				return errors.New("update failed")
			}
			return nil
		},
	}
	chainRepo := &smartContractChainRepoStub{
		getByChainIDFn: func(_ context.Context, raw string) (*entities.Chain, error) {
			if raw == "8453" {
				return &entities.Chain{ID: legacyChainUUID}, nil
			}
			return nil, domainerrors.ErrNotFound
		},
	}
	h := NewSmartContractHandler(repo, chainRepo)

	r := gin.New()
	r.PUT("/contracts/:id", h.UpdateSmartContract)
	r.GET("/contracts/lookup", h.GetContractByChainAndAddress)

	req := httptest.NewRequest(http.MethodPut, "/contracts/"+contractID.String(), strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/contracts/"+contractID.String(), strings.NewReader(`{"chainId":"unknown-chain"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/contracts/"+contractID.String(), strings.NewReader(`{"chainId":"8453","name":"Gateway v2"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/contracts/lookup?chainId=8453&address=0xabc", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSmartContractHandler_UpdateAllMutableFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contractID := uuid.New()
	chainUUID := uuid.New()
	newChainUUID := uuid.New()

	var updated *entities.SmartContract
	repo := &smartContractRepoStub{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.SmartContract, error) {
			return &entities.SmartContract{
				ID:              contractID,
				Name:            "Gateway",
				ChainUUID:       chainUUID,
				ContractAddress: "0xabc",
				Type:            entities.ContractTypeGateway,
				Version:         "1.0.0",
				IsActive:        true,
			}, nil
		},
		updateFn: func(_ context.Context, contract *entities.SmartContract) error {
			updated = contract
			return nil
		},
	}

	h := NewSmartContractHandler(repo, &smartContractChainRepoStub{})
	r := gin.New()
	r.PUT("/contracts/:id", h.UpdateSmartContract)

	body := `{
		"name":"Gateway v3",
		"type":"ROUTER",
		"version":"3.0.0",
		"chainId":"` + newChainUUID.String() + `",
		"contractAddress":"0x123",
		"deployerAddress":"0xdeployer",
		"token0Address":"0xt0",
		"token1Address":"0xt1",
		"feeTier":3000,
		"hookAddress":"0xhook",
		"startBlock":321,
		"abi":[{"name":"x","type":"function"}],
		"isActive":false,
		"metadata":{"k":"v"}
	}`
	req := httptest.NewRequest(http.MethodPut, "/contracts/"+contractID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, updated)
	require.Equal(t, "Gateway v3", updated.Name)
	require.Equal(t, entities.ContractTypeRouter, updated.Type)
	require.Equal(t, "3.0.0", updated.Version)
	require.Equal(t, newChainUUID, updated.ChainUUID)
	require.Equal(t, "0x123", updated.ContractAddress)
	require.True(t, updated.DeployerAddress.Valid)
	require.True(t, updated.Token0Address.Valid)
	require.True(t, updated.Token1Address.Valid)
	require.True(t, updated.FeeTier.Valid)
	require.Equal(t, 3000, updated.FeeTier.Int)
	require.True(t, updated.HookAddress.Valid)
	require.Equal(t, uint64(321), updated.StartBlock)
	require.False(t, updated.IsActive)
	require.True(t, updated.Metadata.Valid)
}

func TestSmartContractHandler_Create_WithOptionalFieldsAndUUIDChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainUUID := uuid.New()
	var created *entities.SmartContract

	repo := &smartContractRepoStub{
		createFn: func(_ context.Context, contract *entities.SmartContract) error {
			created = contract
			return nil
		},
	}
	h := NewSmartContractHandler(repo, &smartContractChainRepoStub{})
	r := gin.New()
	r.POST("/contracts", h.CreateSmartContract)

	body := `{
		"name":"Router",
		"type":"ROUTER",
		"version":"2.0.0",
		"chainId":"` + chainUUID.String() + `",
		"contractAddress":"0xrouter",
		"deployerAddress":"0xdeployer",
		"token0Address":"0xt0",
		"token1Address":"0xt1",
		"feeTier":500,
		"hookAddress":"0xhook",
		"startBlock":10,
		"abi":[{"name":"quote","type":"function"}],
		"isActive":false,
		"metadata":{"x":"y"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/contracts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	require.NotNil(t, created)
	require.Equal(t, chainUUID, created.ChainUUID)
	require.Equal(t, "Router", created.Name)
	require.Equal(t, entities.ContractTypeRouter, created.Type)
	require.False(t, created.IsActive)
	require.True(t, created.DeployerAddress.Valid)
	require.True(t, created.Token0Address.Valid)
	require.True(t, created.Token1Address.Valid)
	require.True(t, created.FeeTier.Valid)
	require.True(t, created.HookAddress.Valid)
	require.True(t, created.Metadata.Valid)
}
