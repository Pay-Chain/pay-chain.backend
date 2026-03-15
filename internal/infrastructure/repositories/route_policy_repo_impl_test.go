package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/pkg/utils"
)

func TestRoutePolicyRepo_UpdateAndListBranches(t *testing.T) {
	db := newTestDB(t)
	mustExec(t, db, `CREATE TABLE route_policies (
		id TEXT PRIMARY KEY,
		source_chain_id TEXT NOT NULL,
		dest_chain_id TEXT NOT NULL,
		default_bridge_type INTEGER NOT NULL,
		fallback_mode TEXT NOT NULL,
		fallback_order TEXT NOT NULL,
		supports_token_bridge BOOLEAN,
		supports_dest_swap BOOLEAN,
		supports_privacy_forward BOOLEAN,
		bridge_token TEXT,
		status TEXT,
		per_byte_rate TEXT,
		overhead_bytes TEXT,
		min_fee TEXT,
		max_fee TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)

	repo := NewRoutePolicyRepository(db)
	ctx := context.Background()

	sourceID := uuid.New()
	destID := uuid.New()
	policy := &entities.RoutePolicy{
		ID:                uuid.New(),
		SourceChainID:     sourceID,
		DestChainID:       destID,
		DefaultBridgeType: 1,
		FallbackMode:      entities.BridgeFallbackModeAutoFallback,
		FallbackOrder:     []uint8{1, 0},
	}
	require.NoError(t, repo.Create(ctx, policy))

	// Update existing row success.
	policy.DefaultBridgeType = 2
	policy.FallbackMode = ""
	policy.FallbackOrder = nil
	policy.SupportsTokenBridge = true
	policy.SupportsDestSwap = true
	policy.SupportsPrivacyForward = true
	policy.BridgeToken = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	policy.Status = "paused"
	require.NoError(t, repo.Update(ctx, policy))

	got, err := repo.GetByID(ctx, policy.ID)
	require.NoError(t, err)
	require.Equal(t, uint8(2), got.DefaultBridgeType)
	require.Equal(t, entities.BridgeFallbackModeStrict, got.FallbackMode)
	require.Equal(t, []uint8{0}, got.FallbackOrder)
	require.True(t, got.SupportsTokenBridge)
	require.True(t, got.SupportsDestSwap)
	require.True(t, got.SupportsPrivacyForward)
	require.Equal(t, "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", got.BridgeToken)
	require.Equal(t, "paused", got.Status)

	// Update not found branch.
	missing := &entities.RoutePolicy{
		ID:                uuid.New(),
		SourceChainID:     sourceID,
		DestChainID:       destID,
		DefaultBridgeType: 0,
		FallbackMode:      entities.BridgeFallbackModeStrict,
		FallbackOrder:     []uint8{0},
	}
	err = repo.Update(ctx, missing)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	// List with filters and pagination.
	items, total, err := repo.List(ctx, &sourceID, &destID, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	// Soft delete and not found branch in GetByRoute.
	mustExec(t, db, `UPDATE route_policies SET deleted_at = ? WHERE id = ?`, time.Now(), policy.ID.String())
	_, err = repo.GetByRoute(ctx, sourceID, destID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestRoutePolicyRepo_Update_DBErrorBranch(t *testing.T) {
	db := newTestDB(t)
	// table is intentionally missing to force db error path.
	repo := NewRoutePolicyRepository(db)
	ctx := context.Background()

	err := repo.Update(ctx, &entities.RoutePolicy{
		ID:                uuid.New(),
		SourceChainID:     uuid.New(),
		DestChainID:       uuid.New(),
		DefaultBridgeType: 1,
		FallbackMode:      entities.BridgeFallbackModeStrict,
		FallbackOrder:     []uint8{0},
	})
	require.Error(t, err)
}

func TestRoutePolicyRepo_DeleteAndGetByIDBranches(t *testing.T) {
	db := newTestDB(t)
	mustExec(t, db, `CREATE TABLE route_policies (
		id TEXT PRIMARY KEY,
		source_chain_id TEXT NOT NULL,
		dest_chain_id TEXT NOT NULL,
		default_bridge_type INTEGER NOT NULL,
		fallback_mode TEXT NOT NULL,
		fallback_order TEXT NOT NULL,
		supports_token_bridge BOOLEAN,
		supports_dest_swap BOOLEAN,
		supports_privacy_forward BOOLEAN,
		bridge_token TEXT,
		status TEXT,
		per_byte_rate TEXT,
		overhead_bytes TEXT,
		min_fee TEXT,
		max_fee TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)

	repo := NewRoutePolicyRepository(db)
	ctx := context.Background()

	policy := &entities.RoutePolicy{
		ID:                uuid.New(),
		SourceChainID:     uuid.New(),
		DestChainID:       uuid.New(),
		DefaultBridgeType: 0,
		FallbackMode:      entities.BridgeFallbackModeStrict,
		FallbackOrder:     []uint8{0},
	}
	require.NoError(t, repo.Create(ctx, policy))

	_, err := repo.GetByID(ctx, policy.ID)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, policy.ID))

	_, err = repo.GetByID(ctx, policy.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Delete(ctx, policy.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestRoutePolicyRepo_List_DBErrorBranch(t *testing.T) {
	db := newTestDB(t)
	// intentionally skip table creation
	repo := NewRoutePolicyRepository(db)
	ctx := context.Background()

	_, _, err := repo.List(ctx, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)
}

func TestRoutePolicyRepo_Create_DefaultsAndDBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	mustExec(t, db, `CREATE TABLE route_policies (
		id TEXT PRIMARY KEY,
		source_chain_id TEXT NOT NULL,
		dest_chain_id TEXT NOT NULL,
		default_bridge_type INTEGER NOT NULL,
		fallback_mode TEXT NOT NULL,
		fallback_order TEXT NOT NULL,
		supports_token_bridge BOOLEAN,
		supports_dest_swap BOOLEAN,
		supports_privacy_forward BOOLEAN,
		bridge_token TEXT,
		status TEXT,
		per_byte_rate TEXT,
		overhead_bytes TEXT,
		min_fee TEXT,
		max_fee TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)

	repo := NewRoutePolicyRepository(db)
	ctx := context.Background()

	p := &entities.RoutePolicy{
		// intentionally nil ID + empty fallback settings to hit default branches
		SourceChainID:     uuid.New(),
		DestChainID:       uuid.New(),
		DefaultBridgeType: 0,
	}
	require.NoError(t, repo.Create(ctx, p))
	require.NotEqual(t, uuid.Nil, p.ID)

	got, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, entities.BridgeFallbackModeStrict, got.FallbackMode)
	require.Equal(t, []uint8{0}, got.FallbackOrder)
	require.Equal(t, "active", got.Status)
	require.False(t, got.SupportsTokenBridge)
	require.False(t, got.SupportsDestSwap)
	require.False(t, got.SupportsPrivacyForward)

	// DB error branch for Create
	badDB := newTestDB(t)
	badRepo := NewRoutePolicyRepository(badDB)
	err = badRepo.Create(ctx, &entities.RoutePolicy{
		SourceChainID:     uuid.New(),
		DestChainID:       uuid.New(),
		DefaultBridgeType: 0,
		FallbackMode:      entities.BridgeFallbackModeStrict,
		FallbackOrder:     []uint8{0},
	})
	require.Error(t, err)
}
