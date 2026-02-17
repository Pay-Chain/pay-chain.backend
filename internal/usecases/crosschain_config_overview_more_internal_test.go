package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

func TestCrosschainConfigUsecase_Overview_Paths(t *testing.T) {
	sourceID := uuid.New()
	destEvmID := uuid.New()
	destSvmID := uuid.New()

	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true}
	destEvm := &entities.Chain{ID: destEvmID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}
	destSvm := &entities.Chain{ID: destSvmID, ChainID: "mainnet", Name: "Solana", Type: entities.ChainTypeSVM, IsActive: true}
	inactiveEvm := &entities.Chain{ID: uuid.New(), ChainID: "10", Name: "Optimism", Type: entities.ChainTypeEVM, IsActive: false}

	chainRepo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{
			sourceID:  source,
			destEvmID: destEvm,
			destSvmID: destSvm,
		},
		byChain: map[string]*entities.Chain{
			"8453":    source,
			"42161":   destEvm,
			"mainnet": destSvm,
		},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":    source,
			"eip155:42161":   destEvm,
			"solana:mainnet": destSvm,
		},
		allChain: []*entities.Chain{source, destEvm, destSvm, inactiveEvm},
	}

	adapter := &crosschainAdapterStub{
		statusFn: func(_ context.Context, _, destInput string) (*OnchainAdapterStatus, error) {
			if destInput == "solana:mainnet" {
				return nil, errors.New("simulated status failure")
			}
			return &OnchainAdapterStatus{
				DefaultBridgeType:      0,
				HasAdapterDefault:      true,
				AdapterDefaultType:     "0x1111111111111111111111111111111111111111",
				HasAdapterType0:        true,
				AdapterType0:           "0x1111111111111111111111111111111111111111",
				HyperbridgeConfigured:  true,
				CCIPChainSelector:      0,
				CCIPDestinationAdapter: "0x",
				LayerZeroConfigured:    false,
			}, nil
		},
	}

	u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, adapter)

	// Source filter path + mix of READY/ERROR route rows + pagination slicing.
	out, err := u.Overview(context.Background(), "eip155:8453", "", utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, int64(2), out.Meta.TotalCount)
	require.Len(t, out.Items, 2)

	foundError := false
	for _, item := range out.Items {
		if item.OverallStatus == "ERROR" {
			foundError = true
		}
	}
	require.True(t, foundError)

	// Offset > len branch.
	out, err = u.Overview(context.Background(), "eip155:8453", "", utils.PaginationParams{Page: 4, Limit: 10})
	require.NoError(t, err)
	require.Len(t, out.Items, 0)

	// Limit <= 0 branch returns all.
	out, err = u.Overview(context.Background(), "eip155:8453", "", utils.PaginationParams{Page: 1, Limit: 0})
	require.NoError(t, err)
	require.Len(t, out.Items, 2)
}
