package handlers

import (
	"strconv"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/internal/usecases"
)

type PaymentRequestHandler struct {
	usecase *usecases.PaymentRequestUsecase
}

func NewPaymentRequestHandler(usecase *usecases.PaymentRequestUsecase) *PaymentRequestHandler {
	return &PaymentRequestHandler{usecase: usecase}
}

type CreatePaymentRequestRequest struct {
	ChainID      string `json:"chainId" binding:"required"`
	TokenAddress string `json:"tokenAddress" binding:"required"`
	Amount       string `json:"amount" binding:"required"`
	Decimals     int    `json:"decimals" binding:"required"`
	Description  string `json:"description"`
}

// CreatePaymentRequest creates a new payment request
// POST /api/v1/payment-requests
func (h *PaymentRequestHandler) CreatePaymentRequest(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, domainerrors.Unauthorized("unauthorized"))
		return
	}

	var req CreatePaymentRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	input := usecases.CreatePaymentRequestInput{
		UserID:       userID.(uuid.UUID),
		ChainID:      req.ChainID,
		TokenAddress: req.TokenAddress,
		Amount:       req.Amount,
		Decimals:     req.Decimals,
		Description:  req.Description,
	}

	result, err := h.usecase.CreatePaymentRequest(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, result)
}

// GetPaymentRequest gets a payment request by ID with transaction data
// GET /api/v1/payment-requests/:id
func (h *PaymentRequestHandler) GetPaymentRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid request ID"))
		return
	}

	request, txData, err := h.usecase.GetPaymentRequest(c.Request.Context(), id)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound(err.Error()))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"request": request,
		"txData":  txData,
	})
}

// ListPaymentRequests lists payment requests for the authenticated merchant
// GET /api/v1/payment-requests
func (h *PaymentRequestHandler) ListPaymentRequests(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, domainerrors.Unauthorized("unauthorized"))
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
	offset := (page - 1) * limit

	requests, total, err := h.usecase.ListPaymentRequests(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"requests": requests,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + limit - 1) / limit,
		},
	})
}

// GetPublicPaymentRequest gets a payment request by ID for payers (public)
// GET /api/v1/pay/:id
func (h *PaymentRequestHandler) GetPublicPaymentRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid request ID"))
		return
	}

	request, txData, err := h.usecase.GetPaymentRequest(c.Request.Context(), id)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound(err.Error()))
			return
		}
		response.Error(c, err)
		return
	}

	// Only return public info
	response.Success(c, http.StatusOK, gin.H{
		"requestId":       request.ID,
		"chainId":         request.ChainID,
		"amount":          request.Amount,
		"decimals":        request.Decimals,
		"description":     request.Description,
		"status":          request.Status,
		"expiresAt":       request.ExpiresAt,
		"contractAddress": txData.ContractAddress,
		"txData": gin.H{
			"hex":    txData.Hex,
			"base64": txData.Base64,
		},
	})
}
