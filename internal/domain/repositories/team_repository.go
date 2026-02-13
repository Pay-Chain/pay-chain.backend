package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

type TeamRepository interface {
	Create(ctx context.Context, team *entities.Team) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Team, error)
	ListPublic(ctx context.Context) ([]*entities.Team, error)
	ListAdmin(ctx context.Context, search string) ([]*entities.Team, error)
	Update(ctx context.Context, team *entities.Team) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
