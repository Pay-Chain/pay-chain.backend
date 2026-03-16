package repositories

import (
	"context"

	"payment-kita.backend/internal/domain/entities"
)

type AuditRepository interface {
	LogResolveAttempt(ctx context.Context, audit *entities.ResolveAudit) error
}
