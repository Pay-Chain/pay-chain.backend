package repositories

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v8"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
	"pay-chain.backend/pkg/utils"
)

type stubChainRepo struct {
	chains map[uuid.UUID]*entities.Chain
}

func (s *stubChainRepo) GetByID(_ context.Context, id uuid.UUID) (*entities.Chain, error) {
	if c, ok := s.chains[id]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}

func (s *stubChainRepo) GetByChainID(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *stubChainRepo) GetByCAIP2(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *stubChainRepo) GetAll(context.Context) ([]*entities.Chain, error) { return nil, nil }
func (s *stubChainRepo) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *stubChainRepo) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *stubChainRepo) Create(context.Context, *entities.Chain) error       { return nil }
func (s *stubChainRepo) Update(context.Context, *entities.Chain) error       { return nil }
func (s *stubChainRepo) Delete(context.Context, uuid.UUID) error             { return nil }
func (s *stubChainRepo) CreateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *stubChainRepo) UpdateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *stubChainRepo) DeleteRPC(context.Context, uuid.UUID) error          { return nil }
func (s *stubChainRepo) GetRPCByID(context.Context, uuid.UUID) (*entities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}

func TestSmartContractRepository_CRUDAndFilters(t *testing.T) {
	db := newTestDB(t)
	createSmartContractTable(t, db)
	ctx := context.Background()

	chainID := uuid.New()
	repo := NewSmartContractRepository(db, &stubChainRepo{
		chains: map[uuid.UUID]*entities.Chain{
			chainID: {ID: chainID, ChainID: "eip155:8453"},
		},
	})

	id := uuid.New()
	err := repo.Create(ctx, &entities.SmartContract{
		ID:              id,
		Name:            "Router",
		Type:            entities.ContractTypeRouter,
		Version:         "1.0.0",
		ChainUUID:       chainID,
		ContractAddress: "0xabc",
		DeployerAddress: null.StringFrom("0xdeployer"),
		ABI:             []map[string]any{{"name": "fn"}},
		Metadata:        null.JSONFrom([]byte(`{"ok":true}`)),
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "eip155:8453", got.BlockchainID)
	require.Equal(t, entities.ContractTypeRouter, got.Type)

	byAddr, err := repo.GetByChainAndAddress(ctx, chainID, "0xabc")
	require.NoError(t, err)
	require.NotNil(t, byAddr)

	active, err := repo.GetActiveContract(ctx, chainID, entities.ContractTypeRouter)
	require.NoError(t, err)
	require.NotNil(t, active)

	allByChain, totalByChain, err := repo.GetByChain(ctx, chainID, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), totalByChain)
	require.Len(t, allByChain, 1)

	filtered, totalFiltered, err := repo.GetFiltered(ctx, &chainID, entities.ContractTypeRouter, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), totalFiltered)
	require.Len(t, filtered, 1)

	got.Name = "Router V2"
	require.NoError(t, repo.Update(ctx, got))

	all, total, err := repo.GetAll(ctx, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, all, 1)

	require.NoError(t, repo.SoftDelete(ctx, id))
	nilByID, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Nil(t, nilByID)

	require.ErrorIs(t, repo.Update(ctx, &entities.SmartContract{ID: uuid.New()}), domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.SoftDelete(ctx, uuid.New()), domainerrors.ErrNotFound)
}

