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
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestCrosschainPolicyHandler_GapValidationBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	routeID := uuid.New()
	lzID := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()
	peerHex := "0x" + strings.Repeat("a", 64)

	chainRepo := &crosschainChainRepoStub{
		getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
			switch chainID {
			case "8453":
				return &entities.Chain{ID: sourceID}, nil
			case "42161":
				return &entities.Chain{ID: destID}, nil
			default:
				return nil, domainerrors.ErrNotFound
			}
		},
		getByCAIP2: func(_ context.Context, caip2 string) (*entities.Chain, error) {
			switch caip2 {
			case "eip155:8453":
				return &entities.Chain{ID: sourceID}, nil
			case "eip155:42161":
				return &entities.Chain{ID: destID}, nil
			default:
				return nil, domainerrors.ErrNotFound
			}
		},
	}
	routeRepo := &routePolicyRepoErrMatrixStub{
		item: &entities.RoutePolicy{ID: routeID},
	}
	lzRepo := &layerZeroRepoErrMatrixStub{
		item: &entities.LayerZeroConfig{ID: lzID},
	}

	h := NewCrosschainPolicyHandler(routeRepo, lzRepo, chainRepo)
	r := gin.New()
	r.GET("/route", h.ListRoutePolicies)
	r.PUT("/route/:id", h.UpdateRoutePolicy)
	r.GET("/lz", h.ListLayerZeroConfigs)
	r.POST("/lz", h.CreateLayerZeroConfig)
	r.PUT("/lz/:id", h.UpdateLayerZeroConfig)

	t.Run("list route invalid dest query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/route?sourceChainId=8453&destChainId=bad-dest", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update route invalid json bind", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/route/"+routeID.String(), strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update route invalid source chain", func(t *testing.T) {
		body := `{"sourceChainId":"bad-source","destChainId":"42161","defaultBridgeType":0}`
		req := httptest.NewRequest(http.MethodPut, "/route/"+routeID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update route invalid dest chain", func(t *testing.T) {
		body := `{"sourceChainId":"8453","destChainId":"bad-dest","defaultBridgeType":0}`
		req := httptest.NewRequest(http.MethodPut, "/route/"+routeID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("list layerzero invalid dest query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/lz?sourceChainId=8453&destChainId=bad-dest", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("create layerzero invalid source chain", func(t *testing.T) {
		body := `{"sourceChainId":"bad-source","destChainId":"42161","dstEid":1,"peerHex":"` + peerHex + `"}`
		req := httptest.NewRequest(http.MethodPost, "/lz", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("create layerzero invalid dest chain", func(t *testing.T) {
		body := `{"sourceChainId":"8453","destChainId":"bad-dest","dstEid":1,"peerHex":"` + peerHex + `"}`
		req := httptest.NewRequest(http.MethodPost, "/lz", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update layerzero invalid json bind", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/lz/"+lzID.String(), strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update layerzero source equals dest", func(t *testing.T) {
		body := `{"sourceChainId":"8453","destChainId":"8453","dstEid":1,"peerHex":"` + peerHex + `"}`
		req := httptest.NewRequest(http.MethodPut, "/lz/"+lzID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

