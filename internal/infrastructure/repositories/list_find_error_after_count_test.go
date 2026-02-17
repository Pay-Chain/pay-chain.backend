package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"pay-chain.backend/pkg/utils"
)

func registerFindErrorAfterCount(t *testing.T, db *gorm.DB, table string) {
	t.Helper()
	cbName := "test:find_error_after_count:" + table
	queryCount := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == table {
			queryCount++
			if queryCount > 1 {
				tx.AddError(gorm.ErrInvalidDB)
			}
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove(cbName) })
}

func TestRepository_List_FindErrorAfterCount_Branches(t *testing.T) {
	ctx := context.Background()

	t.Run("fee config list find error after count", func(t *testing.T) {
		db := newTestDB(t)
		createBridgeAndFeeTables(t, db)
		repo := NewFeeConfigRepository(db)

		registerFindErrorAfterCount(t, db, "fee_configs")
		_, _, err := repo.List(ctx, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
		require.Error(t, err)
	})

	t.Run("bridge config list find error after count", func(t *testing.T) {
		db := newTestDB(t)
		createPaymentBridgeTable(t, db)
		createBridgeAndFeeTables(t, db)
		repo := NewBridgeConfigRepository(db)

		registerFindErrorAfterCount(t, db, "bridge_configs")
		_, _, err := repo.List(ctx, nil, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
		require.Error(t, err)
	})

	t.Run("route policy list find error after count", func(t *testing.T) {
		db := newTestDB(t)
		createRoutePolicyTables(t, db)
		repo := NewRoutePolicyRepository(db)

		registerFindErrorAfterCount(t, db, "route_policies")
		_, _, err := repo.List(ctx, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
		require.Error(t, err)
	})

	t.Run("layerzero config list find error after count", func(t *testing.T) {
		db := newTestDB(t)
		createRoutePolicyTables(t, db)
		repo := NewLayerZeroConfigRepository(db)

		registerFindErrorAfterCount(t, db, "layerzero_configs")
		_, _, err := repo.List(ctx, nil, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
		require.Error(t, err)
	})

	t.Run("token get all tokens find error after count", func(t *testing.T) {
		db := newTestDB(t)
		createChainTables(t, db)
		createTokenTable(t, db)
		repo := NewTokenRepository(db, nil)

		chainID := uuid.New()
		seedChain(t, db, chainID.String(), "8453", "Base", "EVM", true)

		registerFindErrorAfterCount(t, db, "tokens")
		_, _, err := repo.GetAllTokens(ctx, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
		require.Error(t, err)
	})
}
