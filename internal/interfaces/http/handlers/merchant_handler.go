package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
)

// MerchantHandler handles merchant endpoints
type MerchantHandler struct {
	merchantUsecase *usecases.MerchantUsecase
}

// NewMerchantHandler creates a new merchant handler
func NewMerchantHandler(merchantUsecase *usecases.MerchantUsecase) *MerchantHandler {
	return &MerchantHandler{merchantUsecase: merchantUsecase}
}

// ApplyMerchant handles merchant application
// POST /api/v1/merchants/apply
func (h *MerchantHandler) ApplyMerchant(c *gin.Context) {
	var input entities.MerchantApplyInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)

	response, err := h.merchantUsecase.ApplyMerchant(c.Request.Context(), userID, &input)
	if err != nil {
		if err == domainerrors.ErrBadRequest {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchant type"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply as merchant"})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetMerchantStatus gets merchant status for the current user
// GET /api/v1/merchants/status
func (h *MerchantHandler) GetMerchantStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)

	response, err := h.merchantUsecase.GetMerchantStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get merchant status"})
		return
	}

	c.JSON(http.StatusOK, response)
}
