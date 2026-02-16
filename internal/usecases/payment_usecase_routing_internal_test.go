package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

type routePolicyRepoStub struct {
	getByRouteFn func(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.RoutePolicy, error)
}

func (s *routePolicyRepoStub) GetByID(context.Context, uuid.UUID) (*entities.RoutePolicy, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *routePolicyRepoStub) GetByRoute(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.RoutePolicy, error) {
	if s.getByRouteFn != nil {
		return s.getByRouteFn(ctx, sourceChainID, destChainID)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *routePolicyRepoStub) List(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.RoutePolicy, int64, error) {
	return nil, 0, nil
}
func (s *routePolicyRepoStub) Create(context.Context, *entities.RoutePolicy) error { return nil }
func (s *routePolicyRepoStub) Update(context.Context, *entities.RoutePolicy) error { return nil }
func (s *routePolicyRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }

type bridgeConfigRepoStub struct {
	getActiveFn func(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.BridgeConfig, error)
}

func (s *bridgeConfigRepoStub) GetActive(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.BridgeConfig, error) {
	if s.getActiveFn != nil {
		return s.getActiveFn(ctx, sourceChainID, destChainID)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *bridgeConfigRepoStub) GetByID(context.Context, uuid.UUID) (*entities.BridgeConfig, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *bridgeConfigRepoStub) List(context.Context, *uuid.UUID, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.BridgeConfig, int64, error) {
	return nil, 0, nil
}
func (s *bridgeConfigRepoStub) Create(context.Context, *entities.BridgeConfig) error { return nil }
func (s *bridgeConfigRepoStub) Update(context.Context, *entities.BridgeConfig) error { return nil }
func (s *bridgeConfigRepoStub) Delete(context.Context, uuid.UUID) error              { return nil }

type tokenResolveRepoStub struct {
	getNativeFn    func(ctx context.Context, chainID uuid.UUID) (*entities.Token, error)
	getByAddressFn func(ctx context.Context, address string, chainID uuid.UUID) (*entities.Token, error)
}

func (s *tokenResolveRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *tokenResolveRepoStub) GetBySymbol(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *tokenResolveRepoStub) GetByAddress(ctx context.Context, address string, chainID uuid.UUID) (*entities.Token, error) {
	if s.getByAddressFn != nil {
		return s.getByAddressFn(ctx, address, chainID)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *tokenResolveRepoStub) GetAll(context.Context) ([]*entities.Token, error) {
	return nil, nil
}
func (s *tokenResolveRepoStub) GetStablecoins(context.Context) ([]*entities.Token, error) {
	return nil, nil
}
func (s *tokenResolveRepoStub) GetNative(ctx context.Context, chainID uuid.UUID) (*entities.Token, error) {
	if s.getNativeFn != nil {
		return s.getNativeFn(ctx, chainID)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *tokenResolveRepoStub) GetTokensByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (s *tokenResolveRepoStub) GetAllTokens(context.Context, *uuid.UUID, *string, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (s *tokenResolveRepoStub) Create(context.Context, *entities.Token) error { return nil }
func (s *tokenResolveRepoStub) Update(context.Context, *entities.Token) error { return nil }
func (s *tokenResolveRepoStub) SoftDelete(context.Context, uuid.UUID) error   { return nil }

func TestPaymentUsecase_DecideBridge_Priority(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()

	t.Run("route policy has highest priority", func(t *testing.T) {
		u := &PaymentUsecase{
			routePolicyRepo: &routePolicyRepoStub{
				getByRouteFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
					return &entities.RoutePolicy{DefaultBridgeType: 2}, nil
				},
			},
			bridgeConfigRepo: &bridgeConfigRepoStub{
				getActiveFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.BridgeConfig, error) {
					return &entities.BridgeConfig{
						Bridge: &entities.PaymentBridge{ID: uuid.New(), Name: "Hyperbridge"},
					}, nil
				},
			},
		}
		bridgeName, bridgeID := u.decideBridge(context.Background(), sourceID, destID, "eip155:8453", "eip155:42161")
		require.Equal(t, "LayerZero", bridgeName)
		require.Nil(t, bridgeID)
	})

	t.Run("bridge config used when policy missing", func(t *testing.T) {
		cfgBridgeID := uuid.New()
		u := &PaymentUsecase{
			routePolicyRepo: &routePolicyRepoStub{
				getByRouteFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
					return nil, errors.New("missing policy")
				},
			},
			bridgeConfigRepo: &bridgeConfigRepoStub{
				getActiveFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.BridgeConfig, error) {
					return &entities.BridgeConfig{
						Bridge: &entities.PaymentBridge{ID: cfgBridgeID, Name: "Hyperbridge"},
					}, nil
				},
			},
		}
		bridgeName, bridgeID := u.decideBridge(context.Background(), sourceID, destID, "eip155:8453", "eip155:42161")
		require.Equal(t, "Hyperbridge", bridgeName)
		require.NotNil(t, bridgeID)
		require.Equal(t, cfgBridgeID, *bridgeID)
	})

	t.Run("fallback deterministic selection", func(t *testing.T) {
		u := &PaymentUsecase{
			routePolicyRepo: &routePolicyRepoStub{
				getByRouteFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
					return nil, errors.New("missing policy")
				},
			},
			bridgeConfigRepo: &bridgeConfigRepoStub{
				getActiveFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.BridgeConfig, error) {
					return nil, errors.New("missing config")
				},
			},
		}
		bridgeName, bridgeID := u.decideBridge(context.Background(), sourceID, destID, "eip155:8453", "eip155:42161")
		require.Equal(t, "CCIP", bridgeName)
		require.Nil(t, bridgeID)
	})
}

