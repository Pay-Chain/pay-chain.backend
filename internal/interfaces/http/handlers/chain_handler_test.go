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

type chainHandlerRepoStub struct {
	getActiveFn  func(ctx context.Context, pagination utils.PaginationParams) ([]*entities.Chain, int64, error)
	getByChainID func(ctx context.Context, chainID string) (*entities.Chain, error)
	createFn     func(ctx context.Context, chain *entities.Chain) error
	updateFn     func(ctx context.Context, chain *entities.Chain) error
	deleteFn     func(ctx context.Context, id uuid.UUID) error
	getByIDFn    func(ctx context.Context, id uuid.UUID) (*entities.Chain, error)
	getByCAIP2Fn func(ctx context.Context, caip2 string) (*entities.Chain, error)
	getAllFn     func(ctx context.Context) ([]*entities.Chain, error)
	getAllRPCsFn func(ctx context.Context, chainID *uuid.UUID, isActive *bool, search *string, pagination utils.PaginationParams) ([]*entities.ChainRPC, int64, error)
}

func (s *chainHandlerRepoStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.Chain, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, domainerrors.ErrNotFound
}

func (s *chainHandlerRepoStub) GetByChainID(ctx context.Context, chainID string) (*entities.Chain, error) {
	if s.getByChainID != nil {
		return s.getByChainID(ctx, chainID)
	}
	return nil, domainerrors.ErrNotFound
}

func (s *chainHandlerRepoStub) GetByCAIP2(ctx context.Context, caip2 string) (*entities.Chain, error) {
	if s.getByCAIP2Fn != nil {
		return s.getByCAIP2Fn(ctx, caip2)
	}
	return nil, domainerrors.ErrNotFound
}

func (s *chainHandlerRepoStub) GetAll(ctx context.Context) ([]*entities.Chain, error) {
	if s.getAllFn != nil {
		return s.getAllFn(ctx)
	}
	return []*entities.Chain{}, nil
}

func (s *chainHandlerRepoStub) GetAllRPCs(ctx context.Context, chainID *uuid.UUID, isActive *bool, search *string, pagination utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	if s.getAllRPCsFn != nil {
		return s.getAllRPCsFn(ctx, chainID, isActive, search, pagination)
	}
	return []*entities.ChainRPC{}, 0, nil
}

func (s *chainHandlerRepoStub) GetActive(ctx context.Context, pagination utils.PaginationParams) ([]*entities.Chain, int64, error) {
	if s.getActiveFn != nil {
		return s.getActiveFn(ctx, pagination)
	}
	return []*entities.Chain{}, 0, nil
}

func (s *chainHandlerRepoStub) Create(ctx context.Context, chain *entities.Chain) error {
	if s.createFn != nil {
		return s.createFn(ctx, chain)
	}
	return nil
}

func (s *chainHandlerRepoStub) Update(ctx context.Context, chain *entities.Chain) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, chain)
	}
	return nil
}

func (s *chainHandlerRepoStub) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

func (s *chainHandlerRepoStub) CreateRPC(ctx context.Context, rpc *entities.ChainRPC) error {
	return nil
}
func (s *chainHandlerRepoStub) UpdateRPC(ctx context.Context, rpc *entities.ChainRPC) error {
	return nil
}
func (s *chainHandlerRepoStub) DeleteRPC(ctx context.Context, id uuid.UUID) error { return nil }
func (s *chainHandlerRepoStub) GetRPCByID(ctx context.Context, id uuid.UUID) (*entities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}

func TestChainHandler_ListChains_SuccessAndError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	chainID := uuid.New()
	repo := &chainHandlerRepoStub{
		getActiveFn: func(_ context.Context, pagination utils.PaginationParams) ([]*entities.Chain, int64, error) {
			if pagination.Page == 2 {
				return nil, 0, errors.New("db fail")
			}
			return []*entities.Chain{
				{
					ID:             chainID,
					ChainID:        "8453",
					Name:           "Base",
					Type:           entities.ChainTypeEVM,
					RPCURL:         "https://rpc",
					ExplorerURL:    "https://scan",
					CurrencySymbol: "ETH",
					ImageURL:       "logo.png",
					IsActive:       true,
				},
			}, 1, nil
		},
	}
	h := NewChainHandler(repo)

	r := gin.New()
	r.GET("/chains", h.ListChains)

	req := httptest.NewRequest(http.MethodGet, "/chains?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"caip2\":\"eip155:8453\"")

	req = httptest.NewRequest(http.MethodGet, "/chains?page=2&limit=10", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestChainHandler_ListChains_EmptyItemsBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &chainHandlerRepoStub{
		getActiveFn: func(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
			return nil, 0, nil
		},
	}
	h := NewChainHandler(repo)
	r := gin.New()
	r.GET("/chains", h.ListChains)

	req := httptest.NewRequest(http.MethodGet, "/chains", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"items\":[]")
}

func TestChainHandler_CreateUpdateDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainID := uuid.New()

	repo := &chainHandlerRepoStub{
		createFn: func(_ context.Context, chain *entities.Chain) error {
			if chain.Name == "fail-create" {
				return errors.New("create failed")
			}
			return nil
		},
		updateFn: func(_ context.Context, chain *entities.Chain) error {
			switch chain.Name {
			case "missing":
				return domainerrors.ErrNotFound
			case "fail-update":
				return errors.New("update failed")
			default:
				return nil
			}
		},
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			if id == chainID {
				return nil
			}
			if id.String()[0:1] == "a" {
				return domainerrors.ErrNotFound
			}
			return errors.New("delete failed")
		},
	}
	h := NewChainHandler(repo)

	r := gin.New()
	r.POST("/admin/chains", h.CreateChain)
	r.PUT("/admin/chains/:id", h.UpdateChain)
	r.DELETE("/admin/chains/:id", h.DeleteChain)

	req := httptest.NewRequest(http.MethodPost, "/admin/chains", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	createBody := `{"networkId":"8453","name":"fail-create","chainType":"EVM","rpcUrl":"https://rpc","symbol":"ETH"}`
	req = httptest.NewRequest(http.MethodPost, "/admin/chains", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	createBody = `{"networkId":"8453","name":"base","chainType":"EVM","rpcUrl":"https://rpc","symbol":"ETH"}`
	req = httptest.NewRequest(http.MethodPost, "/admin/chains", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/admin/chains/not-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/admin/chains/"+chainID.String(), strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	updateBody := `{"networkId":"8453","name":"missing","chainType":"EVM","rpcUrl":"https://rpc","symbol":"ETH","isActive":true}`
	req = httptest.NewRequest(http.MethodPut, "/admin/chains/"+chainID.String(), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	updateBody = `{"networkId":"8453","name":"fail-update","chainType":"EVM","rpcUrl":"https://rpc","symbol":"ETH","isActive":true}`
	req = httptest.NewRequest(http.MethodPut, "/admin/chains/"+chainID.String(), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	updateBody = `{"networkId":"8453","name":"ok-update","chainType":"EVM","rpcUrl":"https://rpc","symbol":"ETH","isActive":true}`
	req = httptest.NewRequest(http.MethodPut, "/admin/chains/"+chainID.String(), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/admin/chains/not-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	notFoundID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	req = httptest.NewRequest(http.MethodDelete, "/admin/chains/"+notFoundID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	errID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	req = httptest.NewRequest(http.MethodDelete, "/admin/chains/"+errID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/admin/chains/"+chainID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
