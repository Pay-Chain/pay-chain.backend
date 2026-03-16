package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/infrastructure/metrics"
	"payment-kita.backend/internal/interfaces/http/response"
)

type WebhookService interface {
	ProcessIndexerWebhook(ctx context.Context, eventType string, data json.RawMessage) error
	ManualRetry(ctx context.Context, deliveryID uuid.UUID) error
}

// WebhookHandler handles webhook endpoints
type WebhookHandler struct {
	webhookUsecase WebhookService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(webhookUsecase WebhookService) *WebhookHandler {
	return &WebhookHandler{webhookUsecase: webhookUsecase}
}

// HandleIndexerWebhook handles incoming webhooks from the Ponder indexer
// POST /api/v1/webhooks/indexer
func (h *WebhookHandler) HandleIndexerWebhook(c *gin.Context) {
	var input struct {
		EventType string          `json:"eventType"`
		Data      json.RawMessage `json:"data"`
		Timestamp string          `json:"timestamp"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	err := h.webhookUsecase.ProcessIndexerWebhook(c.Request.Context(), input.EventType, input.Data)
	if err != nil {
		response.Error(c, err)
		return
	}

	// Record Indexer Lag if timestamp is present
	if input.Timestamp != "" {
		if ts, err := strconv.ParseInt(input.Timestamp, 10, 64); err == nil {
			lag := float64(time.Now().Unix() - ts)
			// For now using "unknown" chain_id if not in Data, or extract from Data later
			metrics.RecordIndexerLag("unknown", lag)
		}
	}

	response.Success(c, http.StatusOK, gin.H{"received": true})
}

// RetryWebhook manually triggers a webhook delivery attempt
// POST /api/v1/admin/webhooks/:id/retry
func (h *WebhookHandler) RetryWebhook(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid webhook delivery id"))
		return
	}

	err = h.webhookUsecase.ManualRetry(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"sent": true})
}
