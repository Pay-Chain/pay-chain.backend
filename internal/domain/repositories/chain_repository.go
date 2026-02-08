package repositories

import (
	"context"

	"pay-chain.backend/internal/domain/entities"
)

// ChainRepository defines chain data operations
type ChainRepository interface {
	GetByID(ctx context.Context, id int) (*entities.Chain, error)
	GetByCAIP2(ctx context.Context, caip2 string) (*entities.Chain, error)
	GetAll(ctx context.Context) ([]*entities.Chain, error)
	GetActive(ctx context.Context) ([]*entities.Chain, error)
	Create(ctx context.Context, chain *entities.Chain) error
	Update(ctx context.Context, chain *entities.Chain) error
	Delete(ctx context.Context, id int) error
}
