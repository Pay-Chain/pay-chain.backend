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
}

// WalletRepository defines wallet data operations
type WalletRepository interface {
	Create(ctx context.Context, wallet *entities.Wallet) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Wallet, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error)
	GetByAddress(ctx context.Context, chainID, address string) (*entities.Wallet, error)
	SetPrimary(ctx context.Context, userID, walletID uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// TokenRepository defines token data operations
type TokenRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Token, error)
	GetBySymbol(ctx context.Context, symbol string) (*entities.Token, error)
	GetAll(ctx context.Context) ([]*entities.Token, error)
	GetStablecoins(ctx context.Context) ([]*entities.Token, error)
	GetSupportedByChain(ctx context.Context, chainID int) ([]*entities.SupportedToken, error)
}
