package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

type ApiKeyRepository interface {
	Create(ctx context.Context, apiKey *entities.ApiKey) error
	FindByKeyHash(ctx context.Context, keyHash string) (*entities.ApiKey, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.ApiKey, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entities.ApiKey, error)
	Update(ctx context.Context, apiKey *entities.ApiKey) error
	Delete(ctx context.Context, id uuid.UUID) error
}
