package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/interfaces/http/response"
	"payment-kita.backend/internal/usecases"
)

type CreatePaymentHandler struct {
	usecase createPaymentUsecase
}

type createPaymentUsecase interface {
	CreatePayment(ctx context.Context, input *usecases.CreatePaymentInput) (*usecases.CreatePaymentOutput, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*usecases.CreatePaymentOutput, error)
}

func NewCreatePaymentHandler(usecase createPaymentUsecase) *CreatePaymentHandler {
	return &CreatePaymentHandler{usecase: usecase}
}

func (h *CreatePaymentHandler) CreatePayment(c *gin.Context) {
	if h == nil || h.usecase == nil {
		response.Error(c, domainerrors.InternalServerError("create payment handler is not configured"))
		return
	}
	merchantValue, exists := c.Get(middleware.MerchantIDKey)
	if !exists {
		response.Error(c, domainerrors.Forbidden("merchant context required"))
		return
	}
	merchantContextID, ok := merchantValue.(uuid.UUID)
	if !ok || merchantContextID == uuid.Nil {
		response.Error(c, domainerrors.Forbidden("merchant context required"))
		return
	}

	var req struct {
		MerchantID      string `json:"merchant_id"`
		ChainID         string `json:"chain_id" binding:"required"`
		SelectedToken   string `json:"selected_token" binding:"required"`
		PricingType     string `json:"pricing_type" binding:"required"`
		RequestedAmount string `json:"requested_amount"`
		PaymentAmount   string `json:"payment_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}
	requestedAmount := strings.TrimSpace(req.RequestedAmount)
	if requestedAmount == "" {
		requestedAmount = strings.TrimSpace(req.PaymentAmount)
	}

	merchantID := uuid.Nil
	if strings.TrimSpace(req.MerchantID) != "" {
		parsedID, err := uuid.Parse(strings.TrimSpace(req.MerchantID))
		if err != nil {
			response.Error(c, domainerrors.BadRequest("merchant_id must be a valid UUID"))
			return
		}
		merchantID = parsedID
	}

	out, err := h.usecase.CreatePayment(c.Request.Context(), &usecases.CreatePaymentInput{
		MerchantContextID: merchantContextID,
		MerchantID:        merchantID,
		ChainID:           req.ChainID,
		SelectedToken:     req.SelectedToken,
		PricingType:       req.PricingType,
		RequestedAmount:   requestedAmount,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

func (h *CreatePaymentHandler) GetPayment(c *gin.Context) {
	if h == nil || h.usecase == nil {
		response.Error(c, domainerrors.InternalServerError("create payment handler is not configured"))
		return
	}

	paymentID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("payment id must be a valid UUID"))
		return
	}

	out, err := h.usecase.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}
