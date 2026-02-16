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
