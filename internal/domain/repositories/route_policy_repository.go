package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

type RoutePolicyRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.RoutePolicy, error)
	GetByRoute(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.RoutePolicy, error)
	List(ctx context.Context, sourceChainID, destChainID *uuid.UUID, pagination utils.PaginationParams) ([]*entities.RoutePolicy, int64, error)
	Create(ctx context.Context, policy *entities.RoutePolicy) error
	Update(ctx context.Context, policy *entities.RoutePolicy) error
	Delete(ctx context.Context, id uuid.UUID) error
}
