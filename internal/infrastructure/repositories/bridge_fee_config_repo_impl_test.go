package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

func TestBridgeConfigRepository_CRUDAndNotFound(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	createBridgeAndFeeTables(t, db)
	ctx := context.Background()
	repo := NewBridgeConfigRepository(db)

	bridgeID := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()
	id := uuid.New()

	mustExec(t, db, `INSERT INTO payment_bridge(id,name,created_at,updated_at) VALUES (?,?,?,?)`,
		bridgeID.String(), "Hyperbridge", time.Now(), time.Now())

	err := repo.Create(ctx, &entities.BridgeConfig{
		ID:            id,
		BridgeID:      bridgeID,
		SourceChainID: sourceID,
		DestChainID:   destID,
		RouterAddress: "0xrouter",
		FeePercentage: "0.1",
		Config:        `{"k":"v"}`,
		IsActive:      true,
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, bridgeID, got.BridgeID)
	require.NotNil(t, got.Bridge)

	active, err := repo.GetActive(ctx, sourceID, destID)
	require.NoError(t, err)
	require.Equal(t, id, active.ID)

	items, total, err := repo.List(ctx, &sourceID, &destID, &bridgeID, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	got.IsActive = false
	require.NoError(t, repo.Update(ctx, got))
	_, err = repo.GetActive(ctx, sourceID, destID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	require.NoError(t, repo.Delete(ctx, id))
	_, err = repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	require.ErrorIs(t, repo.Update(ctx, &entities.BridgeConfig{ID: uuid.New()}), domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.Delete(ctx, uuid.New()), domainerrors.ErrNotFound)
}

func TestBridgeConfigRepository_Create_AssignsIDWhenNil(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	createBridgeAndFeeTables(t, db)
	ctx := context.Background()
	repo := NewBridgeConfigRepository(db)

	bridgeID := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()
	mustExec(t, db, `INSERT INTO payment_bridge(id,name,created_at,updated_at) VALUES (?,?,?,?)`,
		bridgeID.String(), "CCIP", time.Now(), time.Now())

	item := &entities.BridgeConfig{
		BridgeID:      bridgeID,
		SourceChainID: sourceID,
		DestChainID:   destID,
		RouterAddress: "0xrouter",
		FeePercentage: "0.2",
		Config:        "{}",
		IsActive:      true,
	}
	require.NoError(t, repo.Create(ctx, item))
	require.NotEqual(t, uuid.Nil, item.ID)
}

func TestFeeConfigRepository_CRUDAndNotFound(t *testing.T) {
	db := newTestDB(t)
	createBridgeAndFeeTables(t, db)
	ctx := context.Background()
	repo := NewFeeConfigRepository(db)

	chainID := uuid.New()
	tokenID := uuid.New()
	id := uuid.New()

	maxFee := "99"
	err := repo.Create(ctx, &entities.FeeConfig{
		ID:                 id,
		ChainID:            chainID,
		TokenID:            tokenID,
		PlatformFeePercent: "0.5",
		FixedBaseFee:       "1",
		MinFee:             "1",
		MaxFee:             &maxFee,
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, chainID, got.ChainID)

	gotByRoute, err := repo.GetByChainAndToken(ctx, chainID, tokenID)
	require.NoError(t, err)
	require.Equal(t, id, gotByRoute.ID)

	items, total, err := repo.List(ctx, &chainID, &tokenID, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	got.MinFee = "2"
	require.NoError(t, repo.Update(ctx, got))

	require.NoError(t, repo.Delete(ctx, id))
	_, err = repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
	_, err = repo.GetByChainAndToken(ctx, chainID, tokenID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	require.ErrorIs(t, repo.Update(ctx, &entities.FeeConfig{ID: uuid.New()}), domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.Delete(ctx, uuid.New()), domainerrors.ErrNotFound)
}

func TestFeeConfigRepository_Create_AssignsIDWhenNil(t *testing.T) {
	db := newTestDB(t)
	createBridgeAndFeeTables(t, db)
	ctx := context.Background()
	repo := NewFeeConfigRepository(db)

	item := &entities.FeeConfig{
		ChainID:            uuid.New(),
		TokenID:            uuid.New(),
		PlatformFeePercent: "0.1",
		FixedBaseFee:       "1",
		MinFee:             "0",
	}
	require.NoError(t, repo.Create(ctx, item))
	require.NotEqual(t, uuid.Nil, item.ID)
}

func TestBridgeAndFeeConfigRepository_DBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// intentionally skip table creation
	ctx := context.Background()

	bridgeRepo := NewBridgeConfigRepository(db)
	feeRepo := NewFeeConfigRepository(db)

	_, err := bridgeRepo.GetByID(ctx, uuid.New())
	require.Error(t, err)
	_, err = bridgeRepo.GetActive(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
	_, _, err = bridgeRepo.List(ctx, nil, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)
	err = bridgeRepo.Update(ctx, &entities.BridgeConfig{ID: uuid.New()})
	require.Error(t, err)
	err = bridgeRepo.Delete(ctx, uuid.New())
	require.Error(t, err)

	_, err = feeRepo.GetByID(ctx, uuid.New())
	require.Error(t, err)
	_, err = feeRepo.GetByChainAndToken(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
	_, _, err = feeRepo.List(ctx, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)
	err = feeRepo.Update(ctx, &entities.FeeConfig{ID: uuid.New()})
	require.Error(t, err)
	err = feeRepo.Delete(ctx, uuid.New())
	require.Error(t, err)
}
