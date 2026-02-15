package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	uc "pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/utils"
)

func TestContractConfigAuditUsecase_Check_InvalidSource(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	chainRepo.On("GetByChainID", mock.Anything, "invalid-source").Return((*entities.Chain)(nil), errors.New("not found")).Twice()

	u := uc.NewContractConfigAuditUsecase(chainRepo, contractRepo, nil)
	_, err := u.Check(context.Background(), "invalid-source", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid sourceChainId")
}

func TestContractConfigAuditUsecase_Check_SourceChainLoadError(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	id := uuid.New()
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(&entities.Chain{ID: id}, nil)
	chainRepo.On("GetByID", mock.Anything, id).Return((*entities.Chain)(nil), errors.New("db down"))

	u := uc.NewContractConfigAuditUsecase(chainRepo, contractRepo, nil)
	_, err := u.Check(context.Background(), "eip155:8453", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load source chain")
	chainRepo.AssertExpectations(t)
}

func TestContractConfigAuditUsecase_Check_NoActiveContractWarn(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	id := uuid.New()
	chain := &entities.Chain{ID: id, ChainID: "8453", Type: entities.ChainTypeSVM, IsActive: true, Name: "Solana"}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(chain, nil)
	chainRepo.On("GetByID", mock.Anything, id).Return(chain, nil)
	contractRepo.On("GetFiltered", mock.Anything, &id, entities.SmartContractType(""), utils.PaginationParams{Page: 1, Limit: 0}).Return([]*entities.SmartContract{}, int64(0), nil)

	u := uc.NewContractConfigAuditUsecase(chainRepo, contractRepo, nil)
	res, err := u.Check(context.Background(), "eip155:8453", "")
	require.NoError(t, err)
	require.Equal(t, "WARN", res.OverallStatus)
	require.Equal(t, 1, res.Summary["warn"])
	require.NotEmpty(t, res.GlobalChecks)

	chainRepo.AssertExpectations(t)
	contractRepo.AssertExpectations(t)
}

func TestContractConfigAuditUsecase_CheckByContractID_NonEvmSkipsOnchain(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	sourceID := uuid.New()
	destID := uuid.New()
	contractID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "mainnet", Type: entities.ChainTypeSVM, IsActive: true, Name: "Solana"}
	dest := &entities.Chain{ID: destID, ChainID: "8453", Type: entities.ChainTypeEVM, IsActive: true, Name: "Base"}
	contract := &entities.SmartContract{ID: contractID, Name: "Gateway", Type: entities.ContractTypeGateway, ChainUUID: sourceID, ContractAddress: "0xabc", IsActive: true}

	contractRepo.On("GetByID", mock.Anything, contractID).Return(contract, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	chainRepo.On("GetAll", mock.Anything).Return([]*entities.Chain{source, dest}, nil)
	contractRepo.On("GetFiltered", mock.Anything, &sourceID, entities.SmartContractType(""), utils.PaginationParams{Page: 1, Limit: 0}).Return([]*entities.SmartContract{contract}, int64(1), nil)

	u := uc.NewContractConfigAuditUsecase(chainRepo, contractRepo, nil)
	res, err := u.CheckByContractID(context.Background(), contractID)
	require.NoError(t, err)
	require.Len(t, res.DestinationAudits, 1)
	require.Equal(t, "WARN", res.DestinationAudits[0].OverallStatus)
	require.Equal(t, "ONCHAIN_AUDIT_SKIPPED", res.DestinationAudits[0].Checks[0].Code)

	chainRepo.AssertExpectations(t)
	contractRepo.AssertExpectations(t)
}
