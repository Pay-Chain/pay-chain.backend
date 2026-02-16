package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/pkg/utils"
)

func TestTokenRepository_GetAllTokens_QueryBranches(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createTokenTable(t, db)
	repo := NewTokenRepository(db, nil)
	ctx := context.Background()

	chainID := uuid.New()
	seedChain(t, db, chainID.String(), "8453", "Base", "EVM", true)
	mustExec(t, db, `INSERT INTO tokens (id, chain_id, symbol, name, decimals, address, type, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New().String(), chainID.String(), "USDC", "USD Coin", 6, "0x1", "ERC20", true, false, true, "0", nil)
	mustExec(t, db, `INSERT INTO tokens (id, chain_id, symbol, name, decimals, address, type, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New().String(), chainID.String(), "IDRX", "IDRX", 6, "0x2", "ERC20", true, false, false, "0", nil)

	// chain filter branch
	items, total, err := repo.GetAllTokens(ctx, &chainID, nil, utils.GetPaginationParams(1, 10))
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	// search branch under sqlite (ILIKE unsupported) should hit query error branch.
	search := "USD"
	_, _, err = repo.GetAllTokens(ctx, nil, &search, utils.GetPaginationParams(1, 10))
	require.Error(t, err)
}
