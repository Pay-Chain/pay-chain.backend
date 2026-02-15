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

type smartContractRepoStub struct {
	createFn            func(ctx context.Context, contract *entities.SmartContract) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*entities.SmartContract, error)
	getFilteredFn       func(ctx context.Context, chainID *uuid.UUID, contractType entities.SmartContractType, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error)
	getByChainAddressFn func(ctx context.Context, chainID uuid.UUID, address string) (*entities.SmartContract, error)
	updateFn            func(ctx context.Context, contract *entities.SmartContract) error
	softDeleteFn        func(ctx context.Context, id uuid.UUID) error
}

func (s *smartContractRepoStub) Create(ctx context.Context, contract *entities.SmartContract) error {
	if s.createFn != nil {
		return s.createFn(ctx, contract)
	}
	return nil
}

func (s *smartContractRepoStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.SmartContract, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, domainerrors.ErrNotFound
}

func (s *smartContractRepoStub) GetByChainAndAddress(ctx context.Context, chainID uuid.UUID, address string) (*entities.SmartContract, error) {
	if s.getByChainAddressFn != nil {
		return s.getByChainAddressFn(ctx, chainID, address)
	}
	return nil, domainerrors.ErrNotFound
}

func (s *smartContractRepoStub) GetActiveContract(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}

func (s *smartContractRepoStub) GetFiltered(ctx context.Context, chainID *uuid.UUID, contractType entities.SmartContractType, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	if s.getFilteredFn != nil {
		return s.getFilteredFn(ctx, chainID, contractType, pagination)
	}
	return []*entities.SmartContract{}, 0, nil
}

func (s *smartContractRepoStub) GetByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return []*entities.SmartContract{}, 0, nil
}

func (s *smartContractRepoStub) GetAll(context.Context, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return []*entities.SmartContract{}, 0, nil
}

func (s *smartContractRepoStub) Update(ctx context.Context, contract *entities.SmartContract) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, contract)
	}
	return nil
}

func (s *smartContractRepoStub) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if s.softDeleteFn != nil {
		return s.softDeleteFn(ctx, id)
	}
	return nil
}

type smartContractChainRepoStub struct {
	getByChainIDFn func(ctx context.Context, chainID string) (*entities.Chain, error)
}

