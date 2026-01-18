package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

// PaymentEventRepository implements payment event data operations
type PaymentEventRepository struct {
	db *sql.DB
}

// NewPaymentEventRepository creates a new payment event repository
func NewPaymentEventRepository(db *sql.DB) *PaymentEventRepository {
	return &PaymentEventRepository{db: db}
}

// Create creates a new payment event
func (r *PaymentEventRepository) Create(ctx context.Context, event *entities.PaymentEvent) error {
	query := `
		INSERT INTO payment_events (
			id, payment_id, event_type, chain, tx_hash, block_number, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	event.ID = uuid.New()
	event.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		event.ID,
		event.PaymentID,
		event.EventType,
		event.Chain,
		event.TxHash,
		event.BlockNumber,
		event.Metadata,
		event.CreatedAt,
	)

	return err
}

// GetByPaymentID gets events for a payment
func (r *PaymentEventRepository) GetByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]*entities.PaymentEvent, error) {
	query := `
		SELECT id, payment_id, event_type, chain, tx_hash, block_number, metadata, created_at
		FROM payment_events
		WHERE payment_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*entities.PaymentEvent
	for rows.Next() {
		event := &entities.PaymentEvent{}
		if err := rows.Scan(
			&event.ID,
			&event.PaymentID,
			&event.EventType,
			&event.Chain,
			&event.TxHash,
			&event.BlockNumber,
			&event.Metadata,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

// GetLatestByPaymentID gets the latest event for a payment
func (r *PaymentEventRepository) GetLatestByPaymentID(ctx context.Context, paymentID uuid.UUID) (*entities.PaymentEvent, error) {
	query := `
		SELECT id, payment_id, event_type, chain, tx_hash, block_number, metadata, created_at
		FROM payment_events
		WHERE payment_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	event := &entities.PaymentEvent{}
	err := r.db.QueryRowContext(ctx, query, paymentID).Scan(
		&event.ID,
		&event.PaymentID,
		&event.EventType,
		&event.Chain,
		&event.TxHash,
		&event.BlockNumber,
		&event.Metadata,
		&event.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return event, nil
}
