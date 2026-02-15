package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

func TestTokenRepository_CRUDAndQueries(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createTokenTable(t, db)
	repo := NewTokenRepository(db, nil)
	ctx := context.Background()

	chainID := uuid.New()
	seedChain(t, db, chainID.String(), "8453", "Base", "EVM", true)
	now := time.Now()

	tokenID := uuid.New()
	nativeID := uuid.New()
	mustExec(t, db, `INSERT INTO tokens(id,chain_id,symbol,name,decimals,address,type,logo_url,is_active,is_native,is_stablecoin,min_amount,max_amount,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, tokenID.String(), chainID.String(), "USDC", "USD Coin", 6, "0x8335", "ERC20", "", true, false, true, "0", nil, now, now)
	mustExec(t, db, `INSERT INTO tokens(id,chain_id,symbol,name,decimals,address,type,logo_url,is_active,is_native,is_stablecoin,min_amount,max_amount,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, nativeID.String(), chainID.String(), "ETH", "Ether", 18, "", "NATIVE", "", true, true, false, "0", nil, now, now)

	byID, err := repo.GetByID(ctx, tokenID)
	require.NoError(t, err)
	require.Equal(t, "USDC", byID.Symbol)

	bySym, err := repo.GetBySymbol(ctx, "USDC", chainID)
	require.NoError(t, err)
	require.Equal(t, tokenID, bySym.ID)

	byAddr, err := repo.GetByAddress(ctx, "0x8335", chainID)
	require.NoError(t, err)
	require.Equal(t, tokenID, byAddr.ID)

	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)

	stable, err := repo.GetStablecoins(ctx)
	require.NoError(t, err)
	require.Len(t, stable, 1)
	require.Equal(t, "USDC", stable[0].Symbol)

	native, err := repo.GetNative(ctx, chainID)
	require.NoError(t, err)
	require.Equal(t, "ETH", native.Symbol)

	byChain, total, err := repo.GetTokensByChain(ctx, chainID, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, byChain, 2)

	allFiltered, totalFiltered, err := repo.GetAllTokens(ctx, &chainID, nil, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), totalFiltered)
	require.Len(t, allFiltered, 2)

	byID.Name = "USD Coin Updated"
	require.NoError(t, repo.Update(ctx, byID))
	require.NoError(t, repo.SoftDelete(ctx, byID.ID))

	_, err = repo.GetByID(ctx, byID.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestTokenRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createTokenTable(t, db)
	repo := NewTokenRepository(db, nil)
	ctx := context.Background()
	id := uuid.New()
	chainID := uuid.New()

	_, err := repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetBySymbol(ctx, "NOPE", chainID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByAddress(ctx, "0xnope", chainID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetNative(ctx, chainID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}
