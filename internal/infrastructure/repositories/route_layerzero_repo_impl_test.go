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

func TestRoutePolicyRepository_CRUDAndNotFound(t *testing.T) {
	db := newTestDB(t)
	createRoutePolicyTables(t, db)
	ctx := context.Background()
	repo := NewRoutePolicyRepository(db)

	id := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()

	err := repo.Create(ctx, &entities.RoutePolicy{
		ID:                id,
		SourceChainID:     sourceID,
		DestChainID:       destID,
		DefaultBridgeType: 0,
		FallbackMode:      entities.BridgeFallbackModeAutoFallback,
		FallbackOrder:     []uint8{0, 1},
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, sourceID, got.SourceChainID)
	require.Equal(t, []uint8{0, 1}, got.FallbackOrder)

	byRoute, err := repo.GetByRoute(ctx, sourceID, destID)
	require.NoError(t, err)
	require.Equal(t, id, byRoute.ID)

	items, total, err := repo.List(ctx, &sourceID, &destID, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	got.FallbackOrder = []uint8{2}
	require.NoError(t, repo.Update(ctx, got))

	require.NoError(t, repo.Delete(ctx, id))
	_, err = repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
	_, err = repo.GetByRoute(ctx, sourceID, destID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	require.ErrorIs(t, repo.Update(ctx, &entities.RoutePolicy{ID: uuid.New()}), domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.Delete(ctx, uuid.New()), domainerrors.ErrNotFound)
}

func TestRoutePolicyRepository_Create_DefaultsAndGeneratedID(t *testing.T) {
	db := newTestDB(t)
	createRoutePolicyTables(t, db)
	ctx := context.Background()
	repo := NewRoutePolicyRepository(db)

	sourceID := uuid.New()
	destID := uuid.New()
	policy := &entities.RoutePolicy{
		SourceChainID:     sourceID,
		DestChainID:       destID,
		DefaultBridgeType: 1,
		// FallbackMode intentionally empty to hit default strict mode branch.
		FallbackOrder: nil, // exercise marshal fallback default ordering branch.
	}

	require.NoError(t, repo.Create(ctx, policy))
	require.NotEqual(t, uuid.Nil, policy.ID)

	got, err := repo.GetByID(ctx, policy.ID)
	require.NoError(t, err)
	require.Equal(t, entities.BridgeFallbackModeStrict, got.FallbackMode)
	require.Equal(t, []uint8{0}, got.FallbackOrder)
}

func TestLayerZeroConfigRepository_CRUDAndNotFound(t *testing.T) {
	db := newTestDB(t)
	createRoutePolicyTables(t, db)
	ctx := context.Background()
	repo := NewLayerZeroConfigRepository(db)

	id := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()

	err := repo.Create(ctx, &entities.LayerZeroConfig{
		ID:            id,
		SourceChainID: sourceID,
		DestChainID:   destID,
		DstEID:        30110,
		PeerHex:       "0x1234",
		OptionsHex:    "  ",
		IsActive:      true,
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, uint32(30110), got.DstEID)
	require.Equal(t, "0x", got.OptionsHex)

	byRoute, err := repo.GetByRoute(ctx, sourceID, destID)
	require.NoError(t, err)
	require.Equal(t, id, byRoute.ID)

	activeOnly := true
	items, total, err := repo.List(ctx, &sourceID, &destID, &activeOnly, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	got.OptionsHex = ""
	require.NoError(t, repo.Update(ctx, got))

	require.NoError(t, repo.Delete(ctx, id))
	_, err = repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
	_, err = repo.GetByRoute(ctx, sourceID, destID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	require.ErrorIs(t, repo.Update(ctx, &entities.LayerZeroConfig{ID: uuid.New()}), domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.Delete(ctx, uuid.New()), domainerrors.ErrNotFound)
}
