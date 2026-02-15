package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

type rpcChainRepoStub struct {
	getAllRPCsFn func(ctx context.Context, chainID *uuid.UUID, isActive *bool, search *string, pagination utils.PaginationParams) ([]*entities.ChainRPC, int64, error)
}

func (s *rpcChainRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Chain, error)        { return nil, nil }
func (s *rpcChainRepoStub) GetByChainID(context.Context, string) (*entities.Chain, error)      { return nil, nil }
func (s *rpcChainRepoStub) GetByCAIP2(context.Context, string) (*entities.Chain, error)        { return nil, nil }
func (s *rpcChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error)                  { return nil, nil }
func (s *rpcChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *rpcChainRepoStub) Create(context.Context, *entities.Chain) error { return nil }
func (s *rpcChainRepoStub) Update(context.Context, *entities.Chain) error { return nil }
func (s *rpcChainRepoStub) Delete(context.Context, uuid.UUID) error       { return nil }
func (s *rpcChainRepoStub) GetAllRPCs(ctx context.Context, chainID *uuid.UUID, isActive *bool, search *string, pagination utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return s.getAllRPCsFn(ctx, chainID, isActive, search, pagination)
}

func TestRpcHandler_ListRPCs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("invalid chainId", func(t *testing.T) {
		r := gin.New()
		h := NewRpcHandler(&rpcChainRepoStub{
			getAllRPCsFn: func(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
				return nil, 0, nil
			},
		})
		r.GET("/rpcs", h.ListRPCs)

		req := httptest.NewRequest(http.MethodGet, "/rpcs?chainId=bad", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid isActive", func(t *testing.T) {
		r := gin.New()
		h := NewRpcHandler(&rpcChainRepoStub{
			getAllRPCsFn: func(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
				return nil, 0, nil
			},
		})
		r.GET("/rpcs", h.ListRPCs)

		req := httptest.NewRequest(http.MethodGet, "/rpcs?isActive=bad", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("repo error", func(t *testing.T) {
		r := gin.New()
		h := NewRpcHandler(&rpcChainRepoStub{
			getAllRPCsFn: func(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
				return nil, 0, errors.New("boom")
			},
		})
		r.GET("/rpcs", h.ListRPCs)

		req := httptest.NewRequest(http.MethodGet, "/rpcs?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		chainID := uuid.New()
		rpcID := uuid.New()
		r := gin.New()
		h := NewRpcHandler(&rpcChainRepoStub{
			getAllRPCsFn: func(_ context.Context, gotChainID *uuid.UUID, gotActive *bool, gotSearch *string, gotPagination utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
				require.NotNil(t, gotChainID)
				require.Equal(t, chainID, *gotChainID)
				require.NotNil(t, gotActive)
				require.True(t, *gotActive)
				require.NotNil(t, gotSearch)
				require.Equal(t, "rpc", *gotSearch)
				require.Equal(t, 2, gotPagination.Page)
				require.Equal(t, 5, gotPagination.Limit)
				return []*entities.ChainRPC{{ID: rpcID, URL: "https://rpc"}}, 1, nil
			},
		})
		r.GET("/rpcs", h.ListRPCs)

		req := httptest.NewRequest(http.MethodGet, "/rpcs?chainId="+chainID.String()+"&isActive=true&search=rpc&page=2&limit=5", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), rpcID.String())
	})
}
