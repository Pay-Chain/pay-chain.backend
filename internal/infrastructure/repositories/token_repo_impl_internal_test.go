package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v8"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/models"
)

func TestTokenRepository_ToEntityAndToModel_InternalBranches(t *testing.T) {
	repo := NewTokenRepository(newTestDB(t), nil)
	now := time.Now()

	model := &models.Token{
		ID:          uuid.New(),
		ChainID:     uuid.New(),
		Symbol:      "USDC",
		Name:        "USD Coin",
		Decimals:    6,
		Type:        "ERC20",
		IsActive:    true,
		IsStablecoin:true,
		MinAmount:   "0",
		MaxAmount:   nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Branch: chain not preloaded (uuid.Nil) => blockchain id fallback empty.
	entity := repo.toEntity(model)
	require.NotNil(t, entity)
	require.Equal(t, "", entity.BlockchainID)
	require.Nil(t, entity.Chain)

	// Branch: toModel maps null max amount pointer.
	entity.MaxAmount = null.StringFrom("100")
	back := repo.toModel(entity)
	require.NotNil(t, back.MaxAmount)
	require.Equal(t, "100", *back.MaxAmount)
}

func TestTokenRepository_Create_ErrorBranch(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createTokenTable(t, db)
	repo := NewTokenRepository(db, nil)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	// Closed DB connection should force repository Create() error branch.
	err = repo.Create(context.Background(), &entities.Token{
		ID:              uuid.New(),
		ChainUUID:       uuid.Nil,
		Symbol:          "BAD",
		Name:            "Bad",
		Decimals:        6,
		ContractAddress: "0x0",
		Type:            entities.TokenTypeERC20,
		IsActive:        true,
		IsNative:        false,
		IsStablecoin:    false,
		MinAmount:       "0",
	})
	require.Error(t, err)
}
