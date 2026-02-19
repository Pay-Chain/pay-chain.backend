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

func TestChainRepository_GetByCAIP2_DirectAndMalformedBranches(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()

	fullCAIP2ID := uuid.New()
	seedChain(t, db, fullCAIP2ID.String(), "eip155:8453", "Base CAIP2", "EVM", true)

	// Direct match branch: chain_id stored as full CAIP-2.
	got, err := repo.GetByCAIP2(ctx, "eip155:8453")
	require.NoError(t, err)
	require.Equal(t, fullCAIP2ID, got.ID)

	// Malformed branch: no separator -> not found.
	_, err = repo.GetByCAIP2(ctx, "eip155-8453")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	// Malformed branch: empty reference -> not found.
	_, err = repo.GetByCAIP2(ctx, "eip155:")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestChainRepository_NormalizeOnCreateUpdateAndLookup(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()

	id := uuid.New()
	err := repo.Create(ctx, &entities.Chain{
		ID:             id,
		ChainID:        "eip155:8453",
		Name:           "Base",
		Type:           entities.ChainTypeEVM,
		RPCURL:         "https://rpc.local",
		ExplorerURL:    "https://exp.local",
		CurrencySymbol: "ETH",
		IsActive:       true,
		CreatedAt:      time.Now(),
	})
	require.NoError(t, err)

	// Stored value should be normalized to reference part.
	var stored string
	row := db.Raw(`SELECT chain_id FROM chains WHERE id = ?`, id).Row()
	require.NoError(t, row.Scan(&stored))
	require.Equal(t, "8453", stored)

	// CAIP-2 lookup should still work through normalization in GetByChainID.
	got, err := repo.GetByChainID(ctx, "eip155:8453")
	require.NoError(t, err)
	require.Equal(t, id, got.ID)

	got.Name = "Base Updated"
	got.ChainID = "eip155:8453"
	require.NoError(t, repo.Update(ctx, got))

	row = db.Raw(`SELECT chain_id FROM chains WHERE id = ?`, id).Row()
	require.NoError(t, row.Scan(&stored))
	require.Equal(t, "8453", stored)
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

	// search filter path
	search := "rpc.local"
	items, total, err = repo.GetAllRPCs(ctx, &id, &active, &search, utils.PaginationParams{Page: 1, Limit: 0})
	require.Error(t, err)
	require.Equal(t, int64(0), total)
	require.Nil(t, items)
}

func TestChainRepository_Query_DBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// Intentionally skip creating tables.
	repo := NewChainRepository(db)
	ctx := context.Background()

	_, err := repo.GetAll(ctx)
	require.Error(t, err)

	_, _, err = repo.GetActive(ctx, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)

	_, _, err = repo.GetAllRPCs(ctx, nil, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)

	_, err = repo.GetByID(ctx, uuid.New())
	require.Error(t, err)

	_, err = repo.GetByChainID(ctx, "8453")
	require.Error(t, err)

	_, err = repo.GetByCAIP2(ctx, "eip155:8453")
	require.Error(t, err)

	err = repo.Delete(ctx, uuid.New())
	require.Error(t, err)

	err = repo.Update(ctx, &entities.Chain{
		ID:      uuid.New(),
		ChainID: "8453",
		Name:    "Base",
		Type:    entities.ChainTypeEVM,
	})
	require.Error(t, err)
}

func TestChainRepository_GetActive_FindErrorAfterCount(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()

	cbName := "test:chain_getactive_find_error"
	queryCount := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "chains" {
			queryCount++
			if queryCount > 1 {
				tx.AddError(gorm.ErrInvalidDB)
			}
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove(cbName) })

	_, _, err := repo.GetActive(ctx, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)
}

func TestChainRepository_GetByCAIP2_FallbackQueryError(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()

	cbName := "test:chain_getbycaip2_fallback_error"
	queryCount := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "chains" {
			queryCount++
			if queryCount > 1 {
				tx.AddError(gorm.ErrInvalidDB)
			}
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove(cbName) })

	_, err := repo.GetByCAIP2(ctx, "eip155:8453")
	require.Error(t, err)
}

func TestChainRepository_GetAllRPCs_WithoutOptionalFilters(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()

	id := uuid.New()
	seedChain(t, db, id.String(), "8453", "Base", "EVM", true)
	rpcID := uuid.New()
	mustExec(t, db, `INSERT INTO chain_rpcs(id,chain_id,url,priority,is_active,error_count,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?)`, rpcID.String(), id.String(), "https://rpc2.local", 2, false, 1, time.Now(), time.Now())

	items, total, err := repo.GetAllRPCs(ctx, nil, nil, nil, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, rpcID, items[0].ID)
}

func TestChainRepository_GetAllRPCs_FindErrorAfterCount(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()

	id := uuid.New()
	seedChain(t, db, id.String(), "8453", "Base", "EVM", true)
	rpcID := uuid.New()
	mustExec(t, db, `INSERT INTO chain_rpcs(id,chain_id,url,priority,is_active,error_count,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?)`, rpcID.String(), id.String(), "https://rpc3.local", 3, false, 0, time.Now(), time.Now())

	cbName := "test:chain_getallrpcs_find_error_after_count"
	queryCount := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "chain_rpcs" {
			queryCount++
			if queryCount > 1 {
				tx.AddError(gorm.ErrInvalidDB)
			}
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove(cbName) })

	search := ""
	isActive := false
	_, _, err := repo.GetAllRPCs(ctx, &id, &isActive, &search, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)
}
