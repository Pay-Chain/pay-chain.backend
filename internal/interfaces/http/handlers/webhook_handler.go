package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/response"
)

type WebhookService interface {
	ProcessIndexerWebhook(ctx context.Context, eventType string, data json.RawMessage) error
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

	response.Success(c, http.StatusOK, gin.H{"received": true})
}
