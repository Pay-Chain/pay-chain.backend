package usecases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/repositories"
)

type ChainResolver struct {
	chainRepo repositories.ChainRepository
}

func NewChainResolver(chainRepo repositories.ChainRepository) *ChainResolver {
	return &ChainResolver{
		chainRepo: chainRepo,
	}
}

// ResolveFromAny takes a chain identifier (UUID string or CAIP-2 string)
// and returns the internal UUID and the canonical CAIP-2 string.
func (r *ChainResolver) ResolveFromAny(ctx context.Context, input string) (uuid.UUID, string, error) {
	if input == "" {
		return uuid.Nil, "", fmt.Errorf("chain identifier cannot be empty")
	}

	// 1. Try to parse as UUID
	if id, err := uuid.Parse(input); err == nil {
		chain, err := r.chainRepo.GetByID(ctx, id)
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("failed to get chain by ID %s: %w", id, err)
		}
		return chain.ID, chain.GetCAIP2ID(), nil
	}

	// 2. Try direct lookup by full input first (covers DB that stores CAIP-2).
	chain, err := r.chainRepo.GetByChainID(ctx, input)
	if err == nil {
		return chain.ID, chain.GetCAIP2ID(), nil
	}

	// 3. Try to look up by raw blockchain ChainID (e.g. "84532" or "devnet")
	// when the input is CAIP-2.

	rawID := input
	// Handle CAIP-2 format "eip155:84532" or "solana:devnet"
	// Note: for solana, the ref might be "devnet"
	if len(input) > 7 && input[0:7] == "eip155:" {
		rawID = input[7:]
	} else if len(input) > 7 && input[0:7] == "solana:" {
		rawID = input[7:]
	}

	chain, err = r.chainRepo.GetByChainID(ctx, rawID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to find chain for %s: %w", input, err)
	}

	return chain.ID, chain.GetCAIP2ID(), nil
}
