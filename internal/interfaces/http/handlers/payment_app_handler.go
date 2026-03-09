package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/interfaces/http/response"
)

type PaymentAppService interface {
	CreatePaymentApp(ctx context.Context, input *entities.CreatePaymentAppInput) (*entities.CreatePaymentResponse, error)
}

type PaymentAppHandler struct {
	paymentAppUsecase PaymentAppService
}

func NewPaymentAppHandler(paymentAppUsecase PaymentAppService) *PaymentAppHandler {
	return &PaymentAppHandler{
		paymentAppUsecase: paymentAppUsecase,
	}
}

// CreatePaymentApp handles payment requests from the public app
func (h *PaymentAppHandler) CreatePaymentApp(c *gin.Context) {
	var input entities.CreatePaymentAppInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	// Note: We don't necessarily need UserID from context because the Usecase logic
	// resolves User logic based on `SenderWalletAddress` in the input.
	// DualAuthMiddleware ensures the request is authenticated/authorized to reach here.

	result, err := h.paymentAppUsecase.CreatePaymentApp(c.Request.Context(), &input)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, 201, result)
}
