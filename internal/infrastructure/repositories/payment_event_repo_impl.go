package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/infrastructure/models"
)

// PaymentEventRepository implements payment event data operations
type PaymentEventRepository struct {
	db *gorm.DB
}

// NewPaymentEventRepository creates a new payment event repository
func NewPaymentEventRepository(db *gorm.DB) *PaymentEventRepository {
	return &PaymentEventRepository{db: db}
}

// Create creates a new payment event
func (r *PaymentEventRepository) Create(ctx context.Context, event *entities.PaymentEvent) error {
	meta := "{}"
	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	// Primary path: transaction-aware DB instance (if any)
	primaryDB := GetDB(ctx, r.db).WithContext(ctx)
	// Fallback path: force base DB (non-transaction) to avoid stale/aborted tx context.
	// This is important for best-effort event writes after parent payment commit.
	fallbackDB := r.db.WithContext(ctx)
	_, inTx := ctx.Value(txKey).(*gorm.DB)
	// Map Entity -> Model
	m := &models.PaymentEvent{
		ID:          event.ID,
		PaymentID:   event.PaymentID,
		EventType:   string(event.EventType),
		TxHash:      event.TxHash,
		Metadata:    meta,
		CreatedAt:   createdAt,
		ChainID:     event.ChainID,
		Chain:       r.resolveLegacyChainValue(event.ChainID),
		BlockNumber: event.BlockNumber,
	}

	// Keep this best-effort write quiet: a single attempt avoids repeated FK spam logs
	// from Postgres/GORM while preserving warning visibility at caller level.
	const maxRetries = 1
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// First attempt uses transaction-aware DB, next attempts force base DB.
		db := primaryDB
		if attempt > 0 && !inTx {
			db = fallbackDB
		}

		lastErr = db.Create(m).Error
		if lastErr == nil {
			return nil
		}

		// In an active transaction, FK/aborted errors poison tx state.
		// Return immediately and let caller decide best-effort handling.
		if inTx {
			return lastErr
		}

		// Non-retryable errors fail fast.
		if !isForeignKeyError(lastErr) && !isTransactionAbortedError(lastErr) {
			return lastErr
		}

		// Backoff before retrying on base DB / eventual visibility.
		time.Sleep(time.Duration(100*(attempt+1)) * time.Millisecond)
	}
	return fmt.Errorf("failed to create payment event after FK retries: %w", lastErr)
}

func isForeignKeyError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return string(pgErr.Code) == "23503"
	}
	return strings.Contains(strings.ToLower(err.Error()), "foreign key")
}

func isTransactionAbortedError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return string(pgErr.Code) == "25P02"
	}
	return strings.Contains(strings.ToLower(err.Error()), "current transaction is aborted")
}

func (r *PaymentEventRepository) resolveLegacyChainValue(chainID *uuid.UUID) string {
	if chainID == nil {
		return "UNKNOWN"
	}
	// Legacy table expects short value, so keep only first 8 chars from UUID.
	return fmt.Sprintf("chain-%s", chainID.String()[:8])
}

// GetByPaymentID gets events for a payment
func (r *PaymentEventRepository) GetByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]*entities.PaymentEvent, error) {
	var ms []models.PaymentEvent
	if err := r.db.WithContext(ctx).
		Where("payment_id = ?", paymentID).
		Order("created_at ASC").
		Find(&ms).Error; err != nil {
		return nil, err
	}

	var events []*entities.PaymentEvent
	for _, m := range ms {
		event := &entities.PaymentEvent{
			ID:          m.ID,
			PaymentID:   m.PaymentID,
			EventType:   entities.PaymentEventType(m.EventType),
			TxHash:      m.TxHash,
			ChainID:     nil,
			BlockNumber: 0,
			// Metadata:    ...,
			CreatedAt: m.CreatedAt,
		}
		events = append(events, event)
	}

	return events, nil
}

// GetLatestByPaymentID gets the latest event for a payment
func (r *PaymentEventRepository) GetLatestByPaymentID(ctx context.Context, paymentID uuid.UUID) (*entities.PaymentEvent, error) {
	var m models.PaymentEvent
	if err := r.db.WithContext(ctx).
		Where("payment_id = ?", paymentID).
		Order("created_at DESC").
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}

	return &entities.PaymentEvent{
		ID:          m.ID,
		PaymentID:   m.PaymentID,
		EventType:   entities.PaymentEventType(m.EventType),
		TxHash:      m.TxHash,
		ChainID:     nil,
		BlockNumber: 0,
		CreatedAt:   m.CreatedAt,
	}, nil
}
