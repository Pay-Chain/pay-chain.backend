package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

// BridgeConfigRepository defines bridge routing config operations.
type BridgeConfigRepository interface {
	GetActive(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.BridgeConfig, error)
}
