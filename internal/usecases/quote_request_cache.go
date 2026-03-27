package usecases

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
)

type quoteRequestCacheKeyType struct{}

var quoteRequestCacheKey = quoteRequestCacheKeyType{}

type quoteRequestCache struct {
	mu sync.RWMutex

	activeContracts map[string]*entities.SmartContract
	resolvedABIs    map[string]abi.ABI
	routePaths      map[string][]string
	quoterAddresses map[string]string
	v3PoolConfigs   map[string]cachedV3PoolConfig
	routeSupports   map[string]*TokenRouteSupportStatus
	chainsByID      map[string]*entities.Chain
	tokensByAddress map[string]*entities.Token
	tokensBySymbol  map[string]*entities.Token
}

type cachedV3PoolConfig struct {
	active  bool
	feeTier uint32
}

func newQuoteRequestCache() *quoteRequestCache {
	return &quoteRequestCache{
		activeContracts: make(map[string]*entities.SmartContract),
		resolvedABIs:    make(map[string]abi.ABI),
		routePaths:      make(map[string][]string),
		quoterAddresses: make(map[string]string),
		v3PoolConfigs:   make(map[string]cachedV3PoolConfig),
		routeSupports:   make(map[string]*TokenRouteSupportStatus),
		chainsByID:      make(map[string]*entities.Chain),
		tokensByAddress: make(map[string]*entities.Token),
		tokensBySymbol:  make(map[string]*entities.Token),
	}
}

func withQuoteRequestCache(ctx context.Context) context.Context {
	if ctx == nil {
		return context.WithValue(context.Background(), quoteRequestCacheKey, newQuoteRequestCache())
	}
	if _, ok := ctx.Value(quoteRequestCacheKey).(*quoteRequestCache); ok {
		return ctx
	}
	return context.WithValue(ctx, quoteRequestCacheKey, newQuoteRequestCache())
}

func getQuoteRequestCache(ctx context.Context) *quoteRequestCache {
	if ctx == nil {
		return nil
	}
	cache, _ := ctx.Value(quoteRequestCacheKey).(*quoteRequestCache)
	return cache
}

func contractCacheKey(chainID uuid.UUID, contractType entities.SmartContractType) string {
	return fmt.Sprintf("%s:%s", chainID.String(), strings.TrimSpace(string(contractType)))
}

func routePathCacheKey(chainID uuid.UUID, swapperAddress, tokenIn, tokenOut string) string {
	return fmt.Sprintf(
		"%s:%s:%s:%s",
		chainID.String(),
		normalizeEvmAddress(swapperAddress),
		normalizeEvmAddress(tokenIn),
		normalizeEvmAddress(tokenOut),
	)
}

func quoterCacheKey(chainID uuid.UUID, swapperAddress string) string {
	return fmt.Sprintf("%s:%s", chainID.String(), normalizeEvmAddress(swapperAddress))
}

func v3PoolCacheKey(chainID uuid.UUID, swapperAddress, tokenIn, tokenOut string) string {
	return fmt.Sprintf(
		"%s:%s:%s:%s",
		chainID.String(),
		normalizeEvmAddress(swapperAddress),
		normalizeEvmAddress(tokenIn),
		normalizeEvmAddress(tokenOut),
	)
}

func routeSupportCacheKey(chainID uuid.UUID, tokenIn, tokenOut string) string {
	return fmt.Sprintf(
		"%s:%s:%s",
		chainID.String(),
		normalizeEvmAddress(tokenIn),
		normalizeEvmAddress(tokenOut),
	)
}

func chainByIDCacheKey(chainID uuid.UUID) string {
	return chainID.String()
}

func tokenByAddressCacheKey(chainID uuid.UUID, tokenAddress string) string {
	return fmt.Sprintf("%s:%s", chainID.String(), normalizeEvmAddress(tokenAddress))
}

func tokenBySymbolCacheKey(chainID uuid.UUID, symbol string) string {
	return fmt.Sprintf("%s:%s", chainID.String(), strings.ToUpper(strings.TrimSpace(symbol)))
}

func cloneStringSlice(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, len(input))
	copy(out, input)
	return out
}

func cloneRouteSupportStatus(input *TokenRouteSupportStatus) *TokenRouteSupportStatus {
	if input == nil {
		return nil
	}
	return &TokenRouteSupportStatus{
		Exists:       input.Exists,
		IsDirect:     input.IsDirect,
		Path:         cloneStringSlice(input.Path),
		Executable:   input.Executable,
		Reasons:      cloneStringSlice(input.Reasons),
		SwapRouterV3: input.SwapRouterV3,
		UniversalV4:  input.UniversalV4,
	}
}
