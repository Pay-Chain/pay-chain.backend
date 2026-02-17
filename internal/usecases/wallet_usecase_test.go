package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestWalletUsecase_ConnectWallet_ErrorBranches(t *testing.T) {
	t.Run("user lookup error", func(t *testing.T) {
		mockWalletRepo := new(MockWalletRepository)
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, new(MockChainRepository))

		userID := uuid.New()
		mockUserRepo.On("GetByID", context.Background(), userID).Return(nil, errors.New("user repo down")).Once()

		_, err := uc.ConnectWallet(context.Background(), userID, &entities.ConnectWalletInput{
			ChainID: "eip155:8453",
			Address: "0xabc",
		})
		assert.EqualError(t, err, "user repo down")
	})

	t.Run("get wallets error", func(t *testing.T) {
		mockWalletRepo := new(MockWalletRepository)
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, new(MockChainRepository))

		userID := uuid.New()
		user := &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}
		mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
		mockWalletRepo.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{}, errors.New("wallet repo down")).Once()

		_, err := uc.ConnectWallet(context.Background(), userID, &entities.ConnectWalletInput{
			ChainID: "eip155:8453",
			Address: "0xabc",
		})
		assert.EqualError(t, err, "wallet repo down")
	})

	t.Run("invalid chain id resolver", func(t *testing.T) {
		mockWalletRepo := new(MockWalletRepository)
		mockUserRepo := new(MockUserRepository)
		mockChainRepo := new(MockChainRepository)
		uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, mockChainRepo)

		userID := uuid.New()
		user := &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}
		mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
		mockWalletRepo.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{}, nil).Once()
		mockChainRepo.On("GetByCAIP2", context.Background(), "bad-chain").Return(nil, domainerrors.ErrNotFound).Twice()
		mockChainRepo.On("GetByChainID", context.Background(), "bad-chain").Return(nil, domainerrors.ErrNotFound).Twice()
		mockChainRepo.On("GetByID", context.Background(), mock.Anything).Return(nil, domainerrors.ErrNotFound).Maybe()

		_, err := uc.ConnectWallet(context.Background(), userID, &entities.ConnectWalletInput{
			ChainID: "bad-chain",
			Address: "0xabc",
		})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})

	t.Run("wallet exists for another user", func(t *testing.T) {
		mockWalletRepo := new(MockWalletRepository)
		mockUserRepo := new(MockUserRepository)
		mockChainRepo := new(MockChainRepository)
		uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, mockChainRepo)

		userID := uuid.New()
		otherUserID := uuid.New()
		chainUUID := uuid.New()
		input := &entities.ConnectWalletInput{ChainID: "eip155:8453", Address: "0xdup"}
		user := &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}
		existing := &entities.Wallet{ID: uuid.New(), UserID: &otherUserID, ChainID: chainUUID, Address: input.Address}

		mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
		mockWalletRepo.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{}, nil).Once()
		mockChainRepo.On("GetByCAIP2", context.Background(), input.ChainID).Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Twice()
		mockWalletRepo.On("GetByAddress", context.Background(), chainUUID, input.Address).Return(existing, nil).Once()

		_, err := uc.ConnectWallet(context.Background(), userID, input)
		assert.ErrorIs(t, err, domainerrors.ErrAlreadyExists)
	})
}

func TestWalletUsecase_ConnectWallet_CreateBranches(t *testing.T) {
	t.Run("create error", func(t *testing.T) {
		mockWalletRepo := new(MockWalletRepository)
		mockUserRepo := new(MockUserRepository)
		mockChainRepo := new(MockChainRepository)
		uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, mockChainRepo)

		userID := uuid.New()
		chainUUID := uuid.New()
		input := &entities.ConnectWalletInput{ChainID: "eip155:8453", Address: "0xnew"}
		user := &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}

		mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
		mockWalletRepo.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{}, nil).Once()
		mockChainRepo.On("GetByCAIP2", context.Background(), input.ChainID).Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Twice()
		mockWalletRepo.On("GetByAddress", context.Background(), chainUUID, input.Address).Return(nil, domainerrors.ErrNotFound).Once()
		mockWalletRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.Wallet")).Return(errors.New("create fail")).Once()

		_, err := uc.ConnectWallet(context.Background(), userID, input)
		assert.EqualError(t, err, "create fail")
	})

	t.Run("admin additional wallet bypass kyc and success", func(t *testing.T) {
		mockWalletRepo := new(MockWalletRepository)
		mockUserRepo := new(MockUserRepository)
		mockChainRepo := new(MockChainRepository)
		uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, mockChainRepo)

		userID := uuid.New()
		chainUUID := uuid.New()
		input := &entities.ConnectWalletInput{ChainID: "eip155:8453", Address: "0xadmin"}
		user := &entities.User{ID: userID, Role: entities.UserRoleAdmin, KYCStatus: entities.KYCNotStarted}

		mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
		mockWalletRepo.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{{ID: uuid.New()}}, nil).Once()
		mockChainRepo.On("GetByCAIP2", context.Background(), input.ChainID).Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Twice()
		mockWalletRepo.On("GetByAddress", context.Background(), chainUUID, input.Address).Return(nil, domainerrors.ErrNotFound).Once()
		mockWalletRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.Wallet")).Return(nil).Once()

		got, err := uc.ConnectWallet(context.Background(), userID, input)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.False(t, got.IsPrimary)
	})
}

