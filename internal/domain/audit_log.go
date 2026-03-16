package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID `json:"id"`
	MerchantID uuid.UUID `json:"merchant_id"`
	Path       string    `json:"path"`
	Method     string    `json:"method"`
	StatusCode int       `json:"status_code"`
	IPAddress  string    `json:"ip_address"`
	Duration   float64   `json:"duration"`
	CreatedAt  time.Time `json:"created_at"`
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
}
