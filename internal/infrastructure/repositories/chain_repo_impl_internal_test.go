package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/models"
)

func TestChainRepo_ToEntity_ToRpcEntity_AndNamespace(t *testing.T) {
	repo := &chainRepo{}
	now := time.Now()

	chainModel := &models.Chain{
		ID:          uuid.New(),
		NetworkID:   "8453",
		Name:        "Base",
		ChainType:   "evm",
		RPCURL:      "https://main-rpc.example",
		ExplorerURL: "https://explorer.example",
		Symbol:      "ETH",
		LogoURL:     "https://img.example/base.png",
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Branch: toRpcEntity without preloaded chain.
	rpcNoChain := models.ChainRPC{
		ID:        uuid.New(),
		ChainID:   chainModel.ID,
		URL:       "https://rpc-1.example",
		Priority:  100,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	rpcEntityNoChain := repo.toRpcEntity(&rpcNoChain)
	require.NotNil(t, rpcEntityNoChain)
	require.Nil(t, rpcEntityNoChain.Chain)

	// Branch: toEntity with preloaded RPCs.
	chainModel.RPCs = []models.ChainRPC{rpcNoChain}
	entity := repo.toEntity(chainModel)
	require.NotNil(t, entity)
	require.Equal(t, entities.ChainTypeEVM, entity.Type)
	require.Len(t, entity.RPCs, 1)
	require.Equal(t, "https://rpc-1.example", entity.RPCs[0].URL)

	// Branch: toRpcEntity with preloaded chain (recursively mapped).
	rpcWithChain := models.ChainRPC{
		ID:        uuid.New(),
		ChainID:   chainModel.ID,
		URL:       "https://rpc-2.example",
		Priority:  90,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
		Chain:     *chainModel,
	}
	rpcEntityWithChain := repo.toRpcEntity(&rpcWithChain)
	require.NotNil(t, rpcEntityWithChain.Chain)
	require.Equal(t, chainModel.ID, rpcEntityWithChain.Chain.ID)
	require.Equal(t, "8453", rpcEntityWithChain.Chain.ChainID)

	// Branches: getNamespace helper mapping.
	require.Equal(t, "eip155", repo.getNamespace(entities.ChainTypeEVM))
	require.Equal(t, "solana", repo.getNamespace(entities.ChainTypeSVM))
	require.Equal(t, "substrate", repo.getNamespace(entities.ChainTypeSubstrate))
	require.Equal(t, "unknown", repo.getNamespace("OTHER"))
}

func TestChainRepo_GetByCAIP2_Branches(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db)
	ctx := context.Background()

	chainID := uuid.New()
	mustExec(t, db, `INSERT INTO chains (id, chain_id, name, type, rpc_url, is_active) VALUES (?, ?, ?, ?, ?, ?)`,
		chainID.String(), "8453", "Base", "evm", "https://rpc.example", true)

	t.Run("invalid input", func(t *testing.T) {
		_, err := repo.GetByCAIP2(ctx, "   ")
		require.Error(t, err)
	})

	t.Run("fallback by reference", func(t *testing.T) {
		chain, err := repo.GetByCAIP2(ctx, "eip155:8453")
		require.NoError(t, err)
		require.NotNil(t, chain)
		require.Equal(t, "8453", chain.ChainID)
	})

	t.Run("not found malformed caip2", func(t *testing.T) {
		_, err := repo.GetByCAIP2(ctx, "999999")
		require.Error(t, err)
	})

	t.Run("not found valid caip2", func(t *testing.T) {
		_, err := repo.GetByCAIP2(ctx, "eip155:999999")
		require.Error(t, err)
	})
}
