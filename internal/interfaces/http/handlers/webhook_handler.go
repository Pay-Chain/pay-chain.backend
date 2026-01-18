package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/usecases"
)

// WebhookHandler handles webhook endpoints
type WebhookHandler struct {
	webhookUsecase *usecases.WebhookUsecase
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(webhookUsecase *usecases.WebhookUsecase) *WebhookHandler {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.webhookUsecase.ProcessIndexerWebhook(c.Request.Context(), input.EventType, input.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process webhook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}
