package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
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

	// 2. Try direct lookup by full input via GetByCAIP2 (optimized for namespace:ref)
	if strings.Contains(value, ":") {
		if chain, err := r.chainRepo.GetByCAIP2(ctx, value); err == nil {
			return chain.ID, chain.GetCAIP2ID(), nil
		}
	}

	// 3. Try lookup by Normalized ID (which is now CAIP-2 preservation if colon exists)
	normalized := entities.NormalizeChainID(value)
	chain, err := r.chainRepo.GetByChainID(ctx, normalized)
	if err == nil {
		return chain.ID, chain.GetCAIP2ID(), nil
	}

	// 4. Legacy Fallback: Try stripping namespace (e.g. eip155:8453 -> 8453)
	// This supports DB rows that still store raw IDs.
	if strings.Contains(normalized, ":") {
		parts := strings.SplitN(normalized, ":", 2)
		if len(parts) == 2 {
			legacyID := parts[1]
			if legacyChain, err := r.chainRepo.GetByChainID(ctx, legacyID); err == nil {
				return legacyChain.ID, legacyChain.GetCAIP2ID(), nil
			}
		}
	}

	// 5. Final fallback with original input
	// Only try if it differs from normalized, otherwise we already tried it in step 3.
	if value != normalized {
		chain, err = r.chainRepo.GetByChainID(ctx, value)
		if err == nil {
			return chain.ID, chain.GetCAIP2ID(), nil
		}
	}

	return uuid.Nil, "", fmt.Errorf("failed to find chain for %s: %w", value, err)
}
