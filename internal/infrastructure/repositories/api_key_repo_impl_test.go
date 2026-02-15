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

func TestApiKeyRepository_CRUDAndFinders(t *testing.T) {
	db := newTestDB(t)
	createUserTable(t, db)
	createAPIKeyTable(t, db)
	repo := NewApiKeyRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()
	mustExec(t, db, `INSERT INTO users(id,email,name,role,kyc_status,password_hash,is_email_verified,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		userID.String(), "admin@paychain.io", "Admin", "ADMIN", "APPROVED", "x", true, now, now)

	ak := &entities.ApiKey{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            "default",
		KeyPrefix:       "pk_live",
		KeyHash:         "hash_1",
		SecretEncrypted: "enc",
		SecretMasked:    "****1234",
		Permissions:     []string{"read", "write"},
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, repo.Create(ctx, ak))

	byHash, err := repo.FindByKeyHash(ctx, "hash_1")
	require.NoError(t, err)
	require.Equal(t, ak.ID, byHash.ID)
	require.Equal(t, 2, len(byHash.Permissions))

	byUser, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, byUser, 1)

	byID, err := repo.FindByID(ctx, ak.ID)
	require.NoError(t, err)
	require.Equal(t, "default", byID.Name)

	ak.Name = "updated"
	ak.IsActive = false
	require.NoError(t, repo.Update(ctx, ak))

	require.NoError(t, repo.Delete(ctx, ak.ID))
	_, err = repo.FindByID(ctx, ak.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestApiKeyRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createUserTable(t, db)
	createAPIKeyTable(t, db)
	repo := NewApiKeyRepository(db)
	ctx := context.Background()
	id := uuid.New()

	_, err := repo.FindByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.FindByKeyHash(ctx, "missing")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Update(ctx, &entities.ApiKey{ID: id, Name: "x", Permissions: []string{}, IsActive: true})
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Delete(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}
