package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPaymentHandler_ValidationAndAuthBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPaymentHandler(nil)

	r := gin.New()
	r.POST("/payments", h.CreatePayment)
	r.GET("/payments/:id", h.GetPayment)
	r.GET("/payments", h.ListPayments)
	r.GET("/payments/:id/events", h.GetPaymentEvents)

	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid create payload, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161","sourceTokenAddress":"native","destTokenAddress":"0x1","amount":"1","decimals":6,"receiverAddress":"0x2"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated create, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/payments/not-a-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payment id, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/payments", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated list, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/payments/not-a-uuid/events", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payment id events, got %d", w.Code)
	}
}
