package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/interfaces/http/response"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, userID uuid.UUID, input *entities.CreatePaymentInput) (*entities.CreatePaymentResponse, error)
	GetPayment(ctx context.Context, id uuid.UUID) (*entities.Payment, error)
	GetPaymentsByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*entities.Payment, int, error)
	GetPaymentEvents(ctx context.Context, paymentID uuid.UUID) ([]*entities.PaymentEvent, error)
	GetPaymentPrivacyStatus(ctx context.Context, paymentID uuid.UUID) (*entities.PaymentPrivacyStatus, error)
	BuildRetryPrivacyRecoveryTx(ctx context.Context, paymentID uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error)
	BuildClaimPrivacyRecoveryTx(ctx context.Context, paymentID uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error)
	BuildRefundPrivacyRecoveryTx(ctx context.Context, paymentID uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error)
}

// PaymentHandler handles payment endpoints
type PaymentHandler struct {
	paymentUsecase PaymentService
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentUsecase PaymentService) *PaymentHandler {
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

// GetPaymentPrivacyStatus gets inferred privacy lifecycle status for a payment
// GET /api/v1/payments/:id/privacy-status
func (h *PaymentHandler) GetPaymentPrivacyStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid payment ID"))
		return
	}

	privacyStatus, err := h.paymentUsecase.GetPaymentPrivacyStatus(c.Request.Context(), id)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Payment not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"privacyStatus": privacyStatus})
}

// RetryPrivacyForward builds tx payload for retryPrivacyForward(bytes32)
// POST /api/v1/payments/:id/privacy/retry
func (h *PaymentHandler) RetryPrivacyForward(c *gin.Context) {
	id, ok := parsePaymentIDParam(c)
	if !ok {
		return
	}

	var req entities.PaymentPrivacyRecoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	txData, err := h.paymentUsecase.BuildRetryPrivacyRecoveryTx(c.Request.Context(), id, req.OnchainPaymentID)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Payment not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"txData": txData})
}

// ClaimPrivacyEscrow builds tx payload for claimPrivacyEscrow(bytes32)
// POST /api/v1/payments/:id/privacy/claim
func (h *PaymentHandler) ClaimPrivacyEscrow(c *gin.Context) {
	id, ok := parsePaymentIDParam(c)
	if !ok {
		return
	}

	var req entities.PaymentPrivacyRecoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	txData, err := h.paymentUsecase.BuildClaimPrivacyRecoveryTx(c.Request.Context(), id, req.OnchainPaymentID)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Payment not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"txData": txData})
}

// RefundPrivacyEscrow builds tx payload for refundPrivacyEscrow(bytes32)
// POST /api/v1/payments/:id/privacy/refund
func (h *PaymentHandler) RefundPrivacyEscrow(c *gin.Context) {
	id, ok := parsePaymentIDParam(c)
	if !ok {
		return
	}

	var req entities.PaymentPrivacyRecoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	txData, err := h.paymentUsecase.BuildRefundPrivacyRecoveryTx(c.Request.Context(), id, req.OnchainPaymentID)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Payment not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"txData": txData})
}

func parsePaymentIDParam(c *gin.Context) (uuid.UUID, bool) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid payment ID"))
		return uuid.Nil, false
	}
	return id, true
}
