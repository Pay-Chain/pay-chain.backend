package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/domain/entities"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Note: We don't necessarily need UserID from context because the Usecase logic
	// resolves User logic based on `SenderWalletAddress` in the input.
	// DualAuthMiddleware ensures the request is authenticated/authorized to reach here.

	response, err := h.paymentAppUsecase.CreatePaymentApp(c.Request.Context(), &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}
