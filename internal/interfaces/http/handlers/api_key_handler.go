package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
)

type ApiKeyHandler struct {
	apiKeyUsecase *usecases.ApiKeyUsecase
}

func NewApiKeyHandler(apiKeyUsecase *usecases.ApiKeyUsecase) *ApiKeyHandler {
	return &ApiKeyHandler{
		apiKeyUsecase: apiKeyUsecase,
	}
}

// CreateApiKey creates a new API key
func (h *ApiKeyHandler) CreateApiKey(c *gin.Context) {
	var input entities.CreateApiKeyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	response, err := h.apiKeyUsecase.CreateApiKey(c.Request.Context(), userID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// ListApiKeys lists API keys for the current user
func (h *ApiKeyHandler) ListApiKeys(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	apiKeys, err := h.apiKeyUsecase.ListApiKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apiKeys)
}

// RevokeApiKey revokes an API key
func (h *ApiKeyHandler) RevokeApiKey(c *gin.Context) {
	idParam := c.Param("id")
	apiKeyID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API Key ID"})
		return
	}

	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.apiKeyUsecase.RevokeApiKey(c.Request.Context(), apiKeyID, userID); err != nil {
		// Basic error mapping - ideally map domain errors to 404/403/500
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API Key revoked successfully"})
}