func TestWalletUsecase_DisconnectWallet_GetByIDError(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	uc := usecases.NewWalletUsecase(mockWalletRepo, new(MockUserRepository), new(MockChainRepository))

	userID := uuid.New()
	walletID := uuid.New()
	mockWalletRepo.On("GetByID", context.Background(), walletID).Return(nil, errors.New("get fail")).Once()

	err := uc.DisconnectWallet(context.Background(), userID, walletID)
	assert.EqualError(t, err, "get fail")
}

func TestWalletUsecase_GetWallets_And_SetPrimary(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	uc := usecases.NewWalletUsecase(mockWalletRepo, new(MockUserRepository), new(MockChainRepository))

	userID := uuid.New()
	walletID := uuid.New()
	wallets := []*entities.Wallet{{ID: walletID, UserID: &userID}}

	mockWalletRepo.On("GetByUserID", context.Background(), userID).Return(wallets, nil).Once()
	got, err := uc.GetWallets(context.Background(), userID)
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, walletID, got[0].ID)

	mockWalletRepo.On("SetPrimary", context.Background(), userID, walletID).Return(nil).Once()
	err = uc.SetPrimaryWallet(context.Background(), userID, walletID)
	assert.NoError(t, err)
}

func TestWalletUsecase_ConnectWallet_GetByAddressErrorAndSecondResolveFail(t *testing.T) {
	t.Run("get by address returns unexpected error", func(t *testing.T) {
		mockWalletRepo := new(MockWalletRepository)
		mockUserRepo := new(MockUserRepository)
		mockChainRepo := new(MockChainRepository)
		uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, mockChainRepo)

		userID := uuid.New()
		chainUUID := uuid.New()
		input := &entities.ConnectWalletInput{ChainID: "eip155:8453", Address: "0xerr"}
		user := &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}

		mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
		mockWalletRepo.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{}, nil).Once()
		mockChainRepo.On("GetByCAIP2", context.Background(), input.ChainID).Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Once()
		mockWalletRepo.On("GetByAddress", context.Background(), chainUUID, input.Address).Return(nil, errors.New("lookup fail")).Once()

		_, err := uc.ConnectWallet(context.Background(), userID, input)
		assert.EqualError(t, err, "lookup fail")
	})

	t.Run("second resolver call fails after first success", func(t *testing.T) {
		mockWalletRepo := new(MockWalletRepository)
		mockUserRepo := new(MockUserRepository)
		mockChainRepo := new(MockChainRepository)
		uc := usecases.NewWalletUsecase(mockWalletRepo, mockUserRepo, mockChainRepo)

		userID := uuid.New()
		chainUUID := uuid.New()
		input := &entities.ConnectWalletInput{ChainID: "eip155:8453", Address: "0xnewfail"}
		user := &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}

		mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
		mockWalletRepo.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{}, nil).Once()
		// first ResolveFromAny -> success
		mockChainRepo.On("GetByCAIP2", context.Background(), input.ChainID).Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Once()
		mockWalletRepo.On("GetByAddress", context.Background(), chainUUID, input.Address).Return(nil, domainerrors.ErrNotFound).Once()
		// second ResolveFromAny -> fail
		mockChainRepo.On("GetByCAIP2", context.Background(), input.ChainID).Return(nil, domainerrors.ErrNotFound).Once()
		mockChainRepo.On("GetByChainID", context.Background(), input.ChainID).Return(nil, domainerrors.ErrNotFound).Once()
		mockChainRepo.On("GetByChainID", context.Background(), "8453").Return(nil, domainerrors.ErrNotFound).Once()

		_, err := uc.ConnectWallet(context.Background(), userID, input)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})
}
