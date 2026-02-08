package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

// UserRepository defines user data operations
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, search string) ([]*entities.User, error)
}

// EmailVerificationRepository defines email verification operations
type EmailVerificationRepository interface {
	Create(ctx context.Context, userID uuid.UUID, token string) error
	GetByToken(ctx context.Context, token string) (*entities.User, error)
	MarkVerified(ctx context.Context, token string) error
}
