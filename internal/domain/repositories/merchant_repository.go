package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

// MerchantRepository defines merchant data operations
type MerchantRepository interface {
	Create(ctx context.Context, merchant *entities.Merchant) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Merchant, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.Merchant, error)
	Update(ctx context.Context, merchant *entities.Merchant) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.MerchantStatus) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*entities.Merchant, error)
}

// WalletRepository defines wallet data operations
type WalletRepository interface {
	Create(ctx context.Context, wallet *entities.Wallet) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Wallet, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error)
	GetByAddress(ctx context.Context, chainID uuid.UUID, address string) (*entities.Wallet, error)
	SetPrimary(ctx context.Context, userID, walletID uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
