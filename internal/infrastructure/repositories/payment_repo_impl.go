package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

// PaymentRepository implements payment data operations
type PaymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository creates a new payment repository
func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create creates a new payment
func (r *PaymentRepository) Create(ctx context.Context, payment *entities.Payment) error {
	query := `
		INSERT INTO payments (
			id, sender_id, merchant_id, receiver_wallet_id,
			source_chain_id, dest_chain_id, source_token_address, dest_token_address,
			source_amount, dest_amount, fee_amount, status,
			bridge_type, source_tx_hash, receiver_address, decimals,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	payment.ID = uuid.New()
	payment.Status = entities.PaymentStatusPending
	payment.CreatedAt = time.Now()
	payment.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		payment.ID,
		payment.SenderID,
		payment.MerchantID,
		payment.ReceiverWalletID,
		payment.SourceChainID,
		payment.DestChainID,
		payment.SourceTokenAddress,
		payment.DestTokenAddress,
		payment.SourceAmount,
		payment.DestAmount,
		payment.FeeAmount,
		payment.Status,
		payment.BridgeType,
		payment.SourceTxHash,
		payment.ReceiverAddress,
		payment.Decimals,
		payment.CreatedAt,
		payment.UpdatedAt,
	)

	return err
}

// GetByID gets a payment by ID
func (r *PaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Payment, error) {
	query := `
		SELECT id, sender_id, merchant_id, receiver_wallet_id,
		       source_chain_id, dest_chain_id, source_token_address, dest_token_address,
		       source_amount, dest_amount, fee_amount, status,
		       bridge_type, source_tx_hash, dest_tx_hash, receiver_address, decimals,
		       refunded_at, created_at, updated_at
		FROM payments
		WHERE id = $1 AND deleted_at IS NULL
	`

	payment := &entities.Payment{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&payment.ID,
		&payment.SenderID,
		&payment.MerchantID,
		&payment.ReceiverWalletID,
		&payment.SourceChainID,
		&payment.DestChainID,
		&payment.SourceTokenAddress,
		&payment.DestTokenAddress,
		&payment.SourceAmount,
		&payment.DestAmount,
		&payment.FeeAmount,
		&payment.Status,
		&payment.BridgeType,
		&payment.SourceTxHash,
		&payment.DestTxHash,
		&payment.ReceiverAddress,
		&payment.Decimals,
		&payment.RefundedAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// GetByUserID gets payments for a user with pagination
func (r *PaymentRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Payment, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM payments WHERE sender_id = $1 AND deleted_at IS NULL`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get payments
	query := `
		SELECT id, sender_id, merchant_id, receiver_wallet_id,
		       source_chain_id, dest_chain_id, source_token_address, dest_token_address,
		       source_amount, dest_amount, fee_amount, status,
		       bridge_type, source_tx_hash, dest_tx_hash, receiver_address, decimals,
		       refunded_at, created_at, updated_at
		FROM payments
		WHERE sender_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []*entities.Payment
	for rows.Next() {
		payment := &entities.Payment{}
		if err := rows.Scan(
			&payment.ID,
			&payment.SenderID,
			&payment.MerchantID,
			&payment.ReceiverWalletID,
			&payment.SourceChainID,
			&payment.DestChainID,
			&payment.SourceTokenAddress,
			&payment.DestTokenAddress,
			&payment.SourceAmount,
			&payment.DestAmount,
			&payment.FeeAmount,
			&payment.Status,
			&payment.BridgeType,
			&payment.SourceTxHash,
			&payment.DestTxHash,
			&payment.ReceiverAddress,
			&payment.Decimals,
			&payment.RefundedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		payments = append(payments, payment)
	}

	return payments, total, nil
}

// GetByMerchantID gets payments for a merchant
func (r *PaymentRepository) GetByMerchantID(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*entities.Payment, int, error) {
	countQuery := `SELECT COUNT(*) FROM payments WHERE merchant_id = $1 AND deleted_at IS NULL`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, merchantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, sender_id, merchant_id, receiver_wallet_id,
		       source_chain_id, dest_chain_id, source_token_address, dest_token_address,
		       source_amount, dest_amount, fee_amount, status,
		       bridge_type, source_tx_hash, dest_tx_hash, receiver_address, decimals,
		       refunded_at, created_at, updated_at
		FROM payments
		WHERE merchant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, merchantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []*entities.Payment
	for rows.Next() {
		payment := &entities.Payment{}
		if err := rows.Scan(
			&payment.ID,
			&payment.SenderID,
			&payment.MerchantID,
			&payment.ReceiverWalletID,
			&payment.SourceChainID,
			&payment.DestChainID,
			&payment.SourceTokenAddress,
			&payment.DestTokenAddress,
			&payment.SourceAmount,
			&payment.DestAmount,
			&payment.FeeAmount,
			&payment.Status,
			&payment.BridgeType,
			&payment.SourceTxHash,
			&payment.DestTxHash,
			&payment.ReceiverAddress,
			&payment.Decimals,
			&payment.RefundedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		payments = append(payments, payment)
	}

	return payments, total, nil
}

// UpdateStatus updates payment status
func (r *PaymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PaymentStatus) error {
	query := `
		UPDATE payments
		SET status = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, status, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}

// UpdateDestTxHash updates destination transaction hash
func (r *PaymentRepository) UpdateDestTxHash(ctx context.Context, id uuid.UUID, txHash string) error {
	query := `
		UPDATE payments
		SET dest_tx_hash = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, id, txHash, time.Now())
	return err
}

// MarkRefunded marks a payment as refunded
func (r *PaymentRepository) MarkRefunded(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE payments
		SET status = $2, refunded_at = $3, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, id, entities.PaymentStatusRefunded, now)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}
