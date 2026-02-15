package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOnchainAdapterHandler_ValidationBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOnchainAdapterHandler(nil)
	r := gin.New()
	r.GET("/status", h.GetStatus)
	r.POST("/register", h.RegisterAdapter)
	r.POST("/default-bridge", h.SetDefaultBridgeType)
	r.POST("/hyperbridge-config", h.SetHyperbridgeConfig)
	r.POST("/ccip-config", h.SetCCIPConfig)
	r.POST("/layerzero-config", h.SetLayerZeroConfig)

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/status", ""},
		{http.MethodPost, "/register", `{}`},
		{http.MethodPost, "/default-bridge", `{}`},
		{http.MethodPost, "/hyperbridge-config", `{}`},
		{http.MethodPost, "/ccip-config", `{}`},
		{http.MethodPost, "/layerzero-config", `{}`},
	}

	for _, tc := range tests {
		var req *http.Request
		if tc.method == http.MethodGet {
			req = httptest.NewRequest(tc.method, tc.path, nil)
		} else {
			req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	}
}

func TestPaymentAppAndWebhookHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/payment-app", NewPaymentAppHandler(nil).CreatePaymentApp)
	r.POST("/webhook", NewWebhookHandler(nil).HandleIndexerWebhook)

	req1 := httptest.NewRequest(http.MethodPost, "/payment-app", strings.NewReader("{"))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusBadRequest, w1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{"))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusBadRequest, w2.Code)
}