func TestSmartContractRepository_NotFoundAndDecodeBranches(t *testing.T) {
	db := newTestDB(t)
	createSmartContractTable(t, db)
	ctx := context.Background()

	chainID := uuid.New()
	repo := NewSmartContractRepository(db, &stubChainRepo{
		chains: map[uuid.UUID]*entities.Chain{
			chainID: {ID: chainID, ChainID: "8453", Name: "Base"},
		},
	})

	// Not found branches
	byAddr, err := repo.GetByChainAndAddress(ctx, chainID, "0xdoesnotexist")
	require.NoError(t, err)
	require.Nil(t, byAddr)

	active, err := repo.GetActiveContract(ctx, chainID, entities.ContractTypeRouter)
	require.NoError(t, err)
	require.Nil(t, active)

	// Insert inactive and malformed ABI row directly to cover decode-safe path.
	rawID := uuid.New()
	mustExec(t, db, `INSERT INTO smart_contracts (
		id,name,type,version,chain_id,address,deployer_address,is_active,abi,metadata,start_block,created_at,updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		rawID.String(),
		"Broken ABI Contract",
		string(entities.ContractTypeGateway),
		"1.0.0",
		chainID.String(),
		"0x3333333333333333333333333333333333333333",
		"",
		false,
		`{invalid-json`,
		`{"test":true}`,
		0,
		time.Now(),
		time.Now(),
	)

	// Should not fail even with malformed ABI; ABI should be nil-ish and entity returned.
	got, err := repo.GetByID(ctx, rawID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "Broken ABI Contract", got.Name)
	require.Equal(t, "8453", got.BlockchainID)
	require.Nil(t, got.ABI)

	// Inactive row should not be returned by GetActiveContract
	active, err = repo.GetActiveContract(ctx, chainID, entities.ContractTypeGateway)
	require.NoError(t, err)
	require.Nil(t, active)
}

func TestSmartContractRepository_CreateMarshalError(t *testing.T) {
	db := newTestDB(t)
	createSmartContractTable(t, db)
	ctx := context.Background()

	chainID := uuid.New()
	repo := NewSmartContractRepository(db, &stubChainRepo{
		chains: map[uuid.UUID]*entities.Chain{
			chainID: {ID: chainID, ChainID: "eip155:8453"},
		},
	})

	err := repo.Create(ctx, &entities.SmartContract{
		ID:              uuid.New(),
		Name:            "Bad ABI",
		Type:            entities.ContractTypeRouter,
		Version:         "1.0.0",
		ChainUUID:       chainID,
		ContractAddress: "0xabc",
		ABI:             []map[string]interface{}{{"bad": func() {}}},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to marshal ABI")
}

func TestSmartContractRepository_Query_DBErrorAndFilterCompat(t *testing.T) {
	ctx := context.Background()

	t.Run("db errors when table missing", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewSmartContractRepository(db, &stubChainRepo{})
		chainID := uuid.New()

		_, _, err := repo.GetByChain(ctx, chainID, utils.PaginationParams{Page: 1, Limit: 10})
		require.Error(t, err)

		_, _, err = repo.GetAll(ctx, utils.PaginationParams{Page: 1, Limit: 10})
		require.Error(t, err)

		_, _, err = repo.GetFiltered(ctx, &chainID, entities.ContractTypeRouter, utils.PaginationParams{Page: 1, Limit: 10})
		require.Error(t, err)
	})

	t.Run("filter compat for pool and DEX_POOL type", func(t *testing.T) {
		db := newTestDB(t)
		createSmartContractTable(t, db)
		chainID := uuid.New()
		repo := NewSmartContractRepository(db, &stubChainRepo{
			chains: map[uuid.UUID]*entities.Chain{
				chainID: {ID: chainID, ChainID: "eip155:8453"},
			},
		})

		// Legacy DEX_POOL row.
		mustExec(t, db, `INSERT INTO smart_contracts (id,name,type,version,chain_id,address,deployer_address,is_active,abi,metadata,start_block,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), "Legacy Pool", "DEX_POOL", "1.0.0", chainID.String(),
			"0x1111111111111111111111111111111111111111", "", true, `[]`, `{}`, 0, time.Now(), time.Now(),
		)

		// New POOL row.
		mustExec(t, db, `INSERT INTO smart_contracts (id,name,type,version,chain_id,address,deployer_address,is_active,abi,metadata,start_block,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), "New Pool", string(entities.ContractTypePool), "1.0.0", chainID.String(),
			"0x2222222222222222222222222222222222222222", "", true, `[]`, `{}`, 0, time.Now(), time.Now(),
		)

		items, total, err := repo.GetFiltered(ctx, &chainID, entities.ContractTypePool, utils.PaginationParams{Page: 1, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, items, 2)
	})
}

func TestSmartContractRepository_DBErrorBranches_OnSingleLookups(t *testing.T) {
	db := newTestDB(t)
	// intentionally skip table creation
	repo := NewSmartContractRepository(db, &stubChainRepo{})
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.Error(t, err)

	_, err = repo.GetByChainAndAddress(ctx, uuid.New(), "0xabc")
	require.Error(t, err)

	_, err = repo.GetActiveContract(ctx, uuid.New(), entities.ContractTypeRouter)
	require.Error(t, err)

	err = repo.SoftDelete(ctx, uuid.New())
	require.Error(t, err)
}

func TestSmartContractRepository_Create_DBErrorBranch(t *testing.T) {
	db := newTestDB(t)
	// Intentionally skip table creation.
	repo := NewSmartContractRepository(db, &stubChainRepo{})
	ctx := context.Background()

	err := repo.Create(ctx, &entities.SmartContract{
		ID:              uuid.New(),
		Name:            "Router",
		Type:            entities.ContractTypeRouter,
		Version:         "1.0.0",
		ChainUUID:       uuid.New(),
		ContractAddress: "0x1111111111111111111111111111111111111111",
		ABI:             []map[string]any{{"name": "fn"}},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	require.Error(t, err)
}

func TestSmartContractRepository_Update_DBErrorBranch(t *testing.T) {
	db := newTestDB(t)
	// Intentionally skip table creation.
	repo := NewSmartContractRepository(db, &stubChainRepo{})
	ctx := context.Background()

	err := repo.Update(ctx, &entities.SmartContract{
		ID:              uuid.New(),
		Name:            "Router",
		Type:            entities.ContractTypeRouter,
		Version:         "1.0.0",
		ChainUUID:       uuid.New(),
		ContractAddress: "0x1111111111111111111111111111111111111111",
		Metadata:        null.JSONFrom([]byte(`{}`)),
		IsActive:        true,
	})
	require.Error(t, err)
}

func TestSmartContractRepository_List_FindErrorAfterCount(t *testing.T) {
	db := newTestDB(t)
	createSmartContractTable(t, db)
	ctx := context.Background()

	chainID := uuid.New()
	repo := NewSmartContractRepository(db, &stubChainRepo{
		chains: map[uuid.UUID]*entities.Chain{
			chainID: {ID: chainID, ChainID: "8453"},
		},
	})

	cbName := "test:smart_contract_find_error_after_count"
	queryCount := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "smart_contracts" {
			queryCount++
			if queryCount > 1 {
				tx.AddError(gorm.ErrInvalidDB)
			}
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove(cbName) })

	_, _, err := repo.GetByChain(ctx, chainID, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)

	queryCount = 0
	_, _, err = repo.GetAll(ctx, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)
}

func TestSmartContractRepository_ToEntity_NullAndInvalidJSONBranches(t *testing.T) {
	repo := &SmartContractRepositoryImpl{
		chainRepo: &stubChainRepo{
			chains: map[uuid.UUID]*entities.Chain{},
		},
	}

	t.Run("null abi and null metadata fallback", func(t *testing.T) {
		m := &models.SmartContract{
			ID:              uuid.New(),
			Name:            "Null ABI",
			Type:            string(entities.ContractTypeRouter),
			Version:         "1.0.0",
			ChainID:         uuid.Nil,
			ContractAddress: "0xabc",
			ABI:             "null",
			Metadata:        "null",
			IsActive:        true,
		}
		e := repo.toEntity(m)
		require.Nil(t, e.ABI)
		require.Equal(t, "{}", string(e.Metadata.JSON))
		require.Equal(t, "", e.BlockchainID)
	})

	t.Run("invalid abi json keeps entity alive", func(t *testing.T) {
		chainID := uuid.New()
		m := &models.SmartContract{
			ID:              uuid.New(),
			Name:            "Broken ABI",
			Type:            string(entities.ContractTypeRouter),
			Version:         "1.0.0",
			ChainID:         chainID,
			ContractAddress: "0xdef",
			ABI:             "{not-json}",
			Metadata:        "{}",
			IsActive:        true,
		}
		e := repo.toEntity(m)
		require.Nil(t, e.ABI)
		var meta map[string]interface{}
		require.NoError(t, json.Unmarshal(e.Metadata.JSON, &meta))
		require.Equal(t, "", e.BlockchainID)
	})
}

func TestSmartContractRepository_GetFiltered_NoTypeAndNoChainFilter(t *testing.T) {
	db := newTestDB(t)
	createSmartContractTable(t, db)
	ctx := context.Background()

	repo := NewSmartContractRepository(db, &stubChainRepo{})
	chainID := uuid.New()

	mustExec(t, db, `INSERT INTO smart_contracts (id,name,type,version,chain_id,address,deployer_address,is_active,abi,metadata,start_block,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		uuid.New().String(), "Gateway", "GATEWAY", "1.0.0", chainID.String(), "0x1", "", true, "[]", "{}", 0, time.Now(), time.Now())
	mustExec(t, db, `INSERT INTO smart_contracts (id,name,type,version,chain_id,address,deployer_address,is_active,abi,metadata,start_block,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		uuid.New().String(), "Router", "ROUTER", "1.0.0", chainID.String(), "0x2", "", true, "[]", "{}", 0, time.Now(), time.Now())

	items, total, err := repo.GetFiltered(ctx, nil, "", utils.PaginationParams{Page: 1, Limit: 0})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)
}

func TestSmartContractRepository_GetFiltered_FindErrorAfterCount(t *testing.T) {
	db := newTestDB(t)
	createSmartContractTable(t, db)
	ctx := context.Background()

	chainID := uuid.New()
	repo := NewSmartContractRepository(db, &stubChainRepo{
		chains: map[uuid.UUID]*entities.Chain{
			chainID: {ID: chainID, ChainID: "8453"},
		},
	})

	mustExec(t, db, `INSERT INTO smart_contracts (id,name,type,version,chain_id,address,deployer_address,is_active,abi,metadata,start_block,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		uuid.New().String(), "Router", "ROUTER", "1.0.0", chainID.String(), "0x1", "", true, "[]", "{}", 0, time.Now(), time.Now())

	cbName := "test:smart_contract_filtered_find_error_after_count"
	queryCount := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "smart_contracts" {
			queryCount++
			if queryCount > 1 {
				tx.AddError(gorm.ErrInvalidDB)
			}
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove(cbName) })

	_, _, err := repo.GetFiltered(ctx, &chainID, entities.ContractTypeRouter, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)
}
