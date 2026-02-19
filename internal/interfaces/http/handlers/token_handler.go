package handlers

import (
	"net/http"
	"strconv"

	"github.com/volatiletech/null/v8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/pkg/utils"
)

// TokenHandler handles token endpoints
type TokenHandler struct {
	tokenRepo repositories.TokenRepository
	chainRepo repositories.ChainRepository
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(tokenRepo repositories.TokenRepository, chainRepo repositories.ChainRepository) *TokenHandler {
	return &TokenHandler{
		tokenRepo: tokenRepo,
		chainRepo: chainRepo,
	}
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
			// Try lookup by legacy blockchain ID
			chain, err := h.chainRepo.GetByChainID(c.Request.Context(), chainIDStr)
			if err != nil {
				response.Error(c, domainerrors.BadRequest("Invalid chainId"))
				return
			}
			chainID = chain.ID
		}

		tokens, totalCount, err := h.tokenRepo.GetTokensByChain(c.Request.Context(), chainID, pagination)
		if err != nil {
			response.Error(c, err)
			return
		}

		meta := utils.CalculateMeta(totalCount, pagination.Page, pagination.Limit)
		response.Success(c, http.StatusOK, gin.H{
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

	tokens, totalCount, err := h.tokenRepo.GetAllTokens(c.Request.Context(), nil, searchPtr, pagination)
	if err != nil {
		response.Error(c, err)
		return
	}

	meta := utils.CalculateMeta(totalCount, pagination.Page, pagination.Limit)
	response.Success(c, http.StatusOK, gin.H{
		"items": tokens,
		"meta":  meta,
	})
}

// ListStablecoins lists only stablecoin tokens
// GET /api/v1/tokens/stablecoins
func (h *TokenHandler) ListStablecoins(c *gin.Context) {
	tokens, err := h.tokenRepo.GetStablecoins(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"tokens": tokens})
}

// CreateToken creates a new token
// POST /api/v1/admin/tokens
func (h *TokenHandler) CreateToken(c *gin.Context) {
	var req struct {
		Symbol          string  `json:"symbol" binding:"required"`
		Name            string  `json:"name" binding:"required"`
		Decimals        int     `json:"decimals" binding:"required"`
		LogoURL         string  `json:"logoUrl"`
		Type            string  `json:"type" binding:"required"`
		ChainID         string  `json:"chainId" binding:"required"`
		ContractAddress string  `json:"contractAddress"`
		MinAmount       string  `json:"minAmount"`
		MaxAmount       *string `json:"maxAmount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	// Sanitize MinAmount
	if req.MinAmount == "" {
		req.MinAmount = "0"
	}

	// Sanitize MaxAmount: treat empty string as nil
	if req.MaxAmount != nil && *req.MaxAmount == "" {
		req.MaxAmount = nil
	}

	chainID, err := uuid.Parse(req.ChainID)
	if err != nil {
		// Try lookup by legacy blockchain ID
		chain, err := h.chainRepo.GetByChainID(c.Request.Context(), req.ChainID)
		if err != nil {
			response.Error(c, domainerrors.BadRequest("Invalid chainId"))
			return
		}
		chainID = chain.ID
	}

	token := &entities.Token{
		ID:              utils.GenerateUUIDv7(),
		Symbol:          req.Symbol,
		Name:            req.Name,
		Decimals:        req.Decimals,
		LogoURL:         req.LogoURL,
		Type:            entities.TokenType(req.Type),
		ChainUUID:       chainID,
		ContractAddress: req.ContractAddress,
		MinAmount:       req.MinAmount,
		MaxAmount:       null.StringFromPtr(req.MaxAmount),
		IsActive:        true,
	}

	if err := h.tokenRepo.Create(c.Request.Context(), token); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"token": token})
}

// UpdateToken updates an existing token
// PUT /api/v1/admin/tokens/:id
func (h *TokenHandler) UpdateToken(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid token ID"))
		return
	}

	var req struct {
		Symbol          string  `json:"symbol"`
		Name            string  `json:"name"`
		Decimals        int     `json:"decimals"`
		LogoURL         string  `json:"logoUrl"`
		Type            string  `json:"type"`
		ContractAddress string  `json:"contractAddress"`
		ChainID         string  `json:"chainId"`
		MinAmount       string  `json:"minAmount"`
		MaxAmount       *string `json:"maxAmount"` // Use pointer to distinguish between missing field and explicit null/empty
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	token, err := h.tokenRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, domainerrors.NotFound("Token not found"))
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
	if req.MinAmount != "" {
		token.MinAmount = req.MinAmount
	}

	// Handle MaxAmount
	if req.MaxAmount != nil {
		if *req.MaxAmount == "" {
			// If explicitly set to empty string, set to null
			token.MaxAmount = null.String{}
		} else {
			token.MaxAmount = null.StringFromPtr(req.MaxAmount)
		}
	}

	if req.ChainID != "" {
		chainID, err := uuid.Parse(req.ChainID)
		if err != nil {
			// Try lookup by legacy blockchain ID
			chain, err := h.chainRepo.GetByChainID(c.Request.Context(), req.ChainID)
			if err == nil {
				chainID = chain.ID
			}
		}
		if chainID != uuid.Nil {
			token.ChainUUID = chainID
		}
	}

	if err := h.tokenRepo.Update(c.Request.Context(), token); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"token": token})
}

// DeleteToken soft deletes a token
// DELETE /api/v1/admin/tokens/:id
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid token ID"))
		return
	}

	if err := h.tokenRepo.SoftDelete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Token deleted successfully"})
}
