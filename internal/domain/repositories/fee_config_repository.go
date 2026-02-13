package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

// FeeConfigRepository defines fee config lookup operations.
type FeeConfigRepository interface {
	GetByChainAndToken(ctx context.Context, chainID, tokenID uuid.UUID) (*entities.FeeConfig, error)
}
