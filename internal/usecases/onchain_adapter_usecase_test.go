package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
	uc "pay-chain.backend/internal/usecases"
)

func TestOnchainAdapterUsecase_GetStatus_InvalidSource(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	chainRepo.On("GetByChainID", mock.Anything, "invalid-source").Return((*entities.Chain)(nil), errors.New("not found")).Twice()

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, blockchain.NewClientFactory(), "")
	_, err := u.GetStatus(context.Background(), "invalid-source", "eip155:42161")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_GetStatus_InvalidDest(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	source := &entities.Chain{ID: uuid.New(), ChainID: "8453", Type: entities.ChainTypeEVM}
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByChainID", mock.Anything, "invalid-dest").Return((*entities.Chain)(nil), errors.New("not found")).Twice()

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, blockchain.NewClientFactory(), "")
	_, err := u.GetStatus(context.Background(), "eip155:8453", "invalid-dest")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_GetStatus_SourceNonEVM(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "mainnet", Type: entities.ChainTypeSVM}
	dest := &entities.Chain{ID: uuid.New(), ChainID: "42161", Type: entities.ChainTypeEVM}

	chainRepo.On("GetByCAIP2", mock.Anything, "solana:mainnet").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, blockchain.NewClientFactory(), "")
	_, err := u.GetStatus(context.Background(), "solana:mainnet", "eip155:42161")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_GetStatus_GatewayMissing(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "http://127.0.0.1:8545"}
	dest := &entities.Chain{ID: uuid.New(), ChainID: "42161", Type: entities.ChainTypeEVM}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return((*entities.SmartContract)(nil), errors.New("missing"))

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, blockchain.NewClientFactory(), "")
	_, err := u.GetStatus(context.Background(), "eip155:8453", "eip155:42161")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_GetStatus_RouterMissing(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "http://127.0.0.1:8545"}
	dest := &entities.Chain{ID: uuid.New(), ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x0000000000000000000000000000000000000001", IsActive: true}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return((*entities.SmartContract)(nil), errors.New("missing"))

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, blockchain.NewClientFactory(), "")
	_, err := u.GetStatus(context.Background(), "eip155:8453", "eip155:42161")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_GetStatus_NoRPC(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
	dest := &entities.Chain{ID: uuid.New(), ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x0000000000000000000000000000000000000001", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x0000000000000000000000000000000000000002", IsActive: true}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, blockchain.NewClientFactory(), "")
	_, err := u.GetStatus(context.Background(), "eip155:8453", "eip155:42161")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_GetStatus_RPCConnectError(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "http://127.0.0.1:0"}
	dest := &entities.Chain{ID: uuid.New(), ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x0000000000000000000000000000000000000001", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x0000000000000000000000000000000000000002", IsActive: true}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, blockchain.NewClientFactory(), "")
	_, err := u.GetStatus(context.Background(), "eip155:8453", "eip155:42161")
	require.Error(t, err)
}
