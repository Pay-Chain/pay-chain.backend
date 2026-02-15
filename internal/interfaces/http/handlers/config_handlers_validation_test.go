package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCrosschainConfigHandler_ValidationPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewCrosschainConfigHandler(nil)

	r := gin.New()
	r.GET("/overview", h.Overview)
	r.POST("/recheck", h.Recheck)
	r.GET("/preflight", h.Preflight)
	r.POST("/autofix", h.AutoFix)
	r.POST("/recheck-bulk", h.RecheckBulk)
	r.POST("/autofix-bulk", h.AutoFixBulk)

	// Preflight: required query missing
	req := httptest.NewRequest(http.MethodGet, "/preflight", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Recheck: invalid JSON body
	req = httptest.NewRequest(http.MethodPost, "/recheck", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// AutoFix: missing required fields
	req = httptest.NewRequest(http.MethodPost, "/autofix", bytes.NewReader([]byte(`{"sourceChainId":""}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// RecheckBulk: invalid payload shape
	req = httptest.NewRequest(http.MethodPost, "/recheck-bulk", bytes.NewReader([]byte(`{"routes":{}}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// AutoFixBulk: invalid payload shape
	req = httptest.NewRequest(http.MethodPost, "/autofix-bulk", bytes.NewReader([]byte(`{"routes":{}}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOnchainAdapterHandler_ValidationPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOnchainAdapterHandler(nil)

	r := gin.New()
	r.GET("/status", h.GetStatus)
	r.POST("/register", h.RegisterAdapter)
	r.POST("/default-bridge", h.SetDefaultBridgeType)
	r.POST("/hyperbridge", h.SetHyperbridgeConfig)
	r.POST("/ccip", h.SetCCIPConfig)
	r.POST("/layerzero", h.SetLayerZeroConfig)

	// GetStatus requires sourceChainId + destChainId
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	for _, tc := range []struct {
		path string
		body string
	}{
		{path: "/register", body: `{"sourceChainId":"eip155:8453"}`},
		{path: "/default-bridge", body: `{"sourceChainId":"eip155:8453"}`},
		{path: "/hyperbridge", body: `{"sourceChainId":""}`},
		{path: "/ccip", body: `{"sourceChainId":""}`},
		{path: "/layerzero", body: `{"sourceChainId":""}`},
	} {
		req = httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader([]byte(tc.body)))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path %s expected 400, got %d body=%s", tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestContractConfigAuditHandler_ValidationPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewContractConfigAuditHandler(nil)

	r := gin.New()
	r.GET("/check", h.Check)
	r.GET("/contracts/:id/check", h.CheckByContract)

	// sourceChainId query required
	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// invalid UUID path param
	req = httptest.NewRequest(http.MethodGet, "/contracts/not-a-uuid/check", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
