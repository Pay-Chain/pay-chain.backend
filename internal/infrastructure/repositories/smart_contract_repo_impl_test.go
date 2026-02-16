package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v8"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
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
func (s *stubChainRepo) Create(context.Context, *entities.Chain) error { return nil }
func (s *stubChainRepo) Update(context.Context, *entities.Chain) error { return nil }
func (s *stubChainRepo) Delete(context.Context, uuid.UUID) error       { return nil }

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
