package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestUserRepository_CRUDAndList(t *testing.T) {
	db := newTestDB(t)
	createUserTable(t, db)
	repo := NewUserRepository(db)
	ctx := context.Background()

	now := time.Now()
	u := &entities.User{
		ID:           uuid.New(),
		Email:        "a@paychain.io",
		Name:         "Alice",
		PasswordHash: "hash",
		Role:         entities.UserRoleAdmin,
		KYCStatus:    entities.KYCFullyVerified,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, repo.Create(ctx, u))

	byID, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, u.Email, byID.Email)

	byEmail, err := repo.GetByEmail(ctx, u.Email)
	require.NoError(t, err)
	require.Equal(t, u.ID, byEmail.ID)

	u.Name = "Alice Updated"
	require.NoError(t, repo.Update(ctx, u))

	require.NoError(t, repo.UpdatePassword(ctx, u.ID, "hash2"))

	items, err := repo.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, items, 1)

	require.NoError(t, repo.SoftDelete(ctx, u.ID))
	_, err = repo.GetByID(ctx, u.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestUserRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createUserTable(t, db)
	repo := NewUserRepository(db)
	ctx := context.Background()
	id := uuid.New()

	_, err := repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByEmail(ctx, "missing@paychain.io")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Update(ctx, &entities.User{ID: id, Name: "x", Role: entities.UserRoleUser, KYCStatus: entities.KYCNotStarted})
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.UpdatePassword(ctx, id, "hash")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.SoftDelete(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}
