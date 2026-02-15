package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

type PaymentBridgeRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.PaymentBridge, error)
	GetByName(ctx context.Context, name string) (*entities.PaymentBridge, error)
	List(ctx context.Context, pagination utils.PaginationParams) ([]*entities.PaymentBridge, int64, error)
	Create(ctx context.Context, bridge *entities.PaymentBridge) error
	Update(ctx context.Context, bridge *entities.PaymentBridge) error
	Delete(ctx context.Context, id uuid.UUID) error
}
