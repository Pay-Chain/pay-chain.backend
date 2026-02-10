package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/internal/usecases"
)

// PaymentHandler handles payment endpoints
type PaymentHandler struct {
	paymentUsecase *usecases.PaymentUsecase
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentUsecase *usecases.PaymentUsecase) *PaymentHandler {
	return &PaymentHandler{paymentUsecase: paymentUsecase}
}

// CreatePayment creates a new payment
// POST /api/v1/payments
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var input entities.CreatePaymentInput

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, domainerrors.Unauthorized("User not authenticated"))
		return
	}

	createResponse, err := h.paymentUsecase.CreatePayment(c.Request.Context(), userID, &input)
	if err != nil {
		if err == domainerrors.ErrBadRequest {
			response.Error(c, domainerrors.BadRequest("Invalid input"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, createResponse)
}

// GetPayment gets a payment by ID
// GET /api/v1/payments/:id
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid payment ID"))
		return
	}

	payment, err := h.paymentUsecase.GetPayment(c.Request.Context(), id)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Payment not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"payment": payment})
}

// ListPayments lists payments for the current user
// GET /api/v1/payments
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, domainerrors.Unauthorized("User not authenticated"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	payments, total, err := h.paymentUsecase.GetPaymentsByUser(c.Request.Context(), userID, page, limit)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := (total + limit - 1) / limit

	response.Success(c, http.StatusOK, gin.H{
		"payments": payments,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// GetPaymentEvents gets events for a payment
// GET /api/v1/payments/:id/events
func (h *PaymentHandler) GetPaymentEvents(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid payment ID"))
		return
	}

	events, err := h.paymentUsecase.GetPaymentEvents(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"events": events})
}
