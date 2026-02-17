package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/utils"
)

type crosschainConfigServiceStub struct {
	overviewFn func(context.Context, string, string, utils.PaginationParams) (*usecases.CrosschainOverview, error)
	recheckFn  func(context.Context, string, string) (*usecases.CrosschainRouteStatus, error)
	preflightFn func(context.Context, string, string) (*usecases.CrosschainPreflightResult, error)
	autofixFn  func(context.Context, *usecases.AutoFixRequest) (*usecases.AutoFixResult, error)
}

func (s crosschainConfigServiceStub) Overview(ctx context.Context, src, dst string, p utils.PaginationParams) (*usecases.CrosschainOverview, error) {
	if s.overviewFn != nil {
		return s.overviewFn(ctx, src, dst, p)
	}
	return &usecases.CrosschainOverview{}, nil
}
func (s crosschainConfigServiceStub) RecheckRoute(ctx context.Context, src, dst string) (*usecases.CrosschainRouteStatus, error) {
	if s.recheckFn != nil {
		return s.recheckFn(ctx, src, dst)
	}
	return &usecases.CrosschainRouteStatus{}, nil
}
func (s crosschainConfigServiceStub) Preflight(ctx context.Context, src, dst string) (*usecases.CrosschainPreflightResult, error) {
	if s.preflightFn != nil {
		return s.preflightFn(ctx, src, dst)
	}
	return &usecases.CrosschainPreflightResult{}, nil
}
func (s crosschainConfigServiceStub) AutoFix(ctx context.Context, req *usecases.AutoFixRequest) (*usecases.AutoFixResult, error) {
	if s.autofixFn != nil {
		return s.autofixFn(ctx, req)
	}
	return &usecases.AutoFixResult{}, nil
}

func TestCrosschainConfigHandler_SuccessFlows_Gap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &CrosschainConfigHandler{usecase: crosschainConfigServiceStub{
		recheckFn: func(_ context.Context, src, dst string) (*usecases.CrosschainRouteStatus, error) {
			return &usecases.CrosschainRouteStatus{RouteKey: src + "->" + dst, OverallStatus: "READY"}, nil
		},
		preflightFn: func(_ context.Context, src, dst string) (*usecases.CrosschainPreflightResult, error) {
			return &usecases.CrosschainPreflightResult{SourceChainID: src, DestChainID: dst, PolicyExecutable: true}, nil
		},
		autofixFn: func(_ context.Context, req *usecases.AutoFixRequest) (*usecases.AutoFixResult, error) {
			return &usecases.AutoFixResult{SourceChainID: req.SourceChainID, DestChainID: req.DestChainID, Steps: []usecases.AutoFixStep{{Step: "setDefaultBridge", Status: "SUCCESS"}}}, nil
		},
	}}

	r := gin.New()
	r.POST("/recheck", h.Recheck)
	r.GET("/preflight", h.Preflight)
	r.POST("/autofix", h.AutoFix)

	req := httptest.NewRequest(http.MethodPost, "/recheck", strings.NewReader(`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "READY")

	req = httptest.NewRequest(http.MethodGet, "/preflight?sourceChainId=eip155:8453&destChainId=eip155:42161", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "policyExecutable")

	req = httptest.NewRequest(http.MethodPost, "/autofix", strings.NewReader(`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "setDefaultBridge")
}
