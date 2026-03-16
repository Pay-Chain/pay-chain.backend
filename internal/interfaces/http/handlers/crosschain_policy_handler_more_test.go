package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/pkg/utils"
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

type stargateRepoListDeleteStub struct {
	item *entities.StargateConfig
}

func (s *stargateRepoListDeleteStub) GetByID(context.Context, uuid.UUID) (*entities.StargateConfig, error) {
	if s.item == nil {
		s.item = &entities.StargateConfig{ID: utils.GenerateUUIDv7()}
	}
	return s.item, nil
}
func (s *stargateRepoListDeleteStub) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.StargateConfig, error) {
	return s.item, nil
}
func (s *stargateRepoListDeleteStub) List(context.Context, *uuid.UUID, *uuid.UUID, *bool, utils.PaginationParams) ([]*entities.StargateConfig, int64, error) {
	if s.item == nil {
		s.item = &entities.StargateConfig{ID: utils.GenerateUUIDv7()}
	}
	return []*entities.StargateConfig{s.item}, 1, nil
}
func (s *stargateRepoListDeleteStub) Create(context.Context, *entities.StargateConfig) error { return nil }
func (s *stargateRepoListDeleteStub) Update(context.Context, *entities.StargateConfig) error { return nil }
func (s *stargateRepoListDeleteStub) Delete(context.Context, uuid.UUID) error                 { return nil }

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
	lzRepo := &stargateRepoListDeleteStub{}
	h := NewCrosschainPolicyHandler(routeRepo, lzRepo, chainRepo)

	r := gin.New()
	r.GET("/route-policies", h.ListRoutePolicies)
	r.DELETE("/route-policies/:id", h.DeleteRoutePolicy)
	r.GET("/lz-configs", h.ListStargateConfigs)
	r.DELETE("/lz-configs/:id", h.DeleteStargateConfig)

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
	require.Contains(t, w.Body.String(), "Stargate config deleted")
}

