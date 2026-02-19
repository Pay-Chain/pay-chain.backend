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

type ccasChainRepoStub struct {
	byID map[uuid.UUID]*entities.Chain
	all  []*entities.Chain
}

func (s *ccasChainRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Chain, error) {
	if c, ok := s.byID[id]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}
func (s *ccasChainRepoStub) GetByChainID(context.Context, string) (*entities.Chain, error) {
	return nil, errors.New("not found")
}
func (s *ccasChainRepoStub) GetByCAIP2(context.Context, string) (*entities.Chain, error) {
	return nil, errors.New("not found")
}
func (s *ccasChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) { return s.all, nil }
func (s *ccasChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *ccasChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *ccasChainRepoStub) Create(context.Context, *entities.Chain) error       { return nil }
func (s *ccasChainRepoStub) Update(context.Context, *entities.Chain) error       { return nil }
func (s *ccasChainRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }
func (s *ccasChainRepoStub) CreateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *ccasChainRepoStub) UpdateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *ccasChainRepoStub) DeleteRPC(context.Context, uuid.UUID) error          { return nil }
func (s *ccasChainRepoStub) GetRPCByID(context.Context, uuid.UUID) (*entities.ChainRPC, error) {
	return nil, errors.New("not found")
}

type ccasContractRepoStub struct {
	contract *entities.SmartContract
	filtered []*entities.SmartContract
}

func (s *ccasContractRepoStub) Create(context.Context, *entities.SmartContract) error { return nil }
func (s *ccasContractRepoStub) GetByID(context.Context, uuid.UUID) (*entities.SmartContract, error) {
	return s.contract, nil
}
func (s *ccasContractRepoStub) GetByChainAndAddress(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
	return nil, errors.New("not found")
}
func (s *ccasContractRepoStub) GetActiveContract(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
	return nil, errors.New("not found")
}
func (s *ccasContractRepoStub) GetFiltered(context.Context, *uuid.UUID, entities.SmartContractType, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return s.filtered, int64(len(s.filtered)), nil
}
func (s *ccasContractRepoStub) GetByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *ccasContractRepoStub) GetAll(context.Context, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *ccasContractRepoStub) Update(context.Context, *entities.SmartContract) error { return nil }
func (s *ccasContractRepoStub) SoftDelete(context.Context, uuid.UUID) error           { return nil }

func TestContractConfigAuditUsecase_CheckByContractID_EvmDestinationSortingAndOnchainBranch(t *testing.T) {
	sourceID := uuid.New()
	contractID := uuid.New()
	chainAID := uuid.New()
	chainBID := uuid.New()

	source := &entities.Chain{ID: sourceID, Name: "Base", ChainID: "8453", Type: entities.ChainTypeEVM, IsActive: true}
	destB := &entities.Chain{ID: chainBID, Name: "B Chain", ChainID: "999", Type: entities.ChainTypeEVM, IsActive: true}
	destA := &entities.Chain{ID: chainAID, Name: "A Chain", ChainID: "42161", Type: entities.ChainTypeEVM, IsActive: true}
	contract := &entities.SmartContract{ID: contractID, Name: "Router", Type: entities.ContractTypeRouter, ChainUUID: sourceID, ContractAddress: "0x1111111111111111111111111111111111111111", IsActive: true}

	chainRepo := &ccasChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{sourceID: source},
		// intentionally unsorted to hit sort.Slice branch
		all: []*entities.Chain{source, destB, destA},
	}
	contractRepo := &ccasContractRepoStub{
		contract: contract,
		filtered: []*entities.SmartContract{contract},
	}

	u := NewContractConfigAuditUsecase(chainRepo, contractRepo, nil)
	result, err := u.CheckByContractID(context.Background(), contractID)
	require.NoError(t, err)
	require.Len(t, result.DestinationAudits, 2)
	require.Equal(t, "eip155:42161", result.DestinationAudits[0].DestChainID)
	require.Equal(t, "eip155:999", result.DestinationAudits[1].DestChainID)
}
