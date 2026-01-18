package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

// WalletRepository implements wallet data operations
type WalletRepository struct {
	db *sql.DB
}

// NewWalletRepository creates a new wallet repository
func NewWalletRepository(db *sql.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

// Create creates a new wallet
func (r *WalletRepository) Create(ctx context.Context, wallet *entities.Wallet) error {
	query := `
		INSERT INTO wallets (id, user_id, merchant_id, chain_id, address, is_primary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	wallet.ID = uuid.New()
	wallet.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		wallet.ID,
		wallet.UserID,
		wallet.MerchantID,
		wallet.ChainID,
		wallet.Address,
		wallet.IsPrimary,
		wallet.CreatedAt,
	)

	return err
}

// GetByID gets a wallet by ID
func (r *WalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Wallet, error) {
	query := `
		SELECT id, user_id, merchant_id, chain_id, address, is_primary, created_at
		FROM wallets
		WHERE id = $1 AND deleted_at IS NULL
	`

	wallet := &entities.Wallet{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.MerchantID,
		&wallet.ChainID,
		&wallet.Address,
		&wallet.IsPrimary,
		&wallet.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

// GetByUserID gets wallets for a user
func (r *WalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error) {
	query := `
		SELECT id, user_id, merchant_id, chain_id, address, is_primary, created_at
		FROM wallets
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY is_primary DESC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []*entities.Wallet
	for rows.Next() {
		wallet := &entities.Wallet{}
		if err := rows.Scan(
			&wallet.ID,
			&wallet.UserID,
			&wallet.MerchantID,
			&wallet.ChainID,
			&wallet.Address,
			&wallet.IsPrimary,
			&wallet.CreatedAt,
		); err != nil {
			return nil, err
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

// GetByAddress gets a wallet by address and chain
func (r *WalletRepository) GetByAddress(ctx context.Context, chainID, address string) (*entities.Wallet, error) {
	query := `
		SELECT id, user_id, merchant_id, chain_id, address, is_primary, created_at
		FROM wallets
		WHERE chain_id = $1 AND address = $2 AND deleted_at IS NULL
	`

	wallet := &entities.Wallet{}
	err := r.db.QueryRowContext(ctx, query, chainID, address).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.MerchantID,
		&wallet.ChainID,
		&wallet.Address,
		&wallet.IsPrimary,
		&wallet.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

// SetPrimary sets a wallet as primary (and unsets others)
func (r *WalletRepository) SetPrimary(ctx context.Context, userID, walletID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Unset all primary for user
	_, err = tx.ExecContext(ctx, 
		`UPDATE wallets SET is_primary = false WHERE user_id = $1 AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		return err
	}

	// Set new primary
	result, err := tx.ExecContext(ctx,
		`UPDATE wallets SET is_primary = true WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		walletID, userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return tx.Commit()
}

// SoftDelete soft deletes a wallet
func (r *WalletRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE wallets SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}
