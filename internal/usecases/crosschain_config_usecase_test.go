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

func TestCrosschainConfigUsecase_Overview_GetAllError(t *testing.T) {
	chainRepo := new(MockChainRepository)
	tokenRepo := new(MockTokenRepository)
	contractRepo := new(MockSmartContractRepository)
	chainRepo.On("GetAll", mock.Anything).Return(([]*entities.Chain)(nil), errors.New("db failed"))

	u := uc.NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, &uc.OnchainAdapterUsecase{})
	_, err := u.Overview(context.Background(), "", "", utils.PaginationParams{Page: 1, Limit: 20})
	require.Error(t, err)
	require.Contains(t, err.Error(), "db failed")
}

func TestCrosschainConfigUsecase_Overview_InvalidSource(t *testing.T) {
	chainRepo := new(MockChainRepository)
	tokenRepo := new(MockTokenRepository)
	contractRepo := new(MockSmartContractRepository)
	chainRepo.On("GetAll", mock.Anything).Return([]*entities.Chain{}, nil)

	chainRepo.On("GetByChainID", mock.Anything, "invalid").Return((*entities.Chain)(nil), errors.New("not found")).Twice()

	u := uc.NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, &uc.OnchainAdapterUsecase{})
	_, err := u.Overview(context.Background(), "invalid", "", utils.PaginationParams{Page: 1, Limit: 20})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestCrosschainConfigUsecase_RecheckRoute_InvalidInput(t *testing.T) {
	chainRepo := new(MockChainRepository)
	tokenRepo := new(MockTokenRepository)
	contractRepo := new(MockSmartContractRepository)

	source := &entities.Chain{ID: uuid.New(), ChainID: "8453", Type: entities.ChainTypeEVM}
	chainRepo.On("GetByCAIP2", mock.Anything, "eip155:8453").Return(source, nil)
	chainRepo.On("GetByChainID", mock.Anything, "invalid").Return((*entities.Chain)(nil), errors.New("not found")).Twice()

	u := uc.NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, &uc.OnchainAdapterUsecase{})
	_, err := u.RecheckRoute(context.Background(), "eip155:8453", "invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}
