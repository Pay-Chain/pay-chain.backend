package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

type routePolicyRepoListDeleteStub struct {
	item *entities.RoutePolicy
}

func (s *routePolicyRepoListDeleteStub) GetByID(context.Context, uuid.UUID) (*entities.RoutePolicy, error) {
	if s.item == nil {
		s.item = &entities.RoutePolicy{ID: utils.GenerateUUIDv7()}
	}
	return s.item, nil
}
func (s *routePolicyRepoListDeleteStub) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
	return s.item, nil
}
func (s *routePolicyRepoListDeleteStub) List(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.RoutePolicy, int64, error) {
	if s.item == nil {
		s.item = &entities.RoutePolicy{ID: utils.GenerateUUIDv7()}
	}
	return []*entities.RoutePolicy{s.item}, 1, nil
}
func (s *routePolicyRepoListDeleteStub) Create(context.Context, *entities.RoutePolicy) error { return nil }
func (s *routePolicyRepoListDeleteStub) Update(context.Context, *entities.RoutePolicy) error { return nil }
func (s *routePolicyRepoListDeleteStub) Delete(context.Context, uuid.UUID) error             { return nil }

type layerZeroRepoListDeleteStub struct {
	item *entities.LayerZeroConfig
}

func (s *layerZeroRepoListDeleteStub) GetByID(context.Context, uuid.UUID) (*entities.LayerZeroConfig, error) {
	if s.item == nil {
		s.item = &entities.LayerZeroConfig{ID: utils.GenerateUUIDv7()}
	}
	return s.item, nil
}
func (s *layerZeroRepoListDeleteStub) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.LayerZeroConfig, error) {
	return s.item, nil
}
func (s *layerZeroRepoListDeleteStub) List(context.Context, *uuid.UUID, *uuid.UUID, *bool, utils.PaginationParams) ([]*entities.LayerZeroConfig, int64, error) {
	if s.item == nil {
		s.item = &entities.LayerZeroConfig{ID: utils.GenerateUUIDv7()}
	}
	return []*entities.LayerZeroConfig{s.item}, 1, nil
}
func (s *layerZeroRepoListDeleteStub) Create(context.Context, *entities.LayerZeroConfig) error { return nil }
func (s *layerZeroRepoListDeleteStub) Update(context.Context, *entities.LayerZeroConfig) error { return nil }
func (s *layerZeroRepoListDeleteStub) Delete(context.Context, uuid.UUID) error                 { return nil }

func TestCrosschainPolicyHandler_ListAndDeleteSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sourceChainID := utils.GenerateUUIDv7()
	destChainID := utils.GenerateUUIDv7()
	chainRepo := &crosschainChainRepoStub{
		getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
			switch chainID {
			case "8453":
				return &entities.Chain{ID: sourceChainID}, nil
			case "42161":
				return &entities.Chain{ID: destChainID}, nil
			default:
				return nil, nil
			}
		},
		getByCAIP2: func(_ context.Context, caip2 string) (*entities.Chain, error) {
			switch caip2 {
			case "eip155:8453":
				return &entities.Chain{ID: sourceChainID}, nil
			case "eip155:42161":
				return &entities.Chain{ID: destChainID}, nil
			default:
				return nil, nil
			}
		},
	}

	routeRepo := &routePolicyRepoListDeleteStub{}
	lzRepo := &layerZeroRepoListDeleteStub{}
	h := NewCrosschainPolicyHandler(routeRepo, lzRepo, chainRepo)

	r := gin.New()
	r.GET("/route-policies", h.ListRoutePolicies)
	r.DELETE("/route-policies/:id", h.DeleteRoutePolicy)
	r.GET("/lz-configs", h.ListLayerZeroConfigs)
	r.DELETE("/lz-configs/:id", h.DeleteLayerZeroConfig)

	req := httptest.NewRequest(http.MethodGet, "/route-policies?sourceChainId=8453&destChainId=eip155:42161&page=1&limit=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"items\"")
	require.Contains(t, w.Body.String(), "\"meta\"")

	req = httptest.NewRequest(http.MethodGet, "/lz-configs?sourceChainId=eip155:8453&destChainId=42161&activeOnly=true", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"items\"")

	routeID := routeRepo.item.ID
	req = httptest.NewRequest(http.MethodDelete, "/route-policies/"+routeID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Route policy deleted")

	lzID := lzRepo.item.ID
	req = httptest.NewRequest(http.MethodDelete, "/lz-configs/"+lzID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "LayerZero config deleted")
}

