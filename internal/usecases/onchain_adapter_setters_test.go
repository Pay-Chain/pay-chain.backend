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
)

func TestOnchainAdapterUsecase_Setters_InvalidSourceChain(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	chainRepo.On("GetByChainID", mock.Anything, "invalid-source").Return(nil, errors.New("not found")).Times(8)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, nil, "")

	_, err := u.SetDefaultBridgeType(context.Background(), "invalid-source", "eip155:42161", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")

	_, _, err = u.SetHyperbridgeConfig(context.Background(), "invalid-source", "eip155:42161", "0x01", "0x02")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")

	_, _, err = u.SetCCIPConfig(context.Background(), "invalid-source", "eip155:42161", nil, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")

	_, _, err = u.SetLayerZeroConfig(context.Background(), "invalid-source", "eip155:42161", nil, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_Setters_InvalidDestChain(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	source := &entities.Chain{ID: uuid.New(), ChainID: "8453", Type: entities.ChainTypeEVM}
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil).Times(4)
	chainRepo.On("GetByChainID", mock.Anything, "invalid-dest").Return(nil, errors.New("not found")).Times(8)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, nil, "")

	_, err := u.SetDefaultBridgeType(context.Background(), "eip155:8453", "invalid-dest", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")

	_, _, err = u.SetHyperbridgeConfig(context.Background(), "eip155:8453", "invalid-dest", "0x01", "0x02")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")

	_, _, err = u.SetCCIPConfig(context.Background(), "eip155:8453", "invalid-dest", nil, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")

	_, _, err = u.SetLayerZeroConfig(context.Background(), "eip155:8453", "invalid-dest", nil, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_SetHyperbridgeConfig_ClientFactoryNotConfigured(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
	dest := &entities.Chain{ID: uuid.New(), ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, nil, "")
	_, _, err := u.SetHyperbridgeConfig(context.Background(), "eip155:8453", "eip155:42161", "0x01", "0x02")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_SetCCIPConfig_ClientFactoryNotConfigured(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
	dest := &entities.Chain{ID: uuid.New(), ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, nil, "")
	selector := uint64(123)
	_, _, err := u.SetCCIPConfig(context.Background(), "eip155:8453", "eip155:42161", &selector, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_RegisterAndDefaultBridge_OwnerKeyMissing(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
	dest := &entities.Chain{ID: uuid.New(), ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil).Twice()
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, nil, "")
	_, err := u.RegisterAdapter(context.Background(), "eip155:8453", "eip155:42161", 0, "0x1111111111111111111111111111111111111111")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")

	_, err = u.SetDefaultBridgeType(context.Background(), "eip155:8453", "eip155:42161", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestOnchainAdapterUsecase_SetLayerZeroConfig_ClientFactoryNotConfigured(t *testing.T) {
	chainRepo := new(MockChainRepository)
	contractRepo := new(MockSmartContractRepository)

	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
	dest := &entities.Chain{ID: uuid.New(), ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:42161").Return(dest, nil)
	chainRepo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeGateway).Return(gateway, nil)
	contractRepo.On("GetActiveContract", mock.Anything, sourceID, entities.ContractTypeRouter).Return(router, nil)

	u := uc.NewOnchainAdapterUsecase(chainRepo, contractRepo, nil, "")
	dstEid := uint32(30110)
	_, _, err := u.SetLayerZeroConfig(
		context.Background(),
		"eip155:8453",
		"eip155:42161",
		&dstEid,
		"0x0000000000000000000000001111111111111111111111111111111111111111",
		"",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}
