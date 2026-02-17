package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestCrosschainConfigUsecase_Preflight_ErrorBranches(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}

	t.Run("invalid destination input", func(t *testing.T) {
		chainRepo := &ccChainRepoStub{
			byID:    map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
			byChain: map[string]*entities.Chain{"8453": source},
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source},
		}
		u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{})
		_, err := u.Preflight(context.Background(), "eip155:8453", "invalid-dest")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid input")
	})

	t.Run("source chain load by id error", func(t *testing.T) {
		chainRepo := &ccChainRepoStub{
			byID:    map[uuid.UUID]*entities.Chain{destID: dest},
			byChain: map[string]*entities.Chain{"8453": source, "42161": dest},
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
		}
		u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{})
		_, err := u.Preflight(context.Background(), "eip155:8453", "eip155:42161")
		require.Error(t, err)
	})

	t.Run("destination chain load by id error", func(t *testing.T) {
		chainRepo := &ccChainRepoStub{
			byID:    map[uuid.UUID]*entities.Chain{sourceID: source},
			byChain: map[string]*entities.Chain{"8453": source, "42161": dest},
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
		}
		u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{})
		_, err := u.Preflight(context.Background(), "eip155:8453", "eip155:42161")
		require.Error(t, err)
	})

	t.Run("adapter status fails on second call", func(t *testing.T) {
		chainRepo := &ccChainRepoStub{
			byID:    map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
			byChain: map[string]*entities.Chain{"8453": source, "42161": dest},
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
		}

		callCount := 0
		adapter := &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				callCount++
				if callCount == 1 {
					return &OnchainAdapterStatus{
						DefaultBridgeType:   0,
						HasAdapterDefault:   true,
						AdapterDefaultType:  "0x1111111111111111111111111111111111111111",
						HasAdapterType0:     true,
						AdapterType0:        "0x1111111111111111111111111111111111111111",
						HyperbridgeConfigured: true,
					}, nil
				}
				return nil, errors.New("status second call failed")
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{byChain: map[uuid.UUID][]*entities.Token{}}, &ccContractRepoStub{}, nil, adapter)
		_, err := u.Preflight(context.Background(), "eip155:8453", "eip155:42161")
		require.Error(t, err)
		require.Contains(t, err.Error(), "status second call failed")
		require.Equal(t, 2, callCount)
	})
}
