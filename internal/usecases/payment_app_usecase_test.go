package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
)

func TestPaymentAppUsecase_CreatePaymentApp_InvalidSourceChain(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockChainRepo := new(MockChainRepository)

	uc := usecases.NewPaymentAppUsecase(nil, mockUserRepo, mockWalletRepo, mockChainRepo)

	mockChainRepo.On("GetByChainID", context.Background(), "invalid-source").Return(nil, errors.New("not found")).Twice()

	_, err := uc.CreatePaymentApp(context.Background(), &entities.CreatePaymentAppInput{
		SourceChainID:       "invalid-source",
		DestChainID:         "eip155:42161",
		SourceTokenAddress:  "0x1",
		DestTokenAddress:    "0x2",
		Amount:              "1",
		Decimals:            6,
		SenderWalletAddress: "0xabc",
		ReceiverAddress:     "0xdef",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source chain")
}

func TestPaymentAppUsecase_CreatePaymentApp_InvalidDestChain(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockChainRepo := new(MockChainRepository)

	uc := usecases.NewPaymentAppUsecase(nil, mockUserRepo, mockWalletRepo, mockChainRepo)

	srcID := uuid.New()
	mockChainRepo.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{
		ID:      srcID,
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}, nil).Once()
	mockChainRepo.On("GetByChainID", context.Background(), "bad-dest").Return(nil, errors.New("not found")).Twice()

	_, err := uc.CreatePaymentApp(context.Background(), &entities.CreatePaymentAppInput{
		SourceChainID:       "eip155:8453",
		DestChainID:         "bad-dest",
		SourceTokenAddress:  "0x1",
		DestTokenAddress:    "0x2",
		Amount:              "1",
		Decimals:            6,
		SenderWalletAddress: "0xabc",
		ReceiverAddress:     "0xdef",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid destination chain")
}

func TestPaymentAppUsecase_CreatePaymentApp_AutoCreateUserFails(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockChainRepo := new(MockChainRepository)

	uc := usecases.NewPaymentAppUsecase(nil, mockUserRepo, mockWalletRepo, mockChainRepo)

	srcID := uuid.New()
	destID := uuid.New()
	sourceCAIP2 := "eip155:8453"
	destCAIP2 := "eip155:42161"
	sender := "0xE6A7d99011257AEc28Ad60EFED58A256c4d5Fea3"

	mockChainRepo.On("GetByCAIP2", context.Background(), sourceCAIP2).Return(&entities.Chain{
		ID:      srcID,
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}, nil).Once()
	mockChainRepo.On("GetByCAIP2", context.Background(), destCAIP2).Return(&entities.Chain{
		ID:      destID,
		Type:    entities.ChainTypeEVM,
		ChainID: "42161",
	}, nil).Once()
	mockWalletRepo.On("GetByAddress", context.Background(), srcID, sender).Return(nil, errors.New("not found")).Once()
	mockUserRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.User")).Return(errors.New("insert error")).Once()

	_, err := uc.CreatePaymentApp(context.Background(), &entities.CreatePaymentAppInput{
		SourceChainID:       sourceCAIP2,
		DestChainID:         destCAIP2,
		SourceTokenAddress:  "0x1",
		DestTokenAddress:    "0x2",
		Amount:              "1",
		Decimals:            6,
		SenderWalletAddress: sender,
		ReceiverAddress:     "0xdef",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to auto-create user")
}

func TestPaymentAppUsecase_CreatePaymentApp_AutoCreateWalletFails(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockChainRepo := new(MockChainRepository)

	uc := usecases.NewPaymentAppUsecase(nil, mockUserRepo, mockWalletRepo, mockChainRepo)

	srcID := uuid.New()
	destID := uuid.New()
	sourceCAIP2 := "eip155:8453"
	destCAIP2 := "eip155:42161"
	sender := "0xE6A7d99011257AEc28Ad60EFED58A256c4d5Fea3"

	mockChainRepo.On("GetByCAIP2", context.Background(), sourceCAIP2).Return(&entities.Chain{
		ID:      srcID,
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}, nil).Once()
	mockChainRepo.On("GetByCAIP2", context.Background(), destCAIP2).Return(&entities.Chain{
		ID:      destID,
		Type:    entities.ChainTypeEVM,
		ChainID: "42161",
	}, nil).Once()
	mockWalletRepo.On("GetByAddress", context.Background(), srcID, sender).Return(nil, errors.New("not found")).Once()
	mockUserRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.User")).Return(nil).Once()
	mockWalletRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.Wallet")).Return(errors.New("wallet insert error")).Once()

	_, err := uc.CreatePaymentApp(context.Background(), &entities.CreatePaymentAppInput{
		SourceChainID:       sourceCAIP2,
		DestChainID:         destCAIP2,
		SourceTokenAddress:  "0x1",
		DestTokenAddress:    "0x2",
		Amount:              "1",
		Decimals:            6,
		SenderWalletAddress: sender,
		ReceiverAddress:     "0xdef",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create wallet")
}
