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
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
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

	// Use transaction-aware DB instance
	db := GetDB(ctx, r.db).WithContext(ctx)
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

	const maxFKRetries = 3
	var lastErr error
	for attempt := 0; attempt < maxFKRetries; attempt++ {
		lastErr = db.Create(m).Error
		if lastErr == nil {
			return nil
		}
		if !isForeignKeyError(lastErr) {
			return lastErr
		}
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
