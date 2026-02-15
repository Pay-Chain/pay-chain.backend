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
