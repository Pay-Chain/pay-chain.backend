package repositories

import (
	"context"

	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
)

type MerchantSettlementProfileRepository interface {
	GetByMerchantID(ctx context.Context, merchantID uuid.UUID) (*entities.MerchantSettlementProfile, error)
	ListMissingMerchantIDs(ctx context.Context) ([]uuid.UUID, error)
	HasProfilesByMerchantIDs(ctx context.Context, merchantIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	Upsert(ctx context.Context, profile *entities.MerchantSettlementProfile) error
}
