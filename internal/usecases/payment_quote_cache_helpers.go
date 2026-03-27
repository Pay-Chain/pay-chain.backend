package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/infrastructure/blockchain"
)

func (u *PaymentUsecase) getCachedActiveContract(
	ctx context.Context,
	chainID uuid.UUID,
	contractType entities.SmartContractType,
) (*entities.SmartContract, error) {
	cache := getQuoteRequestCache(ctx)
	key := contractCacheKey(chainID, contractType)
	if cache != nil {
		cache.mu.RLock()
		if cached, ok := cache.activeContracts[key]; ok {
			cache.mu.RUnlock()
			return cached, nil
		}
		cache.mu.RUnlock()
	}

	contract, err := u.contractRepo.GetActiveContract(ctx, chainID, contractType)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		cache.mu.Lock()
		cache.activeContracts[key] = contract
		cache.mu.Unlock()
	}
	return contract, nil
}

func (u *PaymentUsecase) getCachedResolvedABI(
	ctx context.Context,
	chainID uuid.UUID,
	contractType entities.SmartContractType,
) (abi.ABI, error) {
	cache := getQuoteRequestCache(ctx)
	key := contractCacheKey(chainID, contractType)
	if cache != nil {
		cache.mu.RLock()
		if cached, ok := cache.resolvedABIs[key]; ok {
			cache.mu.RUnlock()
			return cached, nil
		}
		cache.mu.RUnlock()
	}

	resolved, err := u.ResolveABIWithFallback(ctx, chainID, contractType)
	if err != nil {
		return abi.ABI{}, err
	}
	if cache != nil {
		cache.mu.Lock()
		cache.resolvedABIs[key] = resolved
		cache.mu.Unlock()
	}
	return resolved, nil
}

func (u *PaymentUsecase) getCachedRoutePath(
	ctx context.Context,
	client *blockchain.EVMClient,
	chainID uuid.UUID,
	swapperAddress string,
	swapperABI abi.ABI,
	tokenIn, tokenOut string,
) ([]string, error) {
	cache := getQuoteRequestCache(ctx)
	key := routePathCacheKey(chainID, swapperAddress, tokenIn, tokenOut)
	if cache != nil {
		cache.mu.RLock()
		if cached, ok := cache.routePaths[key]; ok {
			cache.mu.RUnlock()
			return cloneStringSlice(cached), nil
		}
		cache.mu.RUnlock()
	}

	routePath, err := readRoutePath(ctx, client, swapperAddress, swapperABI, tokenIn, tokenOut)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		cache.mu.Lock()
		cache.routePaths[key] = cloneStringSlice(routePath)
		cache.mu.Unlock()
	}
	return routePath, nil
}

func (u *PaymentUsecase) getCachedQuoterV3(
	ctx context.Context,
	client *blockchain.EVMClient,
	chainID uuid.UUID,
	chain *entities.Chain,
	swapperAddress string,
) (common.Address, error) {
	cache := getQuoteRequestCache(ctx)
	key := quoterCacheKey(chainID, swapperAddress)
	if cache != nil {
		cache.mu.RLock()
		if cached, ok := cache.quoterAddresses[key]; ok && strings.TrimSpace(cached) != "" {
			cache.mu.RUnlock()
			return common.HexToAddress(cached), nil
		}
		cache.mu.RUnlock()
	}

	quoterV3, err := callAddressBySignature(ctx, client, swapperAddress, "quoterV3()")
	if err != nil {
		return common.Address{}, err
	}
	if quoterV3 == (common.Address{}) {
		if fallback := fallbackKnownUniswapV3Quoter(chain); fallback != "" {
			quoterV3 = common.HexToAddress(fallback)
		} else {
			return common.Address{}, fmt.Errorf("accurate quote unavailable because quoterV3 is not configured")
		}
	}

	if cache != nil {
		cache.mu.Lock()
		cache.quoterAddresses[key] = quoterV3.Hex()
		cache.mu.Unlock()
	}
	return quoterV3, nil
}

func (u *PaymentUsecase) getCachedV3PoolConfig(
	ctx context.Context,
	client *blockchain.EVMClient,
	chainID uuid.UUID,
	swapperAddress, tokenIn, tokenOut string,
) (bool, uint32, error) {
	cache := getQuoteRequestCache(ctx)
	key := v3PoolCacheKey(chainID, swapperAddress, tokenIn, tokenOut)
	if cache != nil {
		cache.mu.RLock()
		if cached, ok := cache.v3PoolConfigs[key]; ok {
			cache.mu.RUnlock()
			return cached.active, cached.feeTier, nil
		}
		cache.mu.RUnlock()
	}

	active, feeTier, err := readV3PoolConfig(ctx, client, swapperAddress, tokenIn, tokenOut)
	if err != nil {
		return false, 0, err
	}
	if cache != nil {
		cache.mu.Lock()
		cache.v3PoolConfigs[key] = cachedV3PoolConfig{
			active:  active,
			feeTier: feeTier,
		}
		cache.mu.Unlock()
	}
	return active, feeTier, nil
}
