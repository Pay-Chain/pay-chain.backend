package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthHandler_ValidationAndUnauthorizedPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(nil, nil)

	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/verify-email", h.VerifyEmail)
	r.POST("/refresh", h.RefreshToken)
	r.GET("/me", h.GetMe)
	r.POST("/change-password", h.ChangePassword)
	r.POST("/logout", h.Logout)

	for _, p := range []string{"/register", "/login", "/verify-email"} {
		req := httptest.NewRequest(http.MethodPost, p, bytes.NewReader([]byte("{")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path %s expected 400, got %d body=%s", p, rec.Code, rec.Body.String())
		}
	}

	// refresh with no token/session
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("refresh expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// unauthorized access (missing user context)
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/me"},
		{method: http.MethodPost, path: "/change-password", body: `{"oldPassword":"a","newPassword":"b"}`},
	} {
		req = httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(tc.body)))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("path %s expected 401, got %d body=%s", tc.path, rec.Code, rec.Body.String())
		}
	}

	// logout should be safe without cookie/session
	req = httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPaymentAndWalletHandlers_ValidationPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payment := NewPaymentHandler(nil)
	wallet := NewWalletHandler(nil)

	r := gin.New()
	r.POST("/payments", payment.CreatePayment)
	r.GET("/payments/:id", payment.GetPayment)
	r.GET("/payments", payment.ListPayments)
	r.GET("/payments/:id/events", payment.GetPaymentEvents)
	r.POST("/wallets/connect", wallet.ConnectWallet)
	r.GET("/wallets", wallet.ListWallets)
	r.PUT("/wallets/:id/primary", wallet.SetPrimaryWallet)
	r.DELETE("/wallets/:id", wallet.DisconnectWallet)

	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create payment expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	for _, p := range []string{"/payments/not-uuid", "/payments/not-uuid/events", "/wallets/not-uuid/primary", "/wallets/not-uuid"} {
		req = httptest.NewRequest(http.MethodGet, p, nil)
		if p == "/wallets/not-uuid" {
			req = httptest.NewRequest(http.MethodDelete, p, nil)
		}
		if p == "/wallets/not-uuid/primary" {
			req = httptest.NewRequest(http.MethodPut, p, nil)
		}
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path %s expected 400, got %d body=%s", p, rec.Code, rec.Body.String())
		}
	}

	for _, p := range []string{"/payments", "/wallets"} {
		req = httptest.NewRequest(http.MethodGet, p, nil)
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("path %s expected 401, got %d body=%s", p, rec.Code, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/wallets/connect", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("connect wallet expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMerchantPaymentRequestApiKeyHandlers_ValidationPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	merchant := NewMerchantHandler(nil)
	paymentRequest := NewPaymentRequestHandler(nil)
	apiKey := NewApiKeyHandler(nil)

	r := gin.New()
	r.POST("/merchants/apply", merchant.ApplyMerchant)
	r.GET("/merchants/status", merchant.GetMerchantStatus)
	r.POST("/payment-requests", paymentRequest.CreatePaymentRequest)
	r.GET("/payment-requests/:id", paymentRequest.GetPaymentRequest)
	r.GET("/pay/:id", paymentRequest.GetPublicPaymentRequest)
	r.GET("/payment-requests", paymentRequest.ListPaymentRequests)
	r.POST("/api-keys", apiKey.CreateApiKey)
	r.GET("/api-keys", apiKey.ListApiKeys)
	r.DELETE("/api-keys/:id", apiKey.RevokeApiKey)

	// merchant invalid json
	req := httptest.NewRequest(http.MethodPost, "/merchants/apply", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("merchant apply expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// merchant unauthorized status
	req = httptest.NewRequest(http.MethodGet, "/merchants/status", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("merchant status expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}

	// payment request unauthorized + invalid id
	req = httptest.NewRequest(http.MethodPost, "/payment-requests", bytes.NewReader([]byte(`{"chainId":"1"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("create payment request expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}

	for _, p := range []string{"/payment-requests/not-uuid", "/pay/not-uuid"} {
		req = httptest.NewRequest(http.MethodGet, p, nil)
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path %s expected 400, got %d body=%s", p, rec.Code, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/payment-requests", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("list payment requests expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}

	// api key validations
	req = httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create api key expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api-keys", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("list api key expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api-keys/not-uuid", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("revoke api key expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
