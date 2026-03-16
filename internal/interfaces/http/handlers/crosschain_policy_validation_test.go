package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
)

func TestCrosschainPolicyHandler_InvalidParamAndBodyBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewCrosschainPolicyHandler(
		routePolicyRepoNoop{},
		stargateRepoNoop{},
		&crosschainChainRepoStub{
			getByChainID: func(_ context.Context, _ string) (*entities.Chain, error) {
				return nil, domainerrors.ErrNotFound
			},
			getByCAIP2: func(_ context.Context, _ string) (*entities.Chain, error) {
				return nil, domainerrors.ErrNotFound
			},
		},
	)

	r := gin.New()
	r.DELETE("/route/:id", h.DeleteRoutePolicy)
	r.PUT("/route/:id", h.UpdateRoutePolicy)
	r.GET("/lz", h.ListStargateConfigs)
	r.POST("/lz", h.CreateStargateConfig)
	r.PUT("/lz/:id", h.UpdateStargateConfig)
	r.DELETE("/lz/:id", h.DeleteStargateConfig)

	req := httptest.NewRequest(http.MethodDelete, "/route/bad-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/route/bad-id", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/lz?sourceChainId=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/lz", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/lz/bad-id", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/lz/bad-id", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
