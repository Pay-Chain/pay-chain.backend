package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

// FeeConfigRepository defines fee config lookup operations.
type FeeConfigRepository interface {
	GetByChainAndToken(ctx context.Context, chainID, tokenID uuid.UUID) (*entities.FeeConfig, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.FeeConfig, error)
	List(ctx context.Context, chainID, tokenID *uuid.UUID, pagination utils.PaginationParams) ([]*entities.FeeConfig, int64, error)
	Create(ctx context.Context, config *entities.FeeConfig) error
	Update(ctx context.Context, config *entities.FeeConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
}
