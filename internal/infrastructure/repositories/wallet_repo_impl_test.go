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

func TestWalletRepository_CRUDAndPrimary(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createWalletTable(t, db)
	repo := NewWalletRepository(db)
	ctx := context.Background()

	chainID := uuid.New()
	seedChain(t, db, chainID.String(), "8453", "Base", "EVM", true)
	userID := uuid.New()

	w1 := &entities.Wallet{ID: uuid.New(), UserID: &userID, ChainID: chainID, Address: "0xabc", IsPrimary: true, CreatedAt: time.Now()}
	w2 := &entities.Wallet{ID: uuid.New(), UserID: &userID, ChainID: chainID, Address: "0xdef", IsPrimary: false, CreatedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, w1))
	require.NoError(t, repo.Create(ctx, w2))

	got, err := repo.GetByID(ctx, w1.ID)
	require.NoError(t, err)
	require.Equal(t, w1.ID, got.ID)

	list, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	byAddr, err := repo.GetByAddress(ctx, chainID, "0xabc")
	require.NoError(t, err)
	require.Equal(t, w1.ID, byAddr.ID)

	require.NoError(t, repo.SetPrimary(ctx, userID, w2.ID))

	require.NoError(t, repo.SoftDelete(ctx, w1.ID))
	_, err = repo.GetByID(ctx, w1.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestWalletRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createWalletTable(t, db)
	repo := NewWalletRepository(db)
	ctx := context.Background()

	id := uuid.New()
	_, err := repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByAddress(ctx, uuid.New(), "0xmissing")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.SetPrimary(ctx, uuid.New(), uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.SoftDelete(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}
