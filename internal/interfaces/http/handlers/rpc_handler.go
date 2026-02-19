package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/pkg/utils"
)

// RpcHandler handles RPC endpoints
type RpcHandler struct {
	chainRepo repositories.ChainRepository
}

// NewRpcHandler creates a new RPC handler
func NewRpcHandler(chainRepo repositories.ChainRepository) *RpcHandler {
	return &RpcHandler{chainRepo: chainRepo}
}

// ListRPCs lists all RPCs with filtering
// GET /api/v1/admin/rpcs
func (h *RpcHandler) ListRPCs(c *gin.Context) {
	var chainID *uuid.UUID
	if cidStr := c.Query("chainId"); cidStr != "" {
		cid, err := uuid.Parse(cidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chainId"})
			return
		}
		chainID = &cid
	}

	var isActive *bool
	if activeStr := c.Query("isActive"); activeStr != "" {
		active, err := strconv.ParseBool(activeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid isActive"})
			return
		}
		isActive = &active
	}

	var search *string
	if s := c.Query("search"); s != "" {
		search = &s
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	pagination := utils.GetPaginationParams(page, limit)

	rpcs, totalCount, err := h.chainRepo.GetAllRPCs(c.Request.Context(), chainID, isActive, search, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list RPCs"})
		return
	}

	meta := utils.CalculateMeta(totalCount, pagination.Page, pagination.Limit)
	c.JSON(http.StatusOK, gin.H{
		"items": rpcs,
		"meta":  meta,
	})
}

// CreateRPC creates a new RPC
// POST /api/v1/admin/rpcs
func (h *RpcHandler) CreateRPC(c *gin.Context) {
	var input struct {
		ChainID  string `json:"chainId" binding:"required"`
		URL      string `json:"url" binding:"required"`
		Priority int    `json:"priority"`
		IsActive bool   `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chainUUID, err := uuid.Parse(input.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chainId"})
		return
	}

	rpc := &entities.ChainRPC{
		ID:        utils.GenerateUUIDv7(),
		ChainID:   chainUUID,
		URL:       input.URL,
		Priority:  input.Priority,
		IsActive:  input.IsActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.chainRepo.CreateRPC(c.Request.Context(), rpc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create RPC"})
		return
	}

	c.JSON(http.StatusCreated, rpc)
}

// UpdateRPC updates an RPC
// PUT /api/v1/admin/rpcs/:id
func (h *RpcHandler) UpdateRPC(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid RPC UUID"})
		return
	}

	var input struct {
		ChainID  string `json:"chainId" binding:"required"`
		URL      string `json:"url" binding:"required"`
		Priority int    `json:"priority"`
		IsActive bool   `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chainUUID, err := uuid.Parse(input.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chainId"})
		return
	}

	existingRPC, err := h.chainRepo.GetRPCByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "RPC not found"})
		return
	}

	existingRPC.ChainID = chainUUID
	existingRPC.URL = input.URL
	existingRPC.Priority = input.Priority
	existingRPC.IsActive = input.IsActive
	existingRPC.UpdatedAt = time.Now()

	if err := h.chainRepo.UpdateRPC(c.Request.Context(), existingRPC); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update RPC"})
		return
	}

	c.JSON(http.StatusOK, existingRPC)
}

// DeleteRPC deletes an RPC
// DELETE /api/v1/admin/rpcs/:id
func (h *RpcHandler) DeleteRPC(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid RPC UUID"})
		return
	}

	if err := h.chainRepo.DeleteRPC(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete RPC"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "RPC deleted"})
}
