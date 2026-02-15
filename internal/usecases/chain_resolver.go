package usecases

import (
	"context"
	"fmt"
	"strings"

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
	value := strings.TrimSpace(input)

	// 1. Try to parse as UUID
	if id, err := uuid.Parse(value); err == nil {
		chain, err := r.chainRepo.GetByID(ctx, id)
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("failed to get chain by ID %s: %w", id, err)
		}
		return chain.ID, chain.GetCAIP2ID(), nil
	}

	// 2. Try direct lookup by full input first (covers DB that stores CAIP-2).
	// 2. Handle CAIP-2 explicitly first to avoid noisy misses on chain_id lookup.
	if strings.Contains(value, ":") {
		if chain, err := r.chainRepo.GetByCAIP2(ctx, value); err == nil {
			return chain.ID, chain.GetCAIP2ID(), nil
		}
	}

	// 3. Try direct lookup by raw chain_id
	chain, err := r.chainRepo.GetByChainID(ctx, value)
	if err == nil {
		return chain.ID, chain.GetCAIP2ID(), nil
	}

	// 4. Try to look up by raw blockchain ChainID (e.g. "84532" or "devnet")
	// when the input is CAIP-2.

	rawID := value
	// Handle CAIP-2 format "eip155:84532" or "solana:devnet"
	// Note: for solana, the ref might be "devnet"
	if len(value) > 7 && value[0:7] == "eip155:" {
		rawID = value[7:]
	} else if len(value) > 7 && value[0:7] == "solana:" {
		rawID = value[7:]
	}

	chain, err = r.chainRepo.GetByChainID(ctx, rawID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to find chain for %s: %w", value, err)
	}

	return chain.ID, chain.GetCAIP2ID(), nil
}
