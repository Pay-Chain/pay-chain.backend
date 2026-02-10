package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

// TokenRepository defines token data operations
type TokenRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Token, error)
	GetBySymbol(ctx context.Context, symbol string, chainID uuid.UUID) (*entities.Token, error)
	GetByAddress(ctx context.Context, address string, chainID uuid.UUID) (*entities.Token, error)
	GetAll(ctx context.Context) ([]*entities.Token, error)
	GetStablecoins(ctx context.Context) ([]*entities.Token, error)
	GetNative(ctx context.Context, chainID uuid.UUID) (*entities.Token, error)
	// GetTokensByChain replaces GetSupportedByChain
	GetTokensByChain(ctx context.Context, chainID uuid.UUID, pagination utils.PaginationParams) ([]*entities.Token, int64, error)
	// GetAllTokens replaces GetAllSupported
	GetAllTokens(ctx context.Context, chainID *uuid.UUID, search *string, pagination utils.PaginationParams) ([]*entities.Token, int64, error)
	Create(ctx context.Context, token *entities.Token) error
	Update(ctx context.Context, token *entities.Token) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
