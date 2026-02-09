package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

// ChainRepository defines chain data operations
type ChainRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Chain, error)
	GetByCAIP2(ctx context.Context, caip2 string) (*entities.Chain, error)
	GetAll(ctx context.Context) ([]*entities.Chain, error)
	GetAllRPCs(ctx context.Context, chainID *uuid.UUID, isActive *bool, search *string, pagination utils.PaginationParams) ([]*entities.ChainRPC, int64, error)
	GetActive(ctx context.Context, pagination utils.PaginationParams) ([]*entities.Chain, int64, error)
	Create(ctx context.Context, chain *entities.Chain) error
	Update(ctx context.Context, chain *entities.Chain) error
	Delete(ctx context.Context, id uuid.UUID) error
}
