package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
