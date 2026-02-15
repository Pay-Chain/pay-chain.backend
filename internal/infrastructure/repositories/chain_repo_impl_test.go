package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

func seedChain(t *testing.T, db *gorm.DB, id, chainID, name, chainType string, isActive bool) {
	t.Helper()
	mustExec(t, db, `INSERT INTO chains(id,chain_id,name,type,rpc_url,explorer_url,currency_symbol,image_url,is_active,state_machine_id,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, id, chainID, name, chainType, "https://rpc.local", "https://exp.local", "ETH", "", isActive, "", time.Now(), time.Now())
}

func TestChainRepository_BasicCRUD(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()

	id := uuid.New()
	seedChain(t, db, id.String(), "8453", "Base", "EVM", true)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "8453", got.ChainID)

	got2, err := repo.GetByChainID(ctx, "8453")
	require.NoError(t, err)
	require.Equal(t, id, got2.ID)

	got3, err := repo.GetByCAIP2(ctx, "eip155:8453")
	require.NoError(t, err)
	require.Equal(t, id, got3.ID)

	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)

	active, total, err := repo.GetActive(ctx, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, active, 1)

	got.Name = "Base Updated"
	require.NoError(t, repo.Update(ctx, got))

	require.NoError(t, repo.Delete(ctx, id))
	_, err = repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestChainRepository_NotFoundAndInvalidBranches(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()
	id := uuid.New()

	_, err := repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByChainID(ctx, "missing")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByCAIP2(ctx, "")
	require.ErrorIs(t, err, domainerrors.ErrInvalidInput)

	_, err = repo.GetByCAIP2(ctx, "eip155:999")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Update(ctx, &entities.Chain{ID: id, ChainID: "1", Name: "x", Type: entities.ChainTypeEVM})
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Delete(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestChainRepository_GetAllRPCs(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()

	id := uuid.New()
	seedChain(t, db, id.String(), "8453", "Base", "EVM", true)
	rpcID := uuid.New()
	mustExec(t, db, `INSERT INTO chain_rpcs(id,chain_id,url,priority,is_active,error_count,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?)`, rpcID.String(), id.String(), "https://rpc.local", 1, true, 0, time.Now(), time.Now())

	active := true
	items, total, err := repo.GetAllRPCs(ctx, &id, &active, nil, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, rpcID, items[0].ID)
}
