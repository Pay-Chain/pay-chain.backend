package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

func TestPaymentBridgeRepository_CRUDAndList(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	repo := NewPaymentBridgeRepository(db)
	ctx := context.Background()

	bridge := &entities.PaymentBridge{ID: uuid.New(), Name: "CCIP"}
	require.NoError(t, repo.Create(ctx, bridge))

	gotByName, err := repo.GetByName(ctx, "ccip")
	require.NoError(t, err)
	require.Equal(t, "CCIP", gotByName.Name)

	gotByID, err := repo.GetByID(ctx, gotByName.ID)
	require.NoError(t, err)
	require.Equal(t, gotByName.ID, gotByID.ID)

	items, total, err := repo.List(ctx, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	gotByID.Name = "Hyperbridge"
	require.NoError(t, repo.Update(ctx, gotByID))

	require.NoError(t, repo.Delete(ctx, gotByID.ID))
	_, err = repo.GetByID(ctx, gotByID.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestPaymentBridgeRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	repo := NewPaymentBridgeRepository(db)
	ctx := context.Background()

	id := uuid.New()
	_, err := repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByName(ctx, "missing")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Update(ctx, &entities.PaymentBridge{ID: id, Name: "x"})
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Delete(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}
