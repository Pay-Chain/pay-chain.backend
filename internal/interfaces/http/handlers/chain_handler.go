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
	"pay-chain.backend/internal/interfaces/http/response"
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
func (h *ChainHandler) ListChains(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	pagination := utils.GetPaginationParams(page, limit)

	chains, totalCount, err := h.chainRepo.GetActive(c.Request.Context(), pagination)
	if err != nil {
		response.Error(c, domainerrors.InternalError(err))
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

	var resp []chainResponse
	for _, chain := range chains {
		resp = append(resp, chainResponse{
			ID:          chain.ID.String(), // Use UUID for the ID field
			NetworkID:   chain.ChainID,     // Keep NetworkID as the string ID
			CAIP2:       chain.GetCAIP2ID(),
			Name:        chain.Name,
			ChainType:   string(chain.Type),
			RPCURL:      chain.RPCURL,
			ExplorerURL: chain.ExplorerURL,
			Symbol:      chain.CurrencySymbol,
			LogoURL:     chain.ImageURL,
			IsActive:    chain.IsActive,
		})
	}

	if resp == nil {
		resp = []chainResponse{}
	}

	meta := utils.CalculateMeta(totalCount, pagination.Page, pagination.Limit)
	response.Success(c, http.StatusOK, gin.H{
		"items": resp,
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
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	chain := &entities.Chain{
		ID:             uuid.New(),
		ChainID:        input.NetworkID,
		Name:           input.Name,
		Type:           entities.ChainType(input.ChainType),
		RPCURL:         input.RPCURL,
		ExplorerURL:    input.ExplorerURL,
		CurrencySymbol: input.Symbol,
		ImageURL:       input.LogoURL,
		IsActive:       true,
		CreatedAt:      time.Now(),
	}

	if err := h.chainRepo.Create(c.Request.Context(), chain); err != nil {
		response.Error(c, domainerrors.InternalError(err))
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"message": "Chain created", "chain": chain})
}

// UpdateChain updates a chain (Admin only)
// PUT /api/v1/admin/chains/:id
func (h *ChainHandler) UpdateChain(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid chain UUID"))
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
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	chain := &entities.Chain{
		ID:             id,
		ChainID:        input.NetworkID,
		Name:           input.Name,
		Type:           entities.ChainType(input.ChainType),
		RPCURL:         input.RPCURL,
		ExplorerURL:    input.ExplorerURL,
		CurrencySymbol: input.Symbol,
		ImageURL:       input.LogoURL,
		IsActive:       input.IsActive,
	}

	if err := h.chainRepo.Update(c.Request.Context(), chain); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Chain not found"))
			return
		}
		response.Error(c, domainerrors.InternalError(err))
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Chain updated", "chain": chain})
}

// DeleteChain deletes a chain (Admin only)
// DELETE /api/v1/admin/chains/:id
func (h *ChainHandler) DeleteChain(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid chain UUID"))
		return
	}

	if err := h.chainRepo.Delete(c.Request.Context(), id); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Chain not found"))
			return
		}
		response.Error(c, domainerrors.InternalError(err))
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Chain deleted"})
}
