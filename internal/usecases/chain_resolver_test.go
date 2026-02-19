package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
)

func TestChainResolver_ResolveFromAny_EmptyInput(t *testing.T) {
	resolver := usecases.NewChainResolver(new(MockChainRepository))

	id, caip2, err := resolver.ResolveFromAny(context.Background(), "")
	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, id)
	assert.Equal(t, "", caip2)
}

func TestChainResolver_ResolveFromAny_UUIDSuccess(t *testing.T) {
	mockRepo := new(MockChainRepository)
	resolver := usecases.NewChainResolver(mockRepo)

	chainID := uuid.New()
	chain := &entities.Chain{
		ID:      chainID,
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}
	mockRepo.On("GetByID", context.Background(), chainID).Return(chain, nil).Once()

	id, caip2, err := resolver.ResolveFromAny(context.Background(), chainID.String())
	assert.NoError(t, err)
	assert.Equal(t, chainID, id)
	assert.Equal(t, "eip155:8453", caip2)
	mockRepo.AssertExpectations(t)
}

func TestChainResolver_ResolveFromAny_CAIP2LookupSuccess(t *testing.T) {
	mockRepo := new(MockChainRepository)
	resolver := usecases.NewChainResolver(mockRepo)

	chain := &entities.Chain{
		ID:      uuid.New(),
		Type:    entities.ChainTypeEVM,
		ChainID: "42161",
	}
	input := "eip155:42161"
	mockRepo.On("GetByCAIP2", context.Background(), input).Return(chain, nil).Once()

	id, caip2, err := resolver.ResolveFromAny(context.Background(), input)
	assert.NoError(t, err)
	assert.Equal(t, chain.ID, id)
	assert.Equal(t, "eip155:42161", caip2)
	mockRepo.AssertExpectations(t)
}

func TestChainResolver_ResolveFromAny_RawChainIDSuccess(t *testing.T) {
	mockRepo := new(MockChainRepository)
	resolver := usecases.NewChainResolver(mockRepo)

	chain := &entities.Chain{
		ID:      uuid.New(),
		Type:    entities.ChainTypeSVM,
		ChainID: "devnet",
	}

	mockRepo.On("GetByChainID", context.Background(), "devnet").Return(chain, nil).Once()

	id, caip2, err := resolver.ResolveFromAny(context.Background(), "devnet")
	assert.NoError(t, err)
	assert.Equal(t, chain.ID, id)
	assert.Equal(t, "solana:devnet", caip2)
	mockRepo.AssertExpectations(t)
}

func TestChainResolver_ResolveFromAny_CAIP2FallbackToRawID(t *testing.T) {
	mockRepo := new(MockChainRepository)
	resolver := usecases.NewChainResolver(mockRepo)

	chain := &entities.Chain{
		ID:      uuid.New(),
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}

	caip2 := "eip155:8453"
	mockRepo.On("GetByCAIP2", context.Background(), caip2).Return(nil, errors.New("not found")).Once()

	mockRepo.On("GetByChainID", context.Background(), "8453").Return(chain, nil).Once()

	id, resolved, err := resolver.ResolveFromAny(context.Background(), caip2)
	assert.NoError(t, err)
	assert.Equal(t, chain.ID, id)
	assert.Equal(t, "eip155:8453", resolved)
	mockRepo.AssertExpectations(t)
}

func TestChainResolver_ResolveFromAny_Failure(t *testing.T) {
	mockRepo := new(MockChainRepository)
	resolver := usecases.NewChainResolver(mockRepo)

	mockRepo.On("GetByChainID", context.Background(), "unknown-chain").Return(nil, errors.New("not found")).Once()

	id, caip2, err := resolver.ResolveFromAny(context.Background(), "unknown-chain")
	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, id)
	assert.Equal(t, "", caip2)
	mockRepo.AssertExpectations(t)
}

func TestChainResolver_ResolveFromAny_UUIDLookupError(t *testing.T) {
	mockRepo := new(MockChainRepository)
	resolver := usecases.NewChainResolver(mockRepo)

	chainID := uuid.New()
	mockRepo.On("GetByID", context.Background(), chainID).Return(nil, errors.New("db down")).Once()

	id, caip2, err := resolver.ResolveFromAny(context.Background(), chainID.String())
	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, id)
	assert.Equal(t, "", caip2)
	assert.Contains(t, err.Error(), "failed to get chain by ID")
	mockRepo.AssertExpectations(t)
}

func TestChainResolver_ResolveFromAny_TrimmedInputAndSolanaFallback(t *testing.T) {
	mockRepo := new(MockChainRepository)
	resolver := usecases.NewChainResolver(mockRepo)

	chain := &entities.Chain{
		ID:      uuid.New(),
		Type:    entities.ChainTypeSVM,
		ChainID: "devnet",
	}

	caip2 := "solana:devnet"
	mockRepo.On("GetByCAIP2", context.Background(), caip2).Return(nil, errors.New("not found")).Once()

	mockRepo.On("GetByChainID", context.Background(), "devnet").Return(chain, nil).Once()

	id, resolved, err := resolver.ResolveFromAny(context.Background(), "  "+caip2+"  ")
	assert.NoError(t, err)
	assert.Equal(t, chain.ID, id)
	assert.Equal(t, "solana:devnet", resolved)
	mockRepo.AssertExpectations(t)
}
