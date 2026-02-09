package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/pkg/utils"
)

// TokenHandler handles token endpoints
type TokenHandler struct {
	tokenRepo repositories.TokenRepository
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(tokenRepo repositories.TokenRepository) *TokenHandler {
	return &TokenHandler{tokenRepo: tokenRepo}
}

// ListSupportedTokens lists tokens, optionally filtered by chain
// GET /api/v1/tokens
func (h *TokenHandler) ListSupportedTokens(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	pagination := utils.GetPaginationParams(page, limit)

	chainIDStr := c.Query("chainId")
	search := c.Query("search")

	if chainIDStr != "" {
		// Get tokens supported on specific chain
		chainID, err := uuid.Parse(chainIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chainId"})
			return
		}

		tokens, totalCount, err := h.tokenRepo.GetSupportedByChain(c.Request.Context(), chainID, pagination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens"})
			return
		}

		meta := utils.CalculateMeta(totalCount, pagination.Page, pagination.Limit)
		c.JSON(http.StatusOK, gin.H{
			"items": tokens,
			"meta":  meta,
		})
		return
	}

	// Get all supported tokens (admin view typically)
	var searchPtr *string
	if search != "" {
		searchPtr = &search
	}

	tokens, totalCount, err := h.tokenRepo.GetAllSupported(c.Request.Context(), nil, searchPtr, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens"})
		return
	}

	meta := utils.CalculateMeta(totalCount, pagination.Page, pagination.Limit)
	c.JSON(http.StatusOK, gin.H{
		"items": tokens,
		"meta":  meta,
	})
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

// CreateToken creates a new token
// POST /api/v1/admin/tokens
func (h *TokenHandler) CreateToken(c *gin.Context) {
	var req struct {
		Symbol          string `json:"symbol" binding:"required"`
		Name            string `json:"name" binding:"required"`
		Decimals        int    `json:"decimals" binding:"required"`
		LogoURL         string `json:"logoUrl"`
		Type            string `json:"type" binding:"required"`
		ChainID         string `json:"chainId" binding:"required"`
		ContractAddress string `json:"contractAddress"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chainID, err := uuid.Parse(req.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chainId"})
		return
	}

	token := &entities.Token{
		ID:              uuid.New(),
		Symbol:          req.Symbol,
		Name:            req.Name,
		Decimals:        req.Decimals,
		LogoURL:         req.LogoURL,
		Type:            entities.TokenType(req.Type),
		ChainID:         chainID,
		ContractAddress: req.ContractAddress,
		IsActive:        true,
	}

	if err := h.tokenRepo.Create(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

// UpdateToken updates an existing token
// PUT /api/v1/admin/tokens/:id
func (h *TokenHandler) UpdateToken(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	var req struct {
		Symbol          string `json:"symbol"`
		Name            string `json:"name"`
		Decimals        int    `json:"decimals"`
		LogoURL         string `json:"logoUrl"`
		Type            string `json:"type"`
		ContractAddress string `json:"contractAddress"`
		ChainID         string `json:"chainId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.tokenRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	if req.Symbol != "" {
		token.Symbol = req.Symbol
	}
	if req.Name != "" {
		token.Name = req.Name
	}
	if req.Decimals != 0 {
		token.Decimals = req.Decimals
	}
	if req.LogoURL != "" {
		token.LogoURL = req.LogoURL
	}
	if req.Type != "" {
		token.Type = entities.TokenType(req.Type)
	}
	if req.ContractAddress != "" {
		token.ContractAddress = req.ContractAddress
	}
	if req.ChainID != "" {
		chainID, err := uuid.Parse(req.ChainID)
		if err == nil {
			token.ChainID = chainID
		}
	}

	if err := h.tokenRepo.Update(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// DeleteToken soft deletes a token
// DELETE /api/v1/admin/tokens/:id
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	if err := h.tokenRepo.SoftDelete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token deleted successfully"})
}
