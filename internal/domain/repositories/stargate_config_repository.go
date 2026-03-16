package repositories

import (
	"context"

	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/pkg/utils"
)

type StargateConfigRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.StargateConfig, error)
	GetByRoute(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.StargateConfig, error)
	List(ctx context.Context, sourceChainID, destChainID *uuid.UUID, activeOnly *bool, pagination utils.PaginationParams) ([]*entities.StargateConfig, int64, error)
	Create(ctx context.Context, config *entities.StargateConfig) error
	Update(ctx context.Context, config *entities.StargateConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
}
