package usecases_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/usecases"
)

func TestWalletUsecase_ConnectWallet_BadInput(t *testing.T) {
	uc := usecases.NewWalletUsecase(new(MockWalletRepository), new(MockUserRepository), new(MockChainRepository))

	_, err := uc.ConnectWallet(context.Background(), uuid.New(), &entities.ConnectWalletInput{
		ChainID: "",
		Address: "",
	})
	assert.ErrorIs(t, err, domainerrors.ErrBadRequest)
}

func TestWalletUsecase_ConnectWallet_ExistingWalletSameUser(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockUserRepo := new(MockUserRepository)
	mockChainRepo := new(MockChainRepository)
	uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, mockChainRepo)

	userID := uuid.New()
	chainUUID := uuid.New()
	input := &entities.ConnectWalletInput{
		ChainID: "eip155:8453",
		Address: "0xabc",
	}
	user := &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}
	existing := &entities.Wallet{ID: uuid.New(), UserID: &userID, ChainID: chainUUID, Address: input.Address}

	mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
	mockWalletRepo.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{}, nil).Once()
	mockChainRepo.On("GetByCAIP2", context.Background(), input.ChainID).Return(&entities.Chain{
		ID:      chainUUID,
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}, nil).Once()
	mockWalletRepo.On("GetByAddress", context.Background(), chainUUID, input.Address).Return(existing, nil).Once()

	got, err := uc.ConnectWallet(context.Background(), userID, input)
	assert.NoError(t, err)
	assert.Equal(t, existing.ID, got.ID)
}

func TestWalletUsecase_ConnectWallet_KYCRequiredForAdditionalWallet(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockUserRepo := new(MockUserRepository)
	mockChainRepo := new(MockChainRepository)
	uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, mockChainRepo)

	userID := uuid.New()
	user := &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCNotStarted}

	mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
	mockWalletRepo.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{{ID: uuid.New()}}, nil).Once()

	_, err := uc.ConnectWallet(context.Background(), userID, &entities.ConnectWalletInput{
		ChainID: "eip155:8453",
		Address: "0xabc",
	})
	assert.Error(t, err)
	assert.Equal(t, domainerrors.ErrForbidden.Error(), err.Error())
}

func TestWalletUsecase_DisconnectWallet_Forbidden(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	uc := usecases.NewWalletUsecase(mockWalletRepo, new(MockUserRepository), new(MockChainRepository))

	userID := uuid.New()
	otherUser := uuid.New()
	walletID := uuid.New()

	mockWalletRepo.On("GetByID", context.Background(), walletID).Return(&entities.Wallet{
		ID:     walletID,
		UserID: &otherUser,
	}, nil).Once()

	err := uc.DisconnectWallet(context.Background(), userID, walletID)
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
}

func TestWalletUsecase_DisconnectWallet_Success(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	uc := usecases.NewWalletUsecase(mockWalletRepo, new(MockUserRepository), new(MockChainRepository))

	userID := uuid.New()
	walletID := uuid.New()

	mockWalletRepo.On("GetByID", context.Background(), walletID).Return(&entities.Wallet{
		ID:     walletID,
		UserID: &userID,
	}, nil).Once()
	mockWalletRepo.On("SoftDelete", context.Background(), walletID).Return(nil).Once()

	err := uc.DisconnectWallet(context.Background(), userID, walletID)
	assert.NoError(t, err)
}
