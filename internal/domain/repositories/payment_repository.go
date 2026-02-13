package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

// PaymentRepository defines payment data operations
type PaymentRepository interface {
	Create(ctx context.Context, payment *entities.Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Payment, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Payment, int, error)
	GetByMerchantID(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*entities.Payment, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PaymentStatus) error
	UpdateDestTxHash(ctx context.Context, id uuid.UUID, txHash string) error
	MarkRefunded(ctx context.Context, id uuid.UUID) error
}
