package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	domainrepos "payment-kita.backend/internal/domain/repositories"
	domainservices "payment-kita.backend/internal/domain/services"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/interfaces/http/response"
	"payment-kita.backend/internal/usecases"
)

type PartnerPaymentSessionHandler struct {
	usecase           partnerPaymentSessionUsecase
	complianceService domainservices.ComplianceService
	auditRepo         domainrepos.AuditRepository
}

type partnerPaymentSessionUsecase interface {
	CreateSession(ctx context.Context, input *usecases.CreatePartnerPaymentSessionInput) (*usecases.CreatePartnerPaymentSessionOutput, error)
	GetSession(ctx context.Context, sessionID uuid.UUID) (*usecases.GetPartnerPaymentSessionOutput, error)
	ResolvePaymentCode(ctx context.Context, input *usecases.ResolvePartnerPaymentCodeInput) (*usecases.ResolvePartnerPaymentCodeOutput, error)
}

func NewPartnerPaymentSessionHandler(
	usecase partnerPaymentSessionUsecase,
	complianceService domainservices.ComplianceService,
	auditRepo domainrepos.AuditRepository,
) *PartnerPaymentSessionHandler {
	return &PartnerPaymentSessionHandler{usecase: usecase, complianceService: complianceService, auditRepo: auditRepo}
}

func (h *PartnerPaymentSessionHandler) CreateSession(c *gin.Context) {
	if h == nil || h.usecase == nil {
		response.Error(c, domainerrors.InternalServerError("partner payment session handler is not configured"))
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
		QuoteID    string                 `json:"quote_id" binding:"required"`
		DestWallet string                 `json:"dest_wallet" binding:"required"`
		Metadata   map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	quoteID, err := uuid.Parse(req.QuoteID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("quote_id must be a valid UUID"))
		return
	}

	out, err := h.usecase.CreateSession(c.Request.Context(), &usecases.CreatePartnerPaymentSessionInput{
		MerchantID: merchantID,
		QuoteID:    quoteID,
		DestWallet: req.DestWallet,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, out)
}

func (h *PartnerPaymentSessionHandler) GetSession(c *gin.Context) {
	if h == nil || h.usecase == nil {
		response.Error(c, domainerrors.InternalServerError("partner payment session handler is not configured"))
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("payment session id must be a valid UUID"))
		return
	}

	out, err := h.usecase.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, out)
}

func (h *PartnerPaymentSessionHandler) ResolvePaymentCode(c *gin.Context) {
	if h == nil || h.usecase == nil {
		response.Error(c, domainerrors.InternalServerError("partner payment session handler is not configured"))
		return
	}

	var req struct {
		PaymentCode string `json:"payment_code" binding:"required"`
		PayerWallet string `json:"payer_wallet"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	riskScore := 0
	riskLevel := "UNKNOWN"
	if stringsTrim(req.PayerWallet) != "" && h.complianceService != nil {
		score, level, complianceErr := h.complianceService.ValidatePayer(c.Request.Context(), req.PayerWallet)
		riskScore = score
		riskLevel = level
		if complianceErr != nil || score > 80 {
			h.logResolveAudit(c, uuid.Nil, req.PayerWallet, riskScore, riskLevel, "BLOCKED", "High risk or compliance error")
			response.Error(c, domainerrors.Forbidden("compliance check failed: risk too high"))
			return
		}
	}

	out, err := h.usecase.ResolvePaymentCode(c.Request.Context(), &usecases.ResolvePartnerPaymentCodeInput{
		PaymentCode: req.PaymentCode,
		PayerWallet: req.PayerWallet,
	})
	if err != nil {
		h.logResolveAudit(c, uuid.Nil, req.PayerWallet, riskScore, riskLevel, "FAILED", err.Error())
		response.Error(c, err)
		return
	}

	if parsedID, parseErr := uuid.Parse(out.PaymentID); parseErr == nil {
		h.logResolveAudit(c, parsedID, req.PayerWallet, riskScore, riskLevel, "SUCCESS", "")
	}
	response.Success(c, http.StatusOK, out)
}

func (h *PartnerPaymentSessionHandler) logResolveAudit(c *gin.Context, sessionID uuid.UUID, wallet string, score int, level string, status string, reason string) {
	if h.auditRepo == nil || sessionID == uuid.Nil {
		return
	}
	audit := &entities.ResolveAudit{
		ID:            uuid.New(),
		SessionID:     sessionID,
		WalletAddress: wallet,
		RiskScore:     score,
		RiskLevel:     level,
		UserAgent:     c.Request.UserAgent(),
		IPAddress:     c.ClientIP(),
		Status:        status,
		Reason:        reason,
		CreatedAt:     time.Now(),
	}
	_ = h.auditRepo.LogResolveAttempt(c.Request.Context(), audit)
}

func stringsTrim(v string) string {
	return strings.TrimSpace(v)
}
