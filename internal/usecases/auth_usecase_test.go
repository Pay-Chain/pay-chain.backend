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
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/crypto"
	"pay-chain.backend/pkg/jwt"
)

func newAuthUsecaseForTest(
	userRepo *MockUserRepository,
	emailRepo *MockEmailVerificationRepository,
	walletRepo *MockWalletRepository,
	chainRepo *MockChainRepository,
) *usecases.AuthUsecase {
	jwtSvc := jwt.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour)
	return usecases.NewAuthUsecase(userRepo, emailRepo, walletRepo, chainRepo, jwtSvc)
}

func TestAuthUsecase_Register_BadWalletInput(t *testing.T) {
	uc := newAuthUsecaseForTest(new(MockUserRepository), new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{
		Email: "a@mail.com",
		Name:  "A",
	})
	assert.ErrorIs(t, err, domainerrors.ErrBadRequest)
}

func TestAuthUsecase_Register_EmailAlreadyExists(t *testing.T) {
	userRepo := new(MockUserRepository)
	emailRepo := new(MockEmailVerificationRepository)
	walletRepo := new(MockWalletRepository)
	chainRepo := new(MockChainRepository)
	uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

	userRepo.On("GetByEmail", context.Background(), "exists@mail.com").Return(&entities.User{ID: uuid.New()}, nil).Once()

	_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{
		Email:           "exists@mail.com",
		Name:            "Exists",
		Password:        "Password123!",
		WalletAddress:   "0xabc",
		WalletChainID:   "8453",
		WalletSignature: "sig",
	})
	assert.ErrorIs(t, err, domainerrors.ErrAlreadyExists)
}

func TestAuthUsecase_Register_Success(t *testing.T) {
	userRepo := new(MockUserRepository)
	emailRepo := new(MockEmailVerificationRepository)
	walletRepo := new(MockWalletRepository)
	chainRepo := new(MockChainRepository)
	uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

	input := &entities.CreateUserInput{
		Email:           "new@mail.com",
		Name:            "New User",
		Password:        "Password123!",
		WalletAddress:   "0xabc",
		WalletChainID:   "eip155:8453",
		WalletSignature: "signature",
	}
	chainUUID := uuid.New()
	createdUserID := uuid.New()

	userRepo.On("GetByEmail", context.Background(), input.Email).Return(nil, domainerrors.ErrNotFound).Once()
	chainRepo.On("GetByCAIP2", context.Background(), input.WalletChainID).Return(&entities.Chain{
		ID:      chainUUID,
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}, nil).Once()
	walletRepo.On("GetByAddress", context.Background(), chainUUID, input.WalletAddress).Return(nil, domainerrors.ErrNotFound).Once()
	userRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.User")).Return(nil).Run(func(args mock.Arguments) {
		u := args.Get(1).(*entities.User)
		u.ID = createdUserID
	}).Once()
	walletRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.Wallet")).Return(nil).Once()
	emailRepo.On("Create", context.Background(), createdUserID, mock.AnythingOfType("string")).Return(nil).Once()

	user, token, err := uc.Register(context.Background(), input)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, token)
	assert.Equal(t, input.Email, user.Email)
}

func TestAuthUsecase_Login_InvalidCredentialCases(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	userRepo.On("GetByEmail", context.Background(), "missing@mail.com").Return(nil, domainerrors.ErrNotFound).Once()
	_, err := uc.Login(context.Background(), &entities.LoginInput{
		Email:    "missing@mail.com",
		Password: "whatever",
	})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidCredentials)

	hashed, _ := crypto.HashPassword("correct-password")
	userRepo.On("GetByEmail", context.Background(), "user@mail.com").Return(&entities.User{
		ID:           uuid.New(),
		Email:        "user@mail.com",
		PasswordHash: hashed,
		Role:         entities.UserRoleUser,
	}, nil).Once()
	_, err = uc.Login(context.Background(), &entities.LoginInput{
		Email:    "user@mail.com",
		Password: "wrong-password",
	})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidCredentials)
}

func TestAuthUsecase_Login_SuccessNoSession(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	hashed, _ := crypto.HashPassword("correct-password")
	user := &entities.User{
		ID:           uuid.New(),
		Email:        "user@mail.com",
		PasswordHash: hashed,
		Role:         entities.UserRoleAdmin,
	}
	userRepo.On("GetByEmail", context.Background(), user.Email).Return(user, nil).Once()

	resp, err := uc.Login(context.Background(), &entities.LoginInput{
		Email:    user.Email,
		Password: "correct-password",
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, user.ID, resp.User.ID)
}

func TestAuthUsecase_VerifyEmail(t *testing.T) {
	userRepo := new(MockUserRepository)
	emailRepo := new(MockEmailVerificationRepository)
	uc := newAuthUsecaseForTest(userRepo, emailRepo, new(MockWalletRepository), new(MockChainRepository))

	emailRepo.On("GetByToken", context.Background(), "bad-token").Return(nil, errors.New("not found")).Once()
	err := uc.VerifyEmail(context.Background(), "bad-token")
	assert.Error(t, err)

	emailRepo.On("GetByToken", context.Background(), "ok-token").Return(&entities.User{ID: uuid.New()}, nil).Once()
	emailRepo.On("MarkVerified", context.Background(), "ok-token").Return(nil).Once()
	err = uc.VerifyEmail(context.Background(), "ok-token")
	assert.NoError(t, err)
}

func TestAuthUsecase_RefreshToken(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	_, err := uc.RefreshToken(context.Background(), "not-a-jwt")
	assert.Error(t, err)

	user := &entities.User{
		ID:    uuid.New(),
		Email: "refresh@mail.com",
		Role:  entities.UserRoleUser,
	}
	jwtSvc := jwt.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour)
	pair, genErr := jwtSvc.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	assert.NoError(t, genErr)

	userRepo.On("GetByID", context.Background(), user.ID).Return(user, nil).Once()
	newPair, err := uc.RefreshToken(context.Background(), pair.RefreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)
}

func TestAuthUsecase_GetTokenExpiry(t *testing.T) {
	uc := newAuthUsecaseForTest(new(MockUserRepository), new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	_, err := uc.GetTokenExpiry("bad-token")
	assert.Error(t, err)

	userID := uuid.New()
	jwtSvc := jwt.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour)
	pair, genErr := jwtSvc.GenerateTokenPair(userID, "exp@mail.com", string(entities.UserRoleUser))
	assert.NoError(t, genErr)

	exp, err := uc.GetTokenExpiry(pair.AccessToken)
	assert.NoError(t, err)
	assert.Greater(t, exp, int64(0))
}

func TestAuthUsecase_ChangePassword(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	userID := uuid.New()
	currentHash, _ := crypto.HashPassword("current-pass")
	user := &entities.User{
		ID:           userID,
		Email:        "cp@mail.com",
		PasswordHash: currentHash,
		Role:         entities.UserRoleUser,
	}
	userRepo.On("GetByID", context.Background(), userID).Return(user, nil).Twice()

	err := uc.ChangePassword(context.Background(), userID, &entities.ChangePasswordInput{
		CurrentPassword: "wrong-pass",
		NewPassword:     "new-pass-123",
	})
	assert.Error(t, err)

	userRepo.On("UpdatePassword", context.Background(), userID, mock.AnythingOfType("string")).Return(nil).Once()
	err = uc.ChangePassword(context.Background(), userID, &entities.ChangePasswordInput{
		CurrentPassword: "current-pass",
		NewPassword:     "new-pass-123",
	})
	assert.NoError(t, err)
}
