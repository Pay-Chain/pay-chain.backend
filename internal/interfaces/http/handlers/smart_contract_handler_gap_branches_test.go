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
	"pay-chain.backend/pkg/utils"
)

func TestSmartContractHandler_Create_InvalidChainIDBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &smartContractRepoStub{
		createFn: func(context.Context, *entities.SmartContract) error {
			t.Fatal("create should not be called on invalid chain")
			return nil
		},
	}
	chainRepo := &smartContractChainRepoStub{
		getByChainIDFn: func(context.Context, string) (*entities.Chain, error) {
			return nil, domainerrors.ErrNotFound
		},
	}
	h := NewSmartContractHandler(repo, chainRepo)

	r := gin.New()
	r.POST("/contracts", h.CreateSmartContract)

	body := `{"name":"Gateway","type":"GATEWAY","version":"1.0.0","chainId":"bad-chain","contractAddress":"0xabc","startBlock":1,"abi":[{"name":"x"}]}`
	req := httptest.NewRequest(http.MethodPost, "/contracts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSmartContractHandler_ListAndLookup_AdditionalErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainUUID := uuid.New()

	repo := &smartContractRepoStub{
		getFilteredFn: func(_ context.Context, chainID *uuid.UUID, _ entities.SmartContractType, _ utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
			require.Nil(t, chainID, "invalid chainId that cannot be resolved should keep chain filter nil")
			return []*entities.SmartContract{}, 0, nil
		},
		getByChainAddressFn: func(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
			return nil, errors.New("lookup failed")
		},
	}
	chainRepo := &smartContractChainRepoStub{
		getByChainIDFn: func(_ context.Context, raw string) (*entities.Chain, error) {
			if raw == "8453" {
				return &entities.Chain{ID: chainUUID, ChainID: "8453"}, nil
			}
			return nil, domainerrors.ErrNotFound
		},
	}
	h := NewSmartContractHandler(repo, chainRepo)

	r := gin.New()
	r.GET("/contracts", h.ListSmartContracts)
	r.GET("/contracts/lookup", h.GetContractByChainAndAddress)

	req := httptest.NewRequest(http.MethodGet, "/contracts?chainId=not-caip2-or-uuid&type=router", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/contracts/lookup?chainId=8453&address=0xabc", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSmartContractHandler_ListSmartContracts_UUIDChainFilterAndEmptyType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainUUID := uuid.New()

	repo := &smartContractRepoStub{
		getFilteredFn: func(_ context.Context, chainID *uuid.UUID, contractType entities.SmartContractType, _ utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
			require.NotNil(t, chainID)
			require.Equal(t, chainUUID, *chainID)
			require.Equal(t, entities.SmartContractType(""), contractType)
			return []*entities.SmartContract{}, 0, nil
		},
	}
	h := NewSmartContractHandler(repo, &smartContractChainRepoStub{})
	r := gin.New()
	r.GET("/contracts", h.ListSmartContracts)

	req := httptest.NewRequest(http.MethodGet, "/contracts?chainId="+chainUUID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
