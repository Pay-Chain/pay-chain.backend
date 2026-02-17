package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/jwt"
)

func TestAuthUsecase_Register_ErrorBranches_GapExtra(t *testing.T) {
	t.Run("get by email unexpected error", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		emailRepo := new(MockEmailVerificationRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

		input := &entities.CreateUserInput{Email: "err@mail.com", Name: "E", Password: "Password123!", WalletAddress: "0xabc", WalletChainID: "eip155:8453", WalletSignature: "sig"}
		userRepo.On("GetByEmail", context.Background(), input.Email).Return(nil, errors.New("db down")).Once()

		_, _, err := uc.Register(context.Background(), input)
		assert.EqualError(t, err, "db down")
	})

	t.Run("wallet chain resolve failed", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		emailRepo := new(MockEmailVerificationRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

		input := &entities.CreateUserInput{Email: "chain@mail.com", Name: "C", Password: "Password123!", WalletAddress: "0xabc", WalletChainID: "eip155:8453", WalletSignature: "sig"}
		userRepo.On("GetByEmail", context.Background(), input.Email).Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), input.WalletChainID).Return(nil, errors.New("chain lookup failed")).Once()
		chainRepo.On("GetByChainID", context.Background(), mock.AnythingOfType("string")).Return(nil, errors.New("chain lookup failed")).Twice()

		_, _, err := uc.Register(context.Background(), input)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})

	t.Run("wallet lookup unexpected error", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		emailRepo := new(MockEmailVerificationRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

		input := &entities.CreateUserInput{Email: "wallet@mail.com", Name: "W", Password: "Password123!", WalletAddress: "0xabc", WalletChainID: "eip155:8453", WalletSignature: "sig"}
		chainUUID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), input.Email).Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), input.WalletChainID).Return(&entities.Chain{ID: chainUUID, Type: entities.ChainTypeEVM, ChainID: "8453"}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, input.WalletAddress).Return(nil, errors.New("wallet query failed")).Once()

		_, _, err := uc.Register(context.Background(), input)
		assert.EqualError(t, err, "wallet query failed")
	})

	t.Run("wallet already linked to another user", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		emailRepo := new(MockEmailVerificationRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

		input := &entities.CreateUserInput{Email: "dup@mail.com", Name: "D", Password: "Password123!", WalletAddress: "0xabc", WalletChainID: "eip155:8453", WalletSignature: "sig"}
		chainUUID := uuid.New()
		otherUserID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), input.Email).Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), input.WalletChainID).Return(&entities.Chain{ID: chainUUID, Type: entities.ChainTypeEVM, ChainID: "8453"}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, input.WalletAddress).Return(&entities.Wallet{ID: uuid.New(), UserID: &otherUserID, ChainID: chainUUID, Address: input.WalletAddress}, nil).Once()

		_, _, err := uc.Register(context.Background(), input)
		assert.Error(t, err)
		assert.Equal(t, domainerrors.ErrAlreadyExists.Error(), err.Error())
	})

	t.Run("create user failed", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		emailRepo := new(MockEmailVerificationRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

		input := &entities.CreateUserInput{Email: "create@mail.com", Name: "CU", Password: "Password123!", WalletAddress: "0xabc", WalletChainID: "eip155:8453", WalletSignature: "sig"}
		chainUUID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), input.Email).Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), input.WalletChainID).Return(&entities.Chain{ID: chainUUID, Type: entities.ChainTypeEVM, ChainID: "8453"}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, input.WalletAddress).Return(nil, domainerrors.ErrNotFound).Once()
		userRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.User")).Return(errors.New("create user failed")).Once()

		_, _, err := uc.Register(context.Background(), input)
		assert.EqualError(t, err, "create user failed")
	})

	t.Run("create wallet failed", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		emailRepo := new(MockEmailVerificationRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

		input := &entities.CreateUserInput{Email: "createwallet@mail.com", Name: "CW", Password: "Password123!", WalletAddress: "0xabc", WalletChainID: "eip155:8453", WalletSignature: "sig"}
		chainUUID := uuid.New()
		createdUserID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), input.Email).Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), input.WalletChainID).Return(&entities.Chain{ID: chainUUID, Type: entities.ChainTypeEVM, ChainID: "8453"}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, input.WalletAddress).Return(nil, domainerrors.ErrNotFound).Once()
		userRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.User")).Return(nil).Run(func(args mock.Arguments) {
			u := args.Get(1).(*entities.User)
			u.ID = createdUserID
		}).Once()
		walletRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.Wallet")).Return(errors.New("create wallet failed")).Once()

		_, _, err := uc.Register(context.Background(), input)
		assert.EqualError(t, err, "create wallet failed")
	})

	t.Run("create email verification failed", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		emailRepo := new(MockEmailVerificationRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

		input := &entities.CreateUserInput{Email: "emailverif@mail.com", Name: "EV", Password: "Password123!", WalletAddress: "0xabc", WalletChainID: "eip155:8453", WalletSignature: "sig"}
		chainUUID := uuid.New()
		createdUserID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), input.Email).Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), input.WalletChainID).Return(&entities.Chain{ID: chainUUID, Type: entities.ChainTypeEVM, ChainID: "8453"}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, input.WalletAddress).Return(nil, domainerrors.ErrNotFound).Once()
		userRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.User")).Return(nil).Run(func(args mock.Arguments) {
			u := args.Get(1).(*entities.User)
			u.ID = createdUserID
		}).Once()
		walletRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.Wallet")).Return(nil).Once()
		emailRepo.On("Create", context.Background(), createdUserID, mock.AnythingOfType("string")).Return(errors.New("email create failed")).Once()

		_, _, err := uc.Register(context.Background(), input)
		assert.EqualError(t, err, "email create failed")
	})
}

func TestAuthUsecase_VerifyEmail_MarkVerifiedError_GapExtra(t *testing.T) {
	userRepo := new(MockUserRepository)
	emailRepo := new(MockEmailVerificationRepository)
	uc := newAuthUsecaseForTest(userRepo, emailRepo, new(MockWalletRepository), new(MockChainRepository))

	token := "ok-mark-fail"
	emailRepo.On("GetByToken", context.Background(), token).Return(&entities.User{ID: uuid.New()}, nil).Once()
	emailRepo.On("MarkVerified", context.Background(), token).Return(errors.New("mark failed")).Once()

	err := uc.VerifyEmail(context.Background(), token)
	assert.EqualError(t, err, "mark failed")
}

func TestAuthUsecase_RefreshToken_GetUserFailed(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	jwtSvc := jwt.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour)
	user := &entities.User{ID: uuid.New(), Email: "refresh-fail@mail.com", Role: entities.UserRoleUser}
	pair, err := jwtSvc.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	assert.NoError(t, err)

	userRepo.On("GetByID", context.Background(), user.ID).Return(nil, errors.New("user lookup failed")).Once()

	_, err = uc.RefreshToken(context.Background(), pair.RefreshToken)
	assert.EqualError(t, err, "user lookup failed")
}
