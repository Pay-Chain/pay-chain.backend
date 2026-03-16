package middleware

import (
	"payment-kita.backend/internal/domain"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditMiddleware logs requests to the audit_logs table
func AuditMiddleware(repo domain.AuditLogRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Skip if merchant ID is not present (not a partner request)
		merchantIDRaw, exists := c.Get("MerchantID")
		if !exists {
			return
		}

		merchantID, ok := merchantIDRaw.(uuid.UUID)
		if !ok {
			return
		}

		duration := time.Since(start).Seconds()

		log := &domain.AuditLog{
			ID:         uuid.New(),
			MerchantID: merchantID,
			Path:       c.Request.URL.Path,
			Method:     c.Request.Method,
			StatusCode: c.Writer.Status(),
			IPAddress:  c.ClientIP(),
			Duration:   duration,
			CreatedAt:  time.Now(),
		}

		// Fail silent to not block response
		_ = repo.Create(c.Request.Context(), log)
	}
}
