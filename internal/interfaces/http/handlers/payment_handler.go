package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)

	response, err := h.paymentUsecase.CreatePayment(c.Request.Context(), userID, &input)
	if err != nil {
		if err == domainerrors.ErrBadRequest {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment"})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetPayment gets a payment by ID
// GET /api/v1/payments/:id
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment ID"})
		return
	}

	payment, err := h.paymentUsecase.GetPayment(c.Request.Context(), id)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment": payment})
}

// ListPayments lists payments for the current user
// GET /api/v1/payments
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	userID := middleware.GetUserID(c)

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list payments"})
		return
	}

	totalPages := (total + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment ID"})
		return
	}

	events, err := h.paymentUsecase.GetPaymentEvents(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}