func (s *smartContractChainRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *smartContractChainRepoStub) GetByChainID(ctx context.Context, chainID string) (*entities.Chain, error) {
	if s.getByChainIDFn != nil {
		return s.getByChainIDFn(ctx, chainID)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *smartContractChainRepoStub) GetByCAIP2(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *smartContractChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) { return nil, nil }
func (s *smartContractChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *smartContractChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *smartContractChainRepoStub) Create(context.Context, *entities.Chain) error { return nil }
func (s *smartContractChainRepoStub) Update(context.Context, *entities.Chain) error { return nil }
func (s *smartContractChainRepoStub) Delete(context.Context, uuid.UUID) error       { return nil }

func TestSmartContractHandler_CRUDAndLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainID := uuid.New()
	contractID := uuid.New()

	repo := &smartContractRepoStub{
		createFn: func(_ context.Context, contract *entities.SmartContract) error {
			contract.ID = contractID
			return nil
		},
		getByIDFn: func(_ context.Context, id uuid.UUID) (*entities.SmartContract, error) {
			if id != contractID {
				return nil, domainerrors.ErrNotFound
			}
			return &entities.SmartContract{ID: contractID, Name: "Gateway", ChainUUID: chainID}, nil
		},
		getFilteredFn: func(_ context.Context, chainUUID *uuid.UUID, contractType entities.SmartContractType, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
			require.NotNil(t, chainUUID)
			require.Equal(t, chainID, *chainUUID)
			require.Equal(t, entities.ContractTypeGateway, contractType)
			require.Equal(t, 1, pagination.Page)
			require.Equal(t, 10, pagination.Limit)
			return []*entities.SmartContract{{ID: contractID, Name: "Gateway"}}, 1, nil
		},
		getByChainAddressFn: func(_ context.Context, id uuid.UUID, address string) (*entities.SmartContract, error) {
			require.Equal(t, chainID, id)
			require.Equal(t, "0xabc", address)
			return &entities.SmartContract{ID: contractID, ContractAddress: address, ChainUUID: id}, nil
		},
		updateFn: func(_ context.Context, contract *entities.SmartContract) error {
			require.Equal(t, "Gateway v2", contract.Name)
			require.Equal(t, chainID, contract.ChainUUID)
			return nil
		},
		softDeleteFn: func(_ context.Context, id uuid.UUID) error {
			require.Equal(t, contractID, id)
			return nil
		},
	}
	chainRepo := &smartContractChainRepoStub{
		getByChainIDFn: func(_ context.Context, raw string) (*entities.Chain, error) {
			if raw == "8453" {
				return &entities.Chain{ID: chainID, ChainID: "8453"}, nil
			}
			return nil, domainerrors.ErrNotFound
		},
	}
	h := NewSmartContractHandler(repo, chainRepo)

	r := gin.New()
	r.POST("/contracts", h.CreateSmartContract)
	r.GET("/contracts/:id", h.GetSmartContract)
	r.GET("/contracts", h.ListSmartContracts)
	r.GET("/contracts/lookup", h.GetContractByChainAndAddress)
	r.PUT("/contracts/:id", h.UpdateSmartContract)
	r.DELETE("/contracts/:id", h.DeleteSmartContract)

	createBody := `{"name":"Gateway","type":"GATEWAY","version":"1.0.0","chainId":"8453","contractAddress":"0xabc","startBlock":1,"abi":[{"name":"x"}]}`
	req := httptest.NewRequest(http.MethodPost, "/contracts", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "Gateway")

	req = httptest.NewRequest(http.MethodGet, "/contracts/"+contractID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Gateway")

	req = httptest.NewRequest(http.MethodGet, "/contracts?page=1&limit=10&chainId=8453&type=gateway", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"items\"")

	req = httptest.NewRequest(http.MethodGet, "/contracts/lookup?chainId=8453&address=0xabc", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "0xabc")

	updateBody := `{"name":"Gateway v2","chainId":"8453"}`
	req = httptest.NewRequest(http.MethodPut, "/contracts/"+contractID.String(), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "updated")

	req = httptest.NewRequest(http.MethodDelete, "/contracts/"+contractID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestSmartContractHandler_ValidationAndErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainID := uuid.New()
	contractID := uuid.New()
	repo := &smartContractRepoStub{
		createFn: func(context.Context, *entities.SmartContract) error { return errors.New("db error") },
		getByIDFn: func(context.Context, uuid.UUID) (*entities.SmartContract, error) {
			return nil, domainerrors.ErrNotFound
		},
		getFilteredFn: func(context.Context, *uuid.UUID, entities.SmartContractType, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
			return nil, 0, errors.New("list failed")
		},
		getByChainAddressFn: func(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
			return nil, domainerrors.ErrNotFound
		},
		softDeleteFn: func(context.Context, uuid.UUID) error {
			return domainerrors.ErrNotFound
		},
	}
	chainRepo := &smartContractChainRepoStub{
		getByChainIDFn: func(_ context.Context, raw string) (*entities.Chain, error) {
			if raw == "ok-chain" {
				return &entities.Chain{ID: chainID}, nil
			}
			return nil, domainerrors.ErrNotFound
		},
	}
	h := NewSmartContractHandler(repo, chainRepo)
	r := gin.New()
	r.POST("/contracts", h.CreateSmartContract)
	r.GET("/contracts/:id", h.GetSmartContract)
	r.GET("/contracts", h.ListSmartContracts)
	r.GET("/contracts/lookup", h.GetContractByChainAndAddress)
	r.PUT("/contracts/:id", h.UpdateSmartContract)
	r.DELETE("/contracts/:id", h.DeleteSmartContract)

	req := httptest.NewRequest(http.MethodPost, "/contracts", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	createBody := `{"name":"Gateway","type":"GATEWAY","version":"1.0.0","chainId":"ok-chain","contractAddress":"0xabc","startBlock":1,"abi":[{"name":"x"}]}`
	req = httptest.NewRequest(http.MethodPost, "/contracts", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/contracts/not-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/contracts/"+contractID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/contracts?chainId=bad-chain", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/contracts/lookup", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/contracts/lookup?chainId=bad-chain&address=0xabc", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/contracts/"+contractID.String(), strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/contracts/not-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/contracts/"+contractID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}
