package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
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
	// Entity fields: Metadata interface{}
	// Model fields: Metadata string (jsonb)

	// Convert metadata to string/json
	// Assuming simple conversion or empty for now or use json.Marshal
	meta := "{}"
	// if event.Metadata != nil ... json.Marshal ...

	m := &models.PaymentEvent{
		ID:          event.ID,
		PaymentID:   event.PaymentID,
		EventType:   event.EventType,
		Chain:       event.Chain,
		TxHash:      event.TxHash,
		BlockNumber: event.BlockNumber,
		Metadata:    meta,
		CreatedAt:   event.CreatedAt,
	}

	return r.db.WithContext(ctx).Create(m).Error
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
		// Convert model to entity
		// Unmarshal metadata
		event := &entities.PaymentEvent{
			ID:          m.ID,
			PaymentID:   m.PaymentID,
			EventType:   m.EventType,
			Chain:       m.Chain,
			TxHash:      m.TxHash,
			BlockNumber: m.BlockNumber,
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
		EventType:   m.EventType,
		Chain:       m.Chain,
		TxHash:      m.TxHash,
		BlockNumber: m.BlockNumber,
		CreatedAt:   m.CreatedAt,
	}, nil
}
