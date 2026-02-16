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

func TestContractConfigAuditUsecase_Check_DestinationInvalidStillReturnsReport(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeSVM, IsActive: true}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:bad").Return((*entities.Chain)(nil), errors.New("not found"))
	chainRepo.On("GetByChainID", mock.Anything, mock.AnythingOfType("string")).Return((*entities.Chain)(nil), errors.New("not found")).Maybe()

	contractRepo.On("GetFiltered", mock.Anything, &sourceID, entities.SmartContractType(""), utils.PaginationParams{Page: 1, Limit: 0}).
		Return([]*entities.SmartContract{}, int64(0), nil)

	u := uc.NewContractConfigAuditUsecase(chainRepo, contractRepo, nil)
	res, err := u.Check(context.Background(), "eip155:8453", "eip155:bad")
	require.NoError(t, err)
	require.Equal(t, "ERROR", res.OverallStatus)
	require.GreaterOrEqual(t, res.Summary["error"], 1)
	require.NotEmpty(t, res.GlobalChecks)
}

func TestContractConfigAuditUsecase_Check_ContractListError(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, IsActive: true}
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetFiltered", mock.Anything, &sourceID, entities.SmartContractType(""), utils.PaginationParams{Page: 1, Limit: 0}).
		Return(nil, int64(0), errors.New("db down"))

	u := uc.NewContractConfigAuditUsecase(chainRepo, contractRepo, nil)
	_, err := u.Check(context.Background(), "eip155:8453", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to list contracts")
}

func TestContractConfigAuditUsecase_Check_EvmDestRunsOnchainChecksWithRpcMissing(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{
		ID:      sourceID,
		ChainID: "8453",
		Type:    entities.ChainTypeEVM,
		IsActive: true,
		// intentionally no RPCURL and no active RPC entries
	}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM, IsActive: true}

	gateway := &entities.SmartContract{
		ID:              uuid.New(),
		Name:            "Gateway",
		Type:            entities.ContractTypeGateway,
		ChainUUID:       sourceID,
		ContractAddress: "0x1111111111111111111111111111111111111111",
		IsActive:        true,
		ABI: []interface{}{
			map[string]interface{}{"type": "function", "name": "createPayment"},
			map[string]interface{}{"type": "function", "name": "createPaymentRequest"},
			map[string]interface{}{"type": "function", "name": "setDefaultBridgeType"},
			map[string]interface{}{"type": "function", "name": "defaultBridgeTypes"},
		},
	}
	router := &entities.SmartContract{
		ID:              uuid.New(),
		Name:            "Router",
		Type:            entities.ContractTypeRouter,
		ChainUUID:       sourceID,
		ContractAddress: "0x2222222222222222222222222222222222222222",
		IsActive:        true,
		ABI: []interface{}{
			map[string]interface{}{"type": "function", "name": "registerAdapter"},
			map[string]interface{}{"type": "function", "name": "hasAdapter"},
			map[string]interface{}{"type": "function", "name": "quotePaymentFee"},
			map[string]interface{}{"type": "function", "name": "routePayment"},
		},
	}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	contractRepo.On("GetFiltered", mock.Anything, &sourceID, entities.SmartContractType(""), utils.PaginationParams{Page: 1, Limit: 0}).
		Return([]*entities.SmartContract{gateway, router}, int64(2), nil)

	u := uc.NewContractConfigAuditUsecase(chainRepo, contractRepo, nil)
	res, err := u.Check(context.Background(), "eip155:8453", "eip155:42161")
	require.NoError(t, err)
	require.Equal(t, "ERROR", res.OverallStatus)
	require.GreaterOrEqual(t, res.Summary["error"], 1)
}
