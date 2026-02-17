package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v8"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestMerchantRepository_CreateGetUpdateStatusDelete(t *testing.T) {
	db := newTestDB(t)
	createMerchantTable(t, db)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	m := &entities.Merchant{
		ID:                 uuid.New(),
		UserID:             uuid.New(),
		BusinessName:       "Acme",
		BusinessEmail:      "acme@example.com",
		MerchantType:       entities.MerchantTypeRetail,
		Status:             entities.MerchantStatusPending,
		TaxID:              null.StringFrom("NPWP-1"),
		BusinessAddress:    null.StringFrom("Jakarta"),
		Documents:          null.JSONFrom([]byte(`{"doc":"ok"}`)),
		FeeDiscountPercent: "0",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	require.NoError(t, repo.Create(ctx, m))

	byID, err := repo.GetByID(ctx, m.ID)
	require.NoError(t, err)
	require.Equal(t, m.ID, byID.ID)

	byUser, err := repo.GetByUserID(ctx, m.UserID)
	require.NoError(t, err)
	require.Equal(t, m.UserID, byUser.UserID)

	m.BusinessName = "Acme Updated"
	require.NoError(t, repo.Update(ctx, m))

	require.NoError(t, repo.UpdateStatus(ctx, m.ID, entities.MerchantStatusActive))
	require.NoError(t, repo.UpdateStatus(ctx, m.ID, entities.MerchantStatusRejected))

	items, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, items, 1)

	require.NoError(t, repo.SoftDelete(ctx, m.ID))
	_, err = repo.GetByID(ctx, m.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMerchantRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createMerchantTable(t, db)
	repo := NewMerchantRepository(db)
	ctx := context.Background()
	id := uuid.New()

	_, err := repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByUserID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Update(ctx, &entities.Merchant{ID: id, BusinessName: "x", BusinessEmail: "x@x", MerchantType: entities.MerchantTypeRetail, Status: entities.MerchantStatusPending})
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.UpdateStatus(ctx, id, entities.MerchantStatusActive)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.SoftDelete(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMerchantRepository_DBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// intentionally skip table creation
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.Error(t, err)
	_, err = repo.GetByUserID(ctx, uuid.New())
	require.Error(t, err)
	_, err = repo.List(ctx)
	require.Error(t, err)
	err = repo.Create(ctx, &entities.Merchant{ID: uuid.New(), UserID: uuid.New(), BusinessName: "x", BusinessEmail: "x@x", MerchantType: entities.MerchantTypeRetail, Status: entities.MerchantStatusPending})
	require.Error(t, err)
	err = repo.Update(ctx, &entities.Merchant{ID: uuid.New(), BusinessName: "x", BusinessEmail: "x@x", MerchantType: entities.MerchantTypeRetail, Status: entities.MerchantStatusPending})
	require.Error(t, err)
	err = repo.UpdateStatus(ctx, uuid.New(), entities.MerchantStatusActive)
	require.Error(t, err)
	err = repo.SoftDelete(ctx, uuid.New())
	require.Error(t, err)
}
