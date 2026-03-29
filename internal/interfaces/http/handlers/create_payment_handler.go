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

type createPaymentRequest struct {
	MerchantID      string `json:"merchant_id"`
	ChainID         string `json:"chain_id" binding:"required"`
	SelectedToken   string `json:"selected_token" binding:"required"`
	PricingType     string `json:"pricing_type" binding:"required"`
	RequestedAmount string `json:"requested_amount"`
	PaymentAmount   string `json:"payment_amount"`
	ExpiresIn       string `json:"expires_in"`
}

type createPaymentUsecase interface {
	CreatePayment(ctx context.Context, input *usecases.CreatePaymentInput) (*usecases.CreatePaymentOutput, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*usecases.CreatePaymentOutput, error)
}

func NewCreatePaymentHandler(usecase createPaymentUsecase) *CreatePaymentHandler {
	return &CreatePaymentHandler{usecase: usecase}
}

func parseCreatePaymentRequest(c *gin.Context) (*createPaymentRequest, error) {
	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, domainerrors.BadRequest(err.Error())
	}
	return &req, nil
}

func parseOptionalMerchantID(raw string) (uuid.UUID, error) {
	if strings.TrimSpace(raw) == "" {
		return uuid.Nil, nil
	}
	parsedID, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, domainerrors.BadRequest("merchant_id must be a valid UUID")
	}
	return parsedID, nil
}

func requestedAmountFromRequest(req *createPaymentRequest) string {
	requestedAmount := strings.TrimSpace(req.RequestedAmount)
	if requestedAmount != "" {
		return requestedAmount
	}
	return strings.TrimSpace(req.PaymentAmount)
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

	req, err := parseCreatePaymentRequest(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	merchantID, err := parseOptionalMerchantID(req.MerchantID)
	if err != nil {
		response.Error(c, err)
		return
	}

	out, err := h.usecase.CreatePayment(c.Request.Context(), &usecases.CreatePaymentInput{
		MerchantContextID: merchantContextID,
		MerchantID:        merchantID,
		ChainID:           req.ChainID,
		SelectedToken:     req.SelectedToken,
		PricingType:       req.PricingType,
		RequestedAmount:   requestedAmountFromRequest(req),
		ExpiresIn:         strings.TrimSpace(req.ExpiresIn),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

func (h *CreatePaymentHandler) CreatePaymentAdmin(c *gin.Context) {
	if h == nil || h.usecase == nil {
		response.Error(c, domainerrors.InternalServerError("create payment handler is not configured"))
		return
	}

	merchantID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil || merchantID == uuid.Nil {
		response.Error(c, domainerrors.BadRequest("merchant id must be a valid UUID"))
		return
	}

	req, err := parseCreatePaymentRequest(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	requestMerchantID, err := parseOptionalMerchantID(req.MerchantID)
	if err != nil {
		response.Error(c, err)
		return
	}
	if requestMerchantID != uuid.Nil && requestMerchantID != merchantID {
		response.Error(c, domainerrors.BadRequest("merchant_id in body must match merchant id in path"))
		return
	}

	out, err := h.usecase.CreatePayment(c.Request.Context(), &usecases.CreatePaymentInput{
		MerchantContextID: merchantID,
		MerchantID:        merchantID,
		ChainID:           req.ChainID,
		SelectedToken:     req.SelectedToken,
		PricingType:       req.PricingType,
		RequestedAmount:   requestedAmountFromRequest(req),
		ExpiresIn:         strings.TrimSpace(req.ExpiresIn),
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
