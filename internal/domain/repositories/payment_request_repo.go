package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

// PaymentRequestRepository interface
type PaymentRequestRepository interface {
	Create(ctx context.Context, request *entities.PaymentRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.PaymentRequest, error)
	GetByMerchantID(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PaymentRequestStatus) error
	MarkCompleted(ctx context.Context, id uuid.UUID, txHash string) error
	GetExpiredPending(ctx context.Context, limit int) ([]*entities.PaymentRequest, error)
	ExpireRequests(ctx context.Context, ids []uuid.UUID) error
}

// Note: MarkCompleted/Expired methods were inferred from usage (webhook usecase and expiry job)