func TestPaymentUsecase_ResolveBridgeOrder_WithPolicy(t *testing.T) {
	u := &PaymentUsecase{
		routePolicyRepo: &routePolicyRepoStub{
			getByRouteFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
				return &entities.RoutePolicy{
					DefaultBridgeType: 1,
					FallbackMode:      entities.BridgeFallbackModeAutoFallback,
					FallbackOrder:     []uint8{2, 0, 1, 9},
				}, nil
			},
		},
	}
	order := u.resolveBridgeOrder(context.Background(), uuid.New(), uuid.New(), "eip155:8453", "eip155:42161")
	require.Equal(t, []uint8{1, 2, 0}, order)
}

func TestPaymentUsecase_ResolveToken_Branches(t *testing.T) {
	chainID := uuid.New()

	t.Run("native token success", func(t *testing.T) {
		u := &PaymentUsecase{
			tokenRepo: &tokenResolveRepoStub{
				getNativeFn: func(context.Context, uuid.UUID) (*entities.Token, error) {
					return &entities.Token{Symbol: "ETH"}, nil
				},
			},
		}
		token, err := u.resolveToken(context.Background(), "native", chainID)
		require.NoError(t, err)
		require.Equal(t, "ETH", token.Symbol)
	})

	t.Run("native token not found", func(t *testing.T) {
		u := &PaymentUsecase{
			tokenRepo: &tokenResolveRepoStub{
				getNativeFn: func(context.Context, uuid.UUID) (*entities.Token, error) {
					return nil, errors.New("not found")
				},
			},
		}
		_, err := u.resolveToken(context.Background(), "", chainID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "native token not found")
	})

	t.Run("erc20 address success", func(t *testing.T) {
		u := &PaymentUsecase{
			tokenRepo: &tokenResolveRepoStub{
				getByAddressFn: func(context.Context, string, uuid.UUID) (*entities.Token, error) {
					return &entities.Token{Symbol: "USDC"}, nil
				},
			},
		}
		token, err := u.resolveToken(context.Background(), "0x1111111111111111111111111111111111111111", chainID)
		require.NoError(t, err)
		require.Equal(t, "USDC", token.Symbol)
	})

	t.Run("erc20 address not found", func(t *testing.T) {
		u := &PaymentUsecase{
			tokenRepo: &tokenResolveRepoStub{
				getByAddressFn: func(context.Context, string, uuid.UUID) (*entities.Token, error) {
					return nil, errors.New("missing token")
				},
			},
		}
		_, err := u.resolveToken(context.Background(), "0x2222222222222222222222222222222222222222", chainID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "token not found for address")
	})
}

func TestPaymentUsecase_SelectBridge_Branches(t *testing.T) {
	u := &PaymentUsecase{}

	t.Run("evm to evm selects ccip", func(t *testing.T) {
		got := u.SelectBridge("eip155:8453", "eip155:42161")
		require.Equal(t, "CCIP", got)
	})

	t.Run("solana to evm selects hyperlane", func(t *testing.T) {
		got := u.SelectBridge("solana:mainnet", "eip155:8453")
		require.Equal(t, "Hyperlane", got)
	})

	t.Run("evm to solana selects hyperlane", func(t *testing.T) {
		got := u.SelectBridge("eip155:8453", "solana:mainnet")
		require.Equal(t, "Hyperlane", got)
	})

	t.Run("non-evm non-solana falls back to hyperbridge", func(t *testing.T) {
		got := u.SelectBridge("cosmos:osmosis-1", "cosmos:cosmoshub-4")
		require.Equal(t, "Hyperbridge", got)
	})
}
