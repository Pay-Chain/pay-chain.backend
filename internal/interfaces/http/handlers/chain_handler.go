package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/pkg/utils"
)

// ChainHandler handles chain endpoints
type ChainHandler struct {
	chainRepo repositories.ChainRepository
}

// NewChainHandler creates a new chain handler
func NewChainHandler(chainRepo repositories.ChainRepository) *ChainHandler {
	return &ChainHandler{chainRepo: chainRepo}
}

// ListChains lists all active chains
// GET /api/v1/chains
// ListChains lists all active chains
// GET /api/v1/chains
func (h *ChainHandler) ListChains(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	pagination := utils.GetPaginationParams(page, limit)

	chains, totalCount, err := h.chainRepo.GetActive(c.Request.Context(), pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list chains"})
		return
	}

	// Format response with CAIP-2 IDs
	type chainResponse struct {
		ID          string `json:"id"`
		NetworkID   string `json:"networkId"`
		CAIP2       string `json:"caip2"`
		Name        string `json:"name"`
		ChainType   string `json:"chainType"`
		RPCURL      string `json:"rpcUrl"`
		ExplorerURL string `json:"explorerUrl"`
		Symbol      string `json:"symbol"`
		LogoURL     string `json:"logoUrl"`
		IsActive    bool   `json:"isActive"`
	}

	var response []chainResponse
	for _, chain := range chains {
		response = append(response, chainResponse{
			ID:          chain.ID.String(),
			NetworkID:   chain.NetworkID,
			CAIP2:       chain.GetCAIP2ID(),
			Name:        chain.Name,
			ChainType:   string(chain.ChainType),
			RPCURL:      chain.RPCURL,
			ExplorerURL: chain.ExplorerURL,
			Symbol:      chain.Symbol,
			LogoURL:     chain.LogoURL,
			IsActive:    chain.IsActive,
		})
	}

	if response == nil {
		response = []chainResponse{}
	}

	meta := utils.CalculateMeta(totalCount, pagination.Page, pagination.Limit)
	c.JSON(http.StatusOK, gin.H{
		"items": response,
		"meta":  meta,
	})
}

// CreateChain creates a new chain (Admin only)
// POST /api/v1/admin/chains
func (h *ChainHandler) CreateChain(c *gin.Context) {
	var input struct {
		NetworkID   string `json:"networkId" binding:"required"` // External Chain ID (e.g. "1", "solana:5ey...")
		Name        string `json:"name" binding:"required"`
		ChainType   string `json:"chainType" binding:"required"` // EVM, SVM
		RPCURL      string `json:"rpcUrl" binding:"required"`
		ExplorerURL string `json:"explorerUrl"`
		Symbol      string `json:"symbol" binding:"required"`
		LogoURL     string `json:"logoUrl"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assuming we map input to entity
	// Note: We might need to handle CAIP-2 namespace logic here or in repo
	namespace := "eip155"
	if input.ChainType == "SVM" {
		namespace = "solana"
	}

	chain := &entities.Chain{
		ID:          uuid.New(),
		NetworkID:   input.NetworkID,
		Namespace:   namespace,
		Name:        input.Name,
		ChainType:   entities.ChainType(input.ChainType),
		RPCURL:      input.RPCURL,
		ExplorerURL: input.ExplorerURL,
		Symbol:      input.Symbol,
		LogoURL:     input.LogoURL,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	if err := h.chainRepo.Create(c.Request.Context(), chain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chain"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Chain created", "chain": chain})
}

// UpdateChain updates a chain (Admin only)
// PUT /api/v1/admin/chains/:id
func (h *ChainHandler) UpdateChain(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain UUID"})
		return
	}

	var input struct {
		NetworkID   string `json:"networkId" binding:"required"`
		Name        string `json:"name" binding:"required"`
		ChainType   string `json:"chainType" binding:"required"`
		RPCURL      string `json:"rpcUrl" binding:"required"`
		ExplorerURL string `json:"explorerUrl"`
		Symbol      string `json:"symbol"`
		LogoURL     string `json:"logoUrl"`
		IsActive    bool   `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		fmt.Printf("UpdateChain validation error: %v\n", err) // Added logging
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine namespace based on chain type
	namespace := "eip155"
	if input.ChainType == "SVM" {
		namespace = "solana"
	}

	chain := &entities.Chain{
		ID:          id,
		NetworkID:   input.NetworkID,
		Namespace:   namespace,
		Name:        input.Name,
		ChainType:   entities.ChainType(input.ChainType),
		RPCURL:      input.RPCURL,
		ExplorerURL: input.ExplorerURL,
		Symbol:      input.Symbol,
		LogoURL:     input.LogoURL,
		IsActive:    input.IsActive,
	}

	if err := h.chainRepo.Update(c.Request.Context(), chain); err != nil {
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chain updated", "chain": chain})
}

// DeleteChain deletes a chain (Admin only)
// DELETE /api/v1/admin/chains/:id
func (h *ChainHandler) DeleteChain(c *gin.Context) {
	idStr := c.Param("id")
	// Parse ID
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain UUID"})
		return
	}

	if err := h.chainRepo.Delete(c.Request.Context(), id); err != nil {
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chain deleted"})
}
