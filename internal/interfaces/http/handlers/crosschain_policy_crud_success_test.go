package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

type routePolicyRepoMemory struct {
	item *entities.RoutePolicy
}

func (m *routePolicyRepoMemory) GetByID(context.Context, uuid.UUID) (*entities.RoutePolicy, error) {
	if m.item == nil {
		m.item = &entities.RoutePolicy{ID: utils.GenerateUUIDv7()}
	}
	return m.item, nil
}
func (m *routePolicyRepoMemory) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
	return m.item, nil
}
func (m *routePolicyRepoMemory) List(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.RoutePolicy, int64, error) {
	if m.item == nil {
		return []*entities.RoutePolicy{}, 0, nil
	}
	return []*entities.RoutePolicy{m.item}, 1, nil
}
func (m *routePolicyRepoMemory) Create(_ context.Context, p *entities.RoutePolicy) error { m.item = p; return nil }
func (m *routePolicyRepoMemory) Update(_ context.Context, p *entities.RoutePolicy) error { m.item = p; return nil }
func (m *routePolicyRepoMemory) Delete(context.Context, uuid.UUID) error                 { return nil }

type layerZeroRepoMemory struct {
	item *entities.LayerZeroConfig
}

func (m *layerZeroRepoMemory) GetByID(context.Context, uuid.UUID) (*entities.LayerZeroConfig, error) {
	if m.item == nil {
		m.item = &entities.LayerZeroConfig{ID: utils.GenerateUUIDv7()}
	}
	return m.item, nil
}
func (m *layerZeroRepoMemory) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.LayerZeroConfig, error) {
	return m.item, nil
}
func (m *layerZeroRepoMemory) List(context.Context, *uuid.UUID, *uuid.UUID, *bool, utils.PaginationParams) ([]*entities.LayerZeroConfig, int64, error) {
	if m.item == nil {
		return []*entities.LayerZeroConfig{}, 0, nil
	}
	return []*entities.LayerZeroConfig{m.item}, 1, nil
}
func (m *layerZeroRepoMemory) Create(_ context.Context, c *entities.LayerZeroConfig) error { m.item = c; return nil }
func (m *layerZeroRepoMemory) Update(_ context.Context, c *entities.LayerZeroConfig) error { m.item = c; return nil }
func (m *layerZeroRepoMemory) Delete(context.Context, uuid.UUID) error                      { return nil }

func TestCrosschainPolicyHandler_CRUDSuccessPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routeRepo := &routePolicyRepoMemory{}
	lzRepo := &layerZeroRepoMemory{}
	h := NewCrosschainPolicyHandler(routeRepo, lzRepo, &crosschainChainRepoStub{})

	sourceID := utils.GenerateUUIDv7()
	destID := utils.GenerateUUIDv7()
	if sourceID == destID {
		destID = utils.GenerateUUIDv7()
	}

	r := gin.New()
	r.POST("/route", h.CreateRoutePolicy)
	r.PUT("/route/:id", h.UpdateRoutePolicy)
	r.POST("/lz", h.CreateLayerZeroConfig)
	r.PUT("/lz/:id", h.UpdateLayerZeroConfig)

	createRouteBody := `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","defaultBridgeType":1,"fallbackMode":"auto_fallback","fallbackOrder":[1,0]}`
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(createRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "\"policy\"")
	require.NotNil(t, routeRepo.item)

	policyID := routeRepo.item.ID
	updateRouteBody := `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","defaultBridgeType":2,"fallbackMode":"strict","fallbackOrder":[2]}`
	req = httptest.NewRequest(http.MethodPut, "/route/"+policyID.String(), strings.NewReader(updateRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"policy\"")
	require.Equal(t, uint8(2), routeRepo.item.DefaultBridgeType)

	peerHex := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	createLZBody := `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","dstEid":30110,"peerHex":"` + peerHex + `","optionsHex":"0x0102","isActive":true}`
	req = httptest.NewRequest(http.MethodPost, "/lz", strings.NewReader(createLZBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "\"config\"")
	require.NotNil(t, lzRepo.item)

	cfgID := lzRepo.item.ID
	updateLZBody := `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","dstEid":30111,"peerHex":"` + peerHex + `","optionsHex":"0x03","isActive":false}`
	req = httptest.NewRequest(http.MethodPut, "/lz/"+cfgID.String(), strings.NewReader(updateLZBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"config\"")
	require.Equal(t, uint32(30111), lzRepo.item.DstEID)
	require.False(t, lzRepo.item.IsActive)
}

func TestCrosschainPolicyHandler_CreateRoutePolicy_DefaultFallbackValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routeRepo := &routePolicyRepoMemory{}
	sourceID := utils.GenerateUUIDv7()
	destID := utils.GenerateUUIDv7()

	chainRepo := &crosschainChainRepoStub{
		getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
			switch chainID {
			case "8453":
				return &entities.Chain{ID: sourceID}, nil
			case "42161":
				return &entities.Chain{ID: destID}, nil
			default:
				return nil, nil
			}
		},
	}

	h := NewCrosschainPolicyHandler(routeRepo, &layerZeroRepoMemory{}, chainRepo)
	r := gin.New()
	r.POST("/route", h.CreateRoutePolicy)

	body := `{"sourceChainId":"8453","destChainId":"42161","defaultBridgeType":2}`
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, routeRepo.item)
	require.Equal(t, entities.BridgeFallbackModeStrict, routeRepo.item.FallbackMode)
	require.Equal(t, []uint8{2}, routeRepo.item.FallbackOrder)
}
