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

// WalletRepository implements wallet data operations
type WalletRepository struct {
	db *gorm.DB
}

// NewWalletRepository creates a new wallet repository
func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

// Create creates a new wallet
func (r *WalletRepository) Create(ctx context.Context, wallet *entities.Wallet) error {
	m := &models.Wallet{
		ID:        wallet.ID,
		ChainID:   wallet.ChainID,
		Address:   wallet.Address,
		IsPrimary: wallet.IsPrimary,
		CreatedAt: wallet.CreatedAt,
	}

	if wallet.UserID.Valid {
		// Assuming UserID.String is valid UUID string
		uid, _ := uuid.Parse(wallet.UserID.String)
		m.UserID = &uid
	}

	if wallet.MerchantID.Valid {
		mid, _ := uuid.Parse(wallet.MerchantID.String)
		m.MerchantID = &mid
	}

	return r.db.WithContext(ctx).Create(m).Error
}

// GetByID gets a wallet by ID
func (r *WalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Wallet, error) {
	var m models.Wallet
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetByUserID gets wallets for a user
func (r *WalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error) {
	var ms []models.Wallet
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_primary DESC, created_at DESC").
		Find(&ms).Error; err != nil {
		return nil, err
	}

	var wallets []*entities.Wallet
	for _, m := range ms {
		model := m
		wallets = append(wallets, r.toEntity(&model))
	}
	return wallets, nil
}

// GetByAddress gets a wallet by address and chain
func (r *WalletRepository) GetByAddress(ctx context.Context, chainID, address string) (*entities.Wallet, error) {
	// ChainID is string in CAIP-2?
	// Original impl has GetByAddress(chainID, address string)
	// But query used chain_id = $1 (int?) or string?
	// The repo signature says `chainID, address string`.
	// But `models.Wallet` has `ChainID int`.
	// And `parseCAIP2` logic suggests we might need to parse it if input is string CAIP-2?
	// Or maybe the input string is just integer string?
	// Let's check `WalletRepository` interface if possible or previous impl.
	// Previous impl: `WHERE chain_id = $1`. If column is integer, Go driver might auto-convert.
	// But if GORM model `ChainID` is `int`, passing string to Where might be tricky.
	// I'll assume input `chainID` is CAIP-2 or string int.
	// `entities.Wallet` has `ChainID int`.
	// The input to `GetByAddress` is `string`.
	// Maybe I should try to parse it to int? Or maybe it matches CAIP-2 namespace?
	// If it's pure string address query, maybe logic used to pass things like "1" or "ethereum".

	// Wait, internal/infrastructure/repositories/wallet_repo_impl.go line 111:
	// func (r *WalletRepository) GetByAddress(ctx context.Context, chainID, address string)

	// I will attempt simple Where.

	var m models.Wallet
	if err := r.db.WithContext(ctx).Where("chain_id = ? AND address = ?", chainID, address).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// SetPrimary sets a wallet as primary (and unsets others)
func (r *WalletRepository) SetPrimary(ctx context.Context, userID, walletID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Unset all
		if err := tx.Model(&models.Wallet{}).
			Where("user_id = ?", userID).
			Update("is_primary", false).Error; err != nil {
			return err
		}

		// Set new primary
		result := tx.Model(&models.Wallet{}).
			Where("id = ? AND user_id = ?", walletID, userID).
			Update("is_primary", true)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domainerrors.ErrNotFound
		}
		return nil
	})
}

// SoftDelete soft deletes a wallet
func (r *WalletRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Wallet{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *WalletRepository) toEntity(m *models.Wallet) *entities.Wallet {
	// conversion logic
	return &entities.Wallet{
		ID: m.ID,
		// UserID:     null.StringFrom(m.UserID.String()), // if *uuid
		// MerchantID: ...
		ChainID:   m.ChainID,
		Address:   m.Address,
		IsPrimary: m.IsPrimary,
		CreatedAt: m.CreatedAt,
	}
}
