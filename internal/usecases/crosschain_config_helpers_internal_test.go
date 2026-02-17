package usecases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

type ccfgChainRepoStub struct {
	getByCAIP2Fn func(ctx context.Context, caip2 string) (*entities.Chain, error)
}

func (s *ccfgChainRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccfgChainRepoStub) GetByChainID(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccfgChainRepoStub) GetByCAIP2(ctx context.Context, caip2 string) (*entities.Chain, error) {
	if s.getByCAIP2Fn != nil {
		return s.getByCAIP2Fn(ctx, caip2)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *ccfgChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) { return nil, nil }
func (s *ccfgChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *ccfgChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *ccfgChainRepoStub) Create(context.Context, *entities.Chain) error { return nil }
func (s *ccfgChainRepoStub) Update(context.Context, *entities.Chain) error { return nil }
func (s *ccfgChainRepoStub) Delete(context.Context, uuid.UUID) error       { return nil }

type ccfgContractRepoStub struct {
	getActiveFn func(ctx context.Context, chainID uuid.UUID, t entities.SmartContractType) (*entities.SmartContract, error)
}

func (s *ccfgContractRepoStub) Create(context.Context, *entities.SmartContract) error { return nil }
func (s *ccfgContractRepoStub) GetByID(context.Context, uuid.UUID) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccfgContractRepoStub) GetByChainAndAddress(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccfgContractRepoStub) GetActiveContract(ctx context.Context, chainID uuid.UUID, t entities.SmartContractType) (*entities.SmartContract, error) {
	if s.getActiveFn != nil {
		return s.getActiveFn(ctx, chainID, t)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *ccfgContractRepoStub) GetFiltered(context.Context, *uuid.UUID, entities.SmartContractType, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *ccfgContractRepoStub) GetByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *ccfgContractRepoStub) GetAll(context.Context, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *ccfgContractRepoStub) Update(context.Context, *entities.SmartContract) error { return nil }
func (s *ccfgContractRepoStub) SoftDelete(context.Context, uuid.UUID) error           { return nil }

func TestCrosschainConfigUsecase_DeriveDestinationContractHex(t *testing.T) {
	destUUID := uuid.New()
	chainRepo := &ccfgChainRepoStub{
		getByCAIP2Fn: func(_ context.Context, caip2 string) (*entities.Chain, error) {
			if caip2 == "eip155:42161" {
				return &entities.Chain{ID: destUUID, ChainID: "42161", Type: entities.ChainTypeEVM}, nil
			}
			return nil, domainerrors.ErrNotFound
		},
	}
	contractRepo := &ccfgContractRepoStub{
		getActiveFn: func(_ context.Context, chainID uuid.UUID, contractType entities.SmartContractType) (*entities.SmartContract, error) {
			require.Equal(t, entities.ContractTypeAdapterHyperbridge, contractType)
			require.Equal(t, destUUID, chainID)
			return &entities.SmartContract{ContractAddress: "0x000000000000000000000000000000000000dEaD"}, nil
		},
	}

	u := &CrosschainConfigUsecase{
		chainRepo:     chainRepo,
		contractRepo:  contractRepo,
		chainResolver: NewChainResolver(chainRepo),
	}
	hex, err := u.deriveDestinationContractHex(context.Background(), "eip155:42161", entities.ContractTypeAdapterHyperbridge)
	require.NoError(t, err)
	require.Equal(t, "0x000000000000000000000000000000000000000000000000000000000000dead", hex)
}

func TestCrosschainConfigUsecase_DeriveDestinationContractHex_NotFound(t *testing.T) {
	destUUID := uuid.New()
	chainRepo := &ccfgChainRepoStub{
		getByCAIP2Fn: func(_ context.Context, caip2 string) (*entities.Chain, error) {
			return &entities.Chain{ID: destUUID, ChainID: "42161", Type: entities.ChainTypeEVM}, nil
		},
	}
	contractRepo := &ccfgContractRepoStub{
		getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
			return nil, domainerrors.ErrNotFound
		},
	}
	u := &CrosschainConfigUsecase{
		chainRepo:     chainRepo,
		contractRepo:  contractRepo,
		chainResolver: NewChainResolver(chainRepo),
	}
	_, err := u.deriveDestinationContractHex(context.Background(), "eip155:42161", entities.ContractTypeAdapterHyperbridge)
	require.Error(t, err)
	require.Contains(t, err.Error(), "active destination contract")
}

func TestCrosschainConfigUsecase_DeriveDestinationContractHex_InvalidDestInput(t *testing.T) {
	chainRepo := &ccfgChainRepoStub{
		getByCAIP2Fn: func(_ context.Context, caip2 string) (*entities.Chain, error) {
			return nil, domainerrors.ErrNotFound
		},
	}
	u := &CrosschainConfigUsecase{
		chainRepo:     chainRepo,
		contractRepo:  &ccfgContractRepoStub{},
		chainResolver: NewChainResolver(chainRepo),
	}
	_, err := u.deriveDestinationContractHex(context.Background(), "bad-dest", entities.ContractTypeAdapterHyperbridge)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestCrosschainConfigUsecase_DeriveDestinationContractHex_InvalidContractAddress(t *testing.T) {
	destUUID := uuid.New()
	chainRepo := &ccfgChainRepoStub{
		getByCAIP2Fn: func(_ context.Context, caip2 string) (*entities.Chain, error) {
			return &entities.Chain{ID: destUUID, ChainID: "42161", Type: entities.ChainTypeEVM}, nil
		},
	}
	contractRepo := &ccfgContractRepoStub{
		getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
			return &entities.SmartContract{ContractAddress: "not-hex-address"}, nil
		},
	}
	u := &CrosschainConfigUsecase{
		chainRepo:     chainRepo,
		contractRepo:  contractRepo,
		chainResolver: NewChainResolver(chainRepo),
	}
	_, err := u.deriveDestinationContractHex(context.Background(), "eip155:42161", entities.ContractTypeAdapterHyperbridge)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid hex address")
}

func TestCrosschainConfigUsecase_CheckFeeQuoteHealth_Guards(t *testing.T) {
	u := &CrosschainConfigUsecase{}
	require.False(t, u.checkFeeQuoteHealth(context.Background(), nil, nil, 0))
}
