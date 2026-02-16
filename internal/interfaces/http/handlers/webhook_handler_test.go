package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

type webhookServiceStub struct {
	processFn func(ctx context.Context, eventType string, data json.RawMessage) error
}

func (s webhookServiceStub) ProcessIndexerWebhook(ctx context.Context, eventType string, data json.RawMessage) error {
	return s.processFn(ctx, eventType, data)
}

func TestWebhookHandler_HandleIndexerWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("bad request", func(t *testing.T) {
		r := gin.New()
		h := NewWebhookHandler(webhookServiceStub{
			processFn: func(context.Context, string, json.RawMessage) error {
				t.Fatal("should not be called")
				return nil
			},
		})
		r.POST("/webhooks/indexer", h.HandleIndexerWebhook)

		req := httptest.NewRequest(http.MethodPost, "/webhooks/indexer", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("usecase error", func(t *testing.T) {
		r := gin.New()
		h := NewWebhookHandler(webhookServiceStub{
			processFn: func(context.Context, string, json.RawMessage) error {
				return domainerrors.InternalServerError("failed")
			},
		})
		r.POST("/webhooks/indexer", h.HandleIndexerWebhook)

		body := `{"eventType":"PAYMENT_CREATED","data":{"id":"1"},"timestamp":"2026-02-16T00:00:00Z"}`
		req := httptest.NewRequest(http.MethodPost, "/webhooks/indexer", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		r := gin.New()
		h := NewWebhookHandler(webhookServiceStub{
			processFn: func(_ context.Context, eventType string, data json.RawMessage) error {
				if eventType != "PAYMENT_CREATED" {
					t.Fatalf("unexpected event type: %s", eventType)
				}
				if len(data) == 0 {
					t.Fatal("expected data payload")
				}
				return nil
			},
		})
		r.POST("/webhooks/indexer", h.HandleIndexerWebhook)

		body := `{"eventType":"PAYMENT_CREATED","data":{"id":"1"},"timestamp":"2026-02-16T00:00:00Z"}`
		req := httptest.NewRequest(http.MethodPost, "/webhooks/indexer", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"received":true`)) {
			t.Fatalf("expected success payload, body=%s", w.Body.String())
		}
	})
}
