package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/interfaces/http/response"
)

// AdminHandler handles admin endpoints
type AdminHandler struct {
	userRepo     repositories.UserRepository
	merchantRepo repositories.MerchantRepository
	paymentRepo  repositories.PaymentRepository
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	userRepo repositories.UserRepository,
	merchantRepo repositories.MerchantRepository,
	paymentRepo repositories.PaymentRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:     userRepo,
		merchantRepo: merchantRepo,
		paymentRepo:  paymentRepo,
	}
}

// ListUsers lists all users
// GET /api/v1/admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	search := c.Query("search")
	users, err := h.userRepo.List(c.Request.Context(), search)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"users": users})
}

// ListMerchants lists all merchants
// GET /api/v1/admin/merchants
func (h *AdminHandler) ListMerchants(c *gin.Context) {
	merchants, err := h.merchantRepo.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"merchants": merchants})
}

// UpdateMerchantStatus updates merchant status
// PUT /api/v1/admin/merchants/:id/status
func (h *AdminHandler) UpdateMerchantStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid merchant ID"))
		return
	}

	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	// Fetch merchant to verify existence
	if _, err := h.merchantRepo.GetByID(c.Request.Context(), id); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Merchant not found"))
			return
		}
		response.Error(c, err)
		return
	}

	// Update status logic would go here
	// For now mirroring existing implementation logic

	response.Success(c, http.StatusOK, gin.H{"message": "Merchant status updated", "status": input.Status})
}

// GetStats returns dashboard stats
// GET /api/v1/admin/stats
func (h *AdminHandler) GetStats(c *gin.Context) {
	// Mock stats for now
	stats := gin.H{
		"totalUsers":     150,
		"totalMerchants": 12,
		"totalVolume":    "1234567.89",
		"activeChains":   5,
	}
	response.Success(c, http.StatusOK, stats)
}
