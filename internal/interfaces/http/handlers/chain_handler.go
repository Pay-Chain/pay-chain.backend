package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
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
func (h *ChainHandler) ListChains(c *gin.Context) {
	chains, err := h.chainRepo.GetActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list chains"})
		return
	}

	// Format response with CAIP-2 IDs
	type chainResponse struct {
		ID          int    `json:"id"`
		CAIP2       string `json:"caip2"`
		Name        string `json:"name"`
		ChainType   string `json:"chainType"`
		ExplorerURL string `json:"explorerUrl"`
		Symbol      string `json:"symbol"`
		LogoURL     string `json:"logoUrl"`
		IsActive    bool   `json:"isActive"`
	}

	var response []chainResponse
	for _, chain := range chains {
		response = append(response, chainResponse{
			ID:          chain.ID,
			CAIP2:       chain.GetCAIP2ID(),
			Name:        chain.Name,
			ChainType:   string(chain.ChainType),
			ExplorerURL: chain.ExplorerURL,
			Symbol:      chain.Symbol,
			LogoURL:     chain.LogoURL,
			IsActive:    chain.IsActive,
		})
	}

	if response == nil {
		response = []chainResponse{}
	}

	c.JSON(http.StatusOK, gin.H{"chains": response})
}

// CreateChain creates a new chain (Admin only)
// POST /api/v1/admin/chains
func (h *ChainHandler) CreateChain(c *gin.Context) {
	var input struct {
		ID          int    `json:"id" binding:"required"` // Chain ID from RPC
		Name        string `json:"name" binding:"required"`
		ChainType   string `json:"chainType" binding:"required"` // EVM, SVM
		RPCURL      string `json:"rpcUrl" binding:"required"`
		ExplorerURL string `json:"explorerUrl" binding:"required"`
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
		ID:          input.ID,
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
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	var input struct {
		Name        string `json:"name" binding:"required"`
		ChainType   string `json:"chainType" binding:"required"`
		RPCURL      string `json:"rpcUrl" binding:"required"`
		ExplorerURL string `json:"explorerUrl" binding:"required"`
		Symbol      string `json:"symbol" binding:"required"`
		LogoURL     string `json:"logoUrl"`
		IsActive    bool   `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
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
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
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
