package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/domain/repositories"
)

// TokenHandler handles token endpoints
type TokenHandler struct {
	tokenRepo repositories.TokenRepository
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(tokenRepo repositories.TokenRepository) *TokenHandler {
	return &TokenHandler{tokenRepo: tokenRepo}
}

// ListTokens lists tokens, optionally filtered by chain
// GET /api/v1/tokens
func (h *TokenHandler) ListTokens(c *gin.Context) {
	chainIDStr := c.Query("chainId")

	if chainIDStr != "" {
		// Get tokens supported on specific chain
		chainID, err := strconv.Atoi(chainIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chainId"})
			return
		}

		tokens, err := h.tokenRepo.GetSupportedByChain(c.Request.Context(), chainID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"tokens": tokens})
		return
	}

	// Get all tokens
	tokens, err := h.tokenRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// ListStablecoins lists only stablecoin tokens
// GET /api/v1/tokens/stablecoins
func (h *TokenHandler) ListStablecoins(c *gin.Context) {
	tokens, err := h.tokenRepo.GetStablecoins(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list stablecoins"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}
