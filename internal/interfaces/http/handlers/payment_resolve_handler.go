package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/domain/services"
	"payment-kita.backend/internal/infrastructure/metrics"
	"payment-kita.backend/internal/interfaces/http/response"
	"payment-kita.backend/internal/usecases"
)

type PaymentResolveHandler struct {
	jweService        services.JWEService
	complianceService services.ComplianceService
	auditRepo         repositories.AuditRepository
	usecase           *usecases.PaymentRequestUsecase
}

func NewPaymentResolveHandler(
	jweService services.JWEService,
	complianceService services.ComplianceService,
	auditRepo repositories.AuditRepository,
	usecase *usecases.PaymentRequestUsecase,
) *PaymentResolveHandler {
	return &PaymentResolveHandler{
		jweService:        jweService,
		complianceService: complianceService,
		auditRepo:         auditRepo,
		usecase:           usecase,
	}
}

func (h *PaymentResolveHandler) Resolve(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_code is required"})
		return
	}

	// 1. Decrypt
	payload, err := h.jweService.Decrypt(code)
	if err != nil {
		metrics.RecordJWEDecryptionError("invalid_signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or tampered payment code"})
		return
	}

	// 2. Expiry Check (Safety)
	if time.Now().Unix() > payload.ExpiresAt {
		metrics.RecordJWEDecryptionError("expired")
		c.JSON(http.StatusGone, gin.H{"error": "payment invitation has expired"})
		return
	}

	// 3. Resolve via Usecase (Internal lookup)
	requestID, err := uuid.Parse(payload.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id in code"})
		return
	}

	session, txData, err := h.usecase.GetPaymentRequest(c.Request.Context(), requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session no longer exists"})
		return
	}

	// 4. Compliance Check (Payer Wallet)
	payerWallet := c.Query("wallet")
	riskScore := 0
	riskLevel := "UNKNOWN"
	if payerWallet != "" {
		score, level, complianceErr := h.complianceService.ValidatePayer(c.Request.Context(), payerWallet)
		riskScore = score
		riskLevel = level
		if complianceErr != nil || score > 80 { // Threshold defined in Phase 8.2
			h.logAudit(c, session.ID, payerWallet, riskScore, riskLevel, "BLOCKED", "High risk or compliance error")
			c.JSON(http.StatusForbidden, gin.H{"error": "compliance check failed: risk too high"})
			return
		}
	}

	// 5. Log Audit (Success)
	h.logAudit(c, session.ID, payerWallet, riskScore, riskLevel, "SUCCESS", "")

	// 6. Response (Instruction focused)
	response.Success(c, http.StatusOK, gin.H{
		"session_id":     session.ID,
		"merchant_id":    session.MerchantID,
		"amount":         session.Amount,
		"currency":       session.NetworkID,
		"instruction":    txData,
		"expires_at":     session.ExpiresAt,
	})
}

func (h *PaymentResolveHandler) logAudit(c *gin.Context, sessionID uuid.UUID, wallet string, score int, level string, status string, reason string) {
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
