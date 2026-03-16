package repositories

import (
	"context"

	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
)

type WebhookLogRepository interface {
	Create(ctx context.Context, log *entities.WebhookDelivery) error
	Update(ctx context.Context, log *entities.WebhookDelivery) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.WebhookDelivery, error)
	GetPendingAttempts(ctx context.Context, limit int) ([]entities.WebhookDelivery, error)
	GetMerchantHistory(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]entities.WebhookDelivery, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, httpCode int, body string) error
}
