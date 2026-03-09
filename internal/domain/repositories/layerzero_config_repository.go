package repositories

import (
	"context"

	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/pkg/utils"
)

type LayerZeroConfigRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.LayerZeroConfig, error)
	GetByRoute(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.LayerZeroConfig, error)
	List(ctx context.Context, sourceChainID, destChainID *uuid.UUID, activeOnly *bool, pagination utils.PaginationParams) ([]*entities.LayerZeroConfig, int64, error)
	Create(ctx context.Context, config *entities.LayerZeroConfig) error
	Update(ctx context.Context, config *entities.LayerZeroConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
}
