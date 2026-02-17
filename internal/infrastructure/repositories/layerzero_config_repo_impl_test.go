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

func TestLayerZeroConfigRepo_CRUDAndList(t *testing.T) {
	db := newTestDB(t)
	createRoutePolicyTables(t, db)
	repo := NewLayerZeroConfigRepository(db)
	ctx := context.Background()

	sourceID := uuid.New()
	destID := uuid.New()
	item := &entities.LayerZeroConfig{
		ID:            uuid.New(),
		SourceChainID: sourceID,
		DestChainID:   destID,
		DstEID:        30110,
		PeerHex:       "0xpeer",
		OptionsHex:    "",
		IsActive:      true,
	}

	require.NoError(t, repo.Create(ctx, item))
	got, err := repo.GetByID(ctx, item.ID)
	require.NoError(t, err)
	require.Equal(t, "0x", got.OptionsHex)

	gotByRoute, err := repo.GetByRoute(ctx, sourceID, destID)
	require.NoError(t, err)
	require.Equal(t, item.ID, gotByRoute.ID)

	active := true
	items, total, err := repo.List(ctx, &sourceID, &destID, &active, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	item.OptionsHex = "0x0102"
	item.PeerHex = " 0xnewpeer "
	item.IsActive = false
	require.NoError(t, repo.Update(ctx, item))

	updated, err := repo.GetByID(ctx, item.ID)
	require.NoError(t, err)
	require.Equal(t, "0x0102", updated.OptionsHex)
	require.Equal(t, "0xnewpeer", updated.PeerHex)
	require.False(t, updated.IsActive)

	require.NoError(t, repo.Delete(ctx, item.ID))
	_, err = repo.GetByID(ctx, item.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	// Create path with zero ID + whitespace options should auto-normalize.
	autoIDItem := &entities.LayerZeroConfig{
		SourceChainID: sourceID,
		DestChainID:   destID,
		DstEID:        30111,
		PeerHex:       "0xpeer2",
		OptionsHex:    "   ",
		IsActive:      true,
	}
	require.NoError(t, repo.Create(ctx, autoIDItem))
	require.NotEqual(t, uuid.Nil, autoIDItem.ID)
}

func TestLayerZeroConfigRepo_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createRoutePolicyTables(t, db)
	repo := NewLayerZeroConfigRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByRoute(ctx, uuid.New(), uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Update(ctx, &entities.LayerZeroConfig{
		ID:            uuid.New(),
		SourceChainID: uuid.New(),
		DestChainID:   uuid.New(),
		DstEID:        1,
		PeerHex:       "0xpeer",
		OptionsHex:    "0x",
		IsActive:      true,
	})
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Delete(ctx, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestLayerZeroConfigRepo_DBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// Intentionally skip table creation.
	repo := NewLayerZeroConfigRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &entities.LayerZeroConfig{
		ID:            uuid.New(),
		SourceChainID: uuid.New(),
		DestChainID:   uuid.New(),
		DstEID:        1,
		PeerHex:       "0xpeer",
		OptionsHex:    "0x",
		IsActive:      true,
	})
	require.Error(t, err)

	_, err = repo.GetByID(ctx, uuid.New())
	require.Error(t, err)

	_, err = repo.GetByRoute(ctx, uuid.New(), uuid.New())
	require.Error(t, err)

	_, _, err = repo.List(ctx, nil, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)

	err = repo.Update(ctx, &entities.LayerZeroConfig{
		ID:            uuid.New(),
		SourceChainID: uuid.New(),
		DestChainID:   uuid.New(),
		DstEID:        1,
		PeerHex:       "0xpeer",
		OptionsHex:    "0x",
		IsActive:      true,
	})
	require.Error(t, err)

	err = repo.Delete(ctx, uuid.New())
	require.Error(t, err)
}
