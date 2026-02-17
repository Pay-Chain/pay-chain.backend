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

func TestCrosschainConfigHandler_BulkSuccessFlows(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &CrosschainConfigHandler{usecase: crosschainConfigServiceStub{
		overviewFn: func(context.Context, string, string, utils.PaginationParams) (*usecases.CrosschainOverview, error) {
			return &usecases.CrosschainOverview{}, nil
		},
		recheckFn: func(_ context.Context, src, dst string) (*usecases.CrosschainRouteStatus, error) {
			return &usecases.CrosschainRouteStatus{RouteKey: src + "->" + dst, OverallStatus: "READY", SourceChainID: src, DestChainID: dst}, nil
		},
		autofixFn: func(_ context.Context, req *usecases.AutoFixRequest) (*usecases.AutoFixResult, error) {
			return &usecases.AutoFixResult{SourceChainID: req.SourceChainID, DestChainID: req.DestChainID, Steps: []usecases.AutoFixStep{{Step: "autoFix", Status: "SUCCESS"}}}, nil
		},
	}}

	r := gin.New()
	r.POST("/recheck-bulk", h.RecheckBulk)
	r.POST("/autofix-bulk", h.AutoFixBulk)

	recheckBody := `{"routes":[{"sourceChainId":"eip155:8453","destChainId":"eip155:42161"},{"sourceChainId":"eip155:8453","destChainId":"eip155:10"}]}`
	req := httptest.NewRequest(http.MethodPost, "/recheck-bulk", strings.NewReader(recheckBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "READY")

	autoBody := `{"routes":[{"sourceChainId":"eip155:8453","destChainId":"eip155:42161"}]}`
	req = httptest.NewRequest(http.MethodPost, "/autofix-bulk", strings.NewReader(autoBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "SUCCESS")
}
