package repositories

import (
	"context"

	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
)

type PaymentQuoteRepository interface {
	Create(ctx context.Context, quote *entities.PaymentQuote) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.PaymentQuote, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PaymentQuoteStatus) error
	MarkUsed(ctx context.Context, id uuid.UUID) error
	GetExpiredActive(ctx context.Context, limit int) ([]*entities.PaymentQuote, error)
	ExpireQuotes(ctx context.Context, ids []uuid.UUID) error
}

type PartnerPaymentSessionRepository interface {
	Create(ctx context.Context, session *entities.PartnerPaymentSession) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.PartnerPaymentSession, error)
	GetByPaymentRequestID(ctx context.Context, paymentRequestID uuid.UUID) (*entities.PartnerPaymentSession, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PartnerPaymentSessionStatus) error
	MarkCompleted(ctx context.Context, id uuid.UUID, paidTxHash string) error
	GetExpiredPending(ctx context.Context, limit int) ([]*entities.PartnerPaymentSession, error)
	ExpireSessions(ctx context.Context, ids []uuid.UUID) error
}
