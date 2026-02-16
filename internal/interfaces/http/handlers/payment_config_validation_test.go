package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestPaymentConfigHandler_InvalidBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewPaymentConfigHandler(
		nil,
		nil,
		nil,
		&crosschainChainRepoStub{
			getByChainID: func(_ context.Context, _ string) (*entities.Chain, error) {
				return nil, domainerrors.ErrNotFound
			},
			getByCAIP2: func(_ context.Context, _ string) (*entities.Chain, error) {
				return nil, domainerrors.ErrNotFound
			},
		},
		nil,
	)

	r := gin.New()
	r.POST("/payment-bridges", h.CreatePaymentBridge)
	r.PUT("/payment-bridges/:id", h.UpdatePaymentBridge)
	r.DELETE("/payment-bridges/:id", h.DeletePaymentBridge)

	r.GET("/bridge-configs", h.ListBridgeConfigs)
	r.POST("/bridge-configs", h.CreateBridgeConfig)
	r.PUT("/bridge-configs/:id", h.UpdateBridgeConfig)
	r.DELETE("/bridge-configs/:id", h.DeleteBridgeConfig)

	r.GET("/fee-configs", h.ListFeeConfigs)
	r.POST("/fee-configs", h.CreateFeeConfig)
	r.PUT("/fee-configs/:id", h.UpdateFeeConfig)
	r.DELETE("/fee-configs/:id", h.DeleteFeeConfig)

	tests := []struct {
		method string
		path   string
		body   string
		code   int
	}{
		{http.MethodPost, "/payment-bridges", `{}`, http.StatusBadRequest},
		{http.MethodPut, "/payment-bridges/bad", `{}`, http.StatusBadRequest},
		{http.MethodDelete, "/payment-bridges/bad", ``, http.StatusBadRequest},
		{http.MethodGet, "/bridge-configs?sourceChainId=bad", ``, http.StatusBadRequest},
		{http.MethodGet, "/bridge-configs?destChainId=bad", ``, http.StatusBadRequest},
		{http.MethodGet, "/bridge-configs?bridgeId=bad", ``, http.StatusBadRequest},
		{http.MethodPost, "/bridge-configs", `{}`, http.StatusBadRequest},
		{http.MethodPut, "/bridge-configs/bad", `{}`, http.StatusBadRequest},
		{http.MethodDelete, "/bridge-configs/bad", ``, http.StatusBadRequest},
		{http.MethodGet, "/fee-configs?chainId=bad", ``, http.StatusBadRequest},
		{http.MethodPost, "/fee-configs", `{}`, http.StatusBadRequest},
		{http.MethodPost, "/fee-configs", `{"chainId":"eip155:8453","tokenId":"bad"}`, http.StatusBadRequest},
		{http.MethodPut, "/fee-configs/bad", `{}`, http.StatusBadRequest},
		{http.MethodDelete, "/fee-configs/bad", ``, http.StatusBadRequest},
	}

	for _, tc := range tests {
		var req *http.Request
		if tc.method == http.MethodGet || tc.method == http.MethodDelete {
			req = httptest.NewRequest(tc.method, tc.path, nil)
		} else {
			req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, tc.code, w.Code, tc.method+" "+tc.path)
	}
}
