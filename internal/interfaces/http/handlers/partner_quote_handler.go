package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/interfaces/http/response"
	"payment-kita.backend/internal/usecases"
)

type PartnerQuoteHandler struct {
	usecase partnerQuoteUsecase
}

type partnerQuoteUsecase interface {
	CreateQuote(ctx context.Context, input *usecases.CreatePartnerQuoteInput) (*usecases.CreatePartnerQuoteOutput, error)
}

func NewPartnerQuoteHandler(usecase partnerQuoteUsecase) *PartnerQuoteHandler {
	return &PartnerQuoteHandler{usecase: usecase}
}

func (h *PartnerQuoteHandler) CreateQuote(c *gin.Context) {
	if h == nil || h.usecase == nil {
		response.Error(c, domainerrors.InternalServerError("partner quote handler is not configured"))
		return
	}

	merchantValue, exists := c.Get(middleware.MerchantIDKey)
	if !exists {
		response.Error(c, domainerrors.Forbidden("merchant context required"))
		return
	}

	merchantID, ok := merchantValue.(uuid.UUID)
	if !ok || merchantID == uuid.Nil {
		response.Error(c, domainerrors.Forbidden("merchant context required"))
		return
	}

	var req struct {
		InvoiceCurrency string                 `json:"invoice_currency" binding:"required"`
		InvoiceAmount   string                 `json:"invoice_amount" binding:"required"`
		SelectedChain   string                 `json:"selected_chain" binding:"required"`
		SelectedToken   string                 `json:"selected_token" binding:"required"`
		DestWallet      string                 `json:"dest_wallet" binding:"required"`
		Metadata        map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	out, err := h.usecase.CreateQuote(c.Request.Context(), &usecases.CreatePartnerQuoteInput{
		MerchantID:      merchantID,
		InvoiceCurrency: req.InvoiceCurrency,
		InvoiceAmount:   req.InvoiceAmount,
		SelectedChain:   req.SelectedChain,
		SelectedToken:   req.SelectedToken,
		DestWallet:      req.DestWallet,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, out)
}
