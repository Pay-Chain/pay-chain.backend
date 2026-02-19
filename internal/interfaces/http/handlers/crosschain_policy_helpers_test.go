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
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

type crosschainChainRepoStub struct {
	getByChainID func(ctx context.Context, chainID string) (*entities.Chain, error)
	getByCAIP2   func(ctx context.Context, caip2 string) (*entities.Chain, error)
}

func (s *crosschainChainRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Chain, error) {
	return nil, nil
}
func (s *crosschainChainRepoStub) GetByChainID(ctx context.Context, chainID string) (*entities.Chain, error) {
	return s.getByChainID(ctx, chainID)
}
func (s *crosschainChainRepoStub) GetByCAIP2(ctx context.Context, caip2 string) (*entities.Chain, error) {
	return s.getByCAIP2(ctx, caip2)
}
func (s *crosschainChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) { return nil, nil }
func (s *crosschainChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *crosschainChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *crosschainChainRepoStub) Create(context.Context, *entities.Chain) error       { return nil }
func (s *crosschainChainRepoStub) Update(context.Context, *entities.Chain) error       { return nil }
func (s *crosschainChainRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }
func (s *crosschainChainRepoStub) CreateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *crosschainChainRepoStub) UpdateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *crosschainChainRepoStub) DeleteRPC(context.Context, uuid.UUID) error          { return nil }
func (s *crosschainChainRepoStub) GetRPCByID(context.Context, uuid.UUID) (*entities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}

type routePolicyRepoNoop struct{}

func (routePolicyRepoNoop) GetByID(context.Context, uuid.UUID) (*entities.RoutePolicy, error) {
	return nil, nil
}
func (routePolicyRepoNoop) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
	return nil, nil
}
func (routePolicyRepoNoop) List(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.RoutePolicy, int64, error) {
	return nil, 0, nil
}
func (routePolicyRepoNoop) Create(context.Context, *entities.RoutePolicy) error { return nil }
func (routePolicyRepoNoop) Update(context.Context, *entities.RoutePolicy) error { return nil }
func (routePolicyRepoNoop) Delete(context.Context, uuid.UUID) error             { return nil }

type layerZeroRepoNoop struct{}

func (layerZeroRepoNoop) GetByID(context.Context, uuid.UUID) (*entities.LayerZeroConfig, error) {
	return nil, nil
}
func (layerZeroRepoNoop) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.LayerZeroConfig, error) {
	return nil, nil
}
func (layerZeroRepoNoop) List(context.Context, *uuid.UUID, *uuid.UUID, *bool, utils.PaginationParams) ([]*entities.LayerZeroConfig, int64, error) {
	return nil, 0, nil
}
func (layerZeroRepoNoop) Create(context.Context, *entities.LayerZeroConfig) error { return nil }
func (layerZeroRepoNoop) Update(context.Context, *entities.LayerZeroConfig) error { return nil }
func (layerZeroRepoNoop) Delete(context.Context, uuid.UUID) error                 { return nil }

func TestCrosschainPolicyHelpers(t *testing.T) {
	chainByID := uuid.New()
	chainByCAIP2 := uuid.New()
	h := NewCrosschainPolicyHandler(
		routePolicyRepoNoop{},
		layerZeroRepoNoop{},
		&crosschainChainRepoStub{
			getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
				if chainID == "8453" {
					return &entities.Chain{ID: chainByID}, nil
				}
				return nil, errors.New("not found")
			},
			getByCAIP2: func(_ context.Context, caip2 string) (*entities.Chain, error) {
				if caip2 == "eip155:8453" {
					return &entities.Chain{ID: chainByCAIP2}, nil
				}
				return nil, errors.New("not found")
			},
		},
	)

	_, err := h.parseChainID(context.Background(), "")
	require.Error(t, err)

	id, err := h.parseChainID(context.Background(), chainByID.String())
	require.NoError(t, err)
	require.Equal(t, chainByID, id)

	id, err = h.parseChainID(context.Background(), "eip155:8453")
	require.NoError(t, err)
	require.Equal(t, chainByCAIP2, id)

	id, err = h.parseChainID(context.Background(), "8453")
	require.NoError(t, err)
	require.Equal(t, chainByID, id)

	_, err = h.parseChainID(context.Background(), "unknown")
	require.Error(t, err)

	opt, err := h.parseChainQuery(context.Background(), "")
	require.NoError(t, err)
	require.Nil(t, opt)

	opt, err = h.parseChainQuery(context.Background(), "8453")
	require.NoError(t, err)
	require.NotNil(t, opt)

	require.True(t, isValidBridgeType(0))
	require.True(t, isValidBridgeType(1))
	require.True(t, isValidBridgeType(2))
	require.False(t, isValidBridgeType(3))

	require.Error(t, validateBridgeOrder(nil))
	require.Error(t, validateBridgeOrder([]uint8{9}))
	require.Error(t, validateBridgeOrder([]uint8{1, 1}))
	require.NoError(t, validateBridgeOrder([]uint8{0, 1, 2}))

	require.Equal(t, "0x", normalizeHex(""))
	require.Equal(t, "0xabc", normalizeHex("abc"))
	require.Equal(t, "0xabc", normalizeHex("0xabc"))
}

func TestCrosschainPolicyHandler_ListRoutePoliciesInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewCrosschainPolicyHandler(
		routePolicyRepoNoop{},
		layerZeroRepoNoop{},
		&crosschainChainRepoStub{
			getByChainID: func(context.Context, string) (*entities.Chain, error) { return nil, domainerrors.ErrNotFound },
			getByCAIP2:   func(context.Context, string) (*entities.Chain, error) { return nil, domainerrors.ErrNotFound },
		},
	)
	r := gin.New()
	r.GET("/policies", h.ListRoutePolicies)

	req := httptest.NewRequest(http.MethodGet, "/policies?sourceChainId=bad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
