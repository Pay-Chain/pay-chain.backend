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

func TestCrosschainConfigUsecase_Overview_GetByIDErrorBranches(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()

	// source resolver success, but GetByID fails due missing byID entry
	sourceRef := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}

	chainRepo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{
			destID: dest,
		},
		byChain: map[string]*entities.Chain{
			"8453":  sourceRef,
			"42161": dest,
		},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  sourceRef,
			"eip155:42161": dest,
		},
		allChain: []*entities.Chain{sourceRef, dest},
	}

	u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{})
	_, err := u.Overview(context.Background(), "eip155:8453", "", utils.PaginationParams{Page: 1, Limit: 20})
	require.Error(t, err)

	// destination resolver success, but GetByID fails due missing byID entry.
	chainRepo2 := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{
			sourceID: sourceRef,
		},
		byChain: map[string]*entities.Chain{
			"8453":  sourceRef,
			"42161": dest,
		},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  sourceRef,
			"eip155:42161": dest,
		},
		allChain: []*entities.Chain{sourceRef, dest},
	}
	u2 := NewCrosschainConfigUsecase(chainRepo2, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{})
	_, err = u2.Overview(context.Background(), "", "eip155:42161", utils.PaginationParams{Page: 1, Limit: 20})
	require.Error(t, err)
}

func TestCrosschainConfigUsecase_Overview_RecheckErrorRowBranch(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()

	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}

	chainRepo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{
			sourceID: source,
			destID:   dest,
		},
		byChain: map[string]*entities.Chain{
			"8453":  source,
			"42161": dest,
		},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  source,
			"eip155:42161": dest,
		},
		allChain: []*entities.Chain{source, dest},
	}

	adapter := &crosschainAdapterStub{
		statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
			return nil, errors.New("status failed")
		},
	}

	u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, adapter)
	out, err := u.Overview(context.Background(), "", "", utils.PaginationParams{Page: 1, Limit: 20})
	require.NoError(t, err)
	require.Len(t, out.Items, 2)
	for _, item := range out.Items {
		require.Equal(t, "ERROR", item.OverallStatus)
		require.NotEmpty(t, item.Issues)
		require.Equal(t, "RECHECK_FAILED", item.Issues[0].Code)
	}
}
