package repositories

import (
	"context"

	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/pkg/utils"
)

// BridgeConfigRepository defines bridge routing config operations.
type BridgeConfigRepository interface {
	GetActive(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.BridgeConfig, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.BridgeConfig, error)
	List(ctx context.Context, sourceChainID, destChainID, bridgeID *uuid.UUID, pagination utils.PaginationParams) ([]*entities.BridgeConfig, int64, error)
	Create(ctx context.Context, config *entities.BridgeConfig) error
	Update(ctx context.Context, config *entities.BridgeConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
}
