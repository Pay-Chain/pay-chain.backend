package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/interfaces/http/response"
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
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, domainerrors.Unauthorized("User not authenticated"))
		return
	}

	res, err := h.merchantUsecase.ApplyMerchant(c.Request.Context(), userID, &input)
	if err != nil {
		if err == domainerrors.ErrBadRequest {
			response.Error(c, domainerrors.BadRequest("Invalid merchant type"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, res)
}

// GetMerchantStatus gets merchant status for the current user
// GET /api/v1/merchants/status
func (h *MerchantHandler) GetMerchantStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, domainerrors.Unauthorized("User not authenticated"))
		return
	}

	res, err := h.merchantUsecase.GetMerchantStatus(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}
