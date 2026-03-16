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
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/pkg/utils"
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
func (m *routePolicyRepoMemory) Create(_ context.Context, p *entities.RoutePolicy) error {
	m.item = p
	return nil
}
func (m *routePolicyRepoMemory) Update(_ context.Context, p *entities.RoutePolicy) error {
	m.item = p
	return nil
}
func (m *routePolicyRepoMemory) Delete(context.Context, uuid.UUID) error { return nil }

type stargateRepoMemory struct {
	item *entities.StargateConfig
}

func (m *stargateRepoMemory) GetByID(context.Context, uuid.UUID) (*entities.StargateConfig, error) {
	if m.item == nil {
		m.item = &entities.StargateConfig{ID: utils.GenerateUUIDv7()}
	}
	return m.item, nil
}
func (m *stargateRepoMemory) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.StargateConfig, error) {
	return m.item, nil
}
func (m *stargateRepoMemory) List(context.Context, *uuid.UUID, *uuid.UUID, *bool, utils.PaginationParams) ([]*entities.StargateConfig, int64, error) {
	if m.item == nil {
		return []*entities.StargateConfig{}, 0, nil
	}
	return []*entities.StargateConfig{m.item}, 1, nil
}
func (m *stargateRepoMemory) Create(_ context.Context, c *entities.StargateConfig) error {
	m.item = c
	return nil
}
func (m *stargateRepoMemory) Update(_ context.Context, c *entities.StargateConfig) error {
	m.item = c
	return nil
}
func (m *stargateRepoMemory) Delete(context.Context, uuid.UUID) error { return nil }

func TestCrosschainPolicyHandler_CRUDSuccessPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routeRepo := &routePolicyRepoMemory{}
	lzRepo := &stargateRepoMemory{}
	h := NewCrosschainPolicyHandler(routeRepo, lzRepo, &crosschainChainRepoStub{})

	sourceID := utils.GenerateUUIDv7()
	destID := utils.GenerateUUIDv7()
	if sourceID == destID {
		destID = utils.GenerateUUIDv7()
	}

	r := gin.New()
	r.POST("/route", h.CreateRoutePolicy)
	r.PUT("/route/:id", h.UpdateRoutePolicy)
	r.POST("/lz", h.CreateStargateConfig)
	r.PUT("/lz/:id", h.UpdateStargateConfig)

	createRouteBody := `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","defaultBridgeType":1,"fallbackMode":"auto_fallback","fallbackOrder":[1,0],"supportsTokenBridge":true,"supportsDestSwap":true,"supportsPrivacyForward":false,"bridgeToken":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","status":"active","perByteRate":"300","overheadBytes":"256","minFee":"1000","maxFee":"999999"}`
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(createRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "\"policy\"")
	require.NotNil(t, routeRepo.item)
	require.Equal(t, "300", routeRepo.item.PerByteRate)
	require.Equal(t, "256", routeRepo.item.OverheadBytes)
	require.Equal(t, "1000", routeRepo.item.MinFee)
	require.Equal(t, "999999", routeRepo.item.MaxFee)
	require.True(t, routeRepo.item.SupportsTokenBridge)
	require.True(t, routeRepo.item.SupportsDestSwap)
	require.False(t, routeRepo.item.SupportsPrivacyForward)
	require.Equal(t, "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", routeRepo.item.BridgeToken)
	require.Equal(t, "active", routeRepo.item.Status)

	policyID := routeRepo.item.ID
	updateRouteBody := `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","defaultBridgeType":2,"fallbackMode":"strict","fallbackOrder":[2],"supportsPrivacyForward":true,"status":"paused","perByteRate":"400","overheadBytes":"300","minFee":"500","maxFee":"1000"}`
	req = httptest.NewRequest(http.MethodPut, "/route/"+policyID.String(), strings.NewReader(updateRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"policy\"")
	require.Equal(t, uint8(2), routeRepo.item.DefaultBridgeType)
	require.Equal(t, "400", routeRepo.item.PerByteRate)
	require.Equal(t, "300", routeRepo.item.OverheadBytes)
	require.Equal(t, "500", routeRepo.item.MinFee)
	require.Equal(t, "1000", routeRepo.item.MaxFee)
	require.True(t, routeRepo.item.SupportsTokenBridge)
	require.True(t, routeRepo.item.SupportsDestSwap)
	require.True(t, routeRepo.item.SupportsPrivacyForward)
	require.Equal(t, "paused", routeRepo.item.Status)

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

	h := NewCrosschainPolicyHandler(routeRepo, &stargateRepoMemory{}, chainRepo)
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
