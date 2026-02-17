package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/crypto"
	"pay-chain.backend/pkg/jwt"
	redispkg "pay-chain.backend/pkg/redis"
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

func TestAuthUsecase_GetTokenExpiry_MissingExpClaim(t *testing.T) {
	uc := newAuthUsecaseForTest(new(MockUserRepository), new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	raw := gojwt.NewWithClaims(gojwt.SigningMethodHS256, &gojwt.MapClaims{
		"userId": uuid.New().String(),
		"email":  "no-exp@mail.com",
		"role":   string(entities.UserRoleUser),
	})
	token, err := raw.SignedString([]byte("test-secret"))
	assert.NoError(t, err)

	_, err = uc.GetTokenExpiry(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing exp")
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

func TestAuthUsecase_ChangePassword_ErrorBranches(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))
	userID := uuid.New()

	userRepo.On("GetByID", context.Background(), userID).Return(nil, errors.New("db down")).Once()
	err := uc.ChangePassword(context.Background(), userID, &entities.ChangePasswordInput{
		CurrentPassword: "any-pass",
		NewPassword:     "new-pass-123",
	})
	assert.EqualError(t, err, "db down")

	currentHash, _ := crypto.HashPassword("current-pass")
	userRepo.On("GetByID", context.Background(), userID).Return(&entities.User{
		ID:           userID,
		Email:        "cp2@mail.com",
		PasswordHash: currentHash,
		Role:         entities.UserRoleUser,
	}, nil).Once()
	userRepo.On("UpdatePassword", context.Background(), userID, mock.AnythingOfType("string")).Return(errors.New("update fail")).Once()

	err = uc.ChangePassword(context.Background(), userID, &entities.ChangePasswordInput{
		CurrentPassword: "current-pass",
		NewPassword:     "another-pass-123",
	})
	assert.EqualError(t, err, "update fail")
}

func TestAuthUsecase_ChangePassword_NewPasswordTooLong(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))
	userID := uuid.New()

	currentHash, _ := crypto.HashPassword("current-pass")
	userRepo.On("GetByID", context.Background(), userID).Return(&entities.User{
		ID:           userID,
		Email:        "cp3@mail.com",
		PasswordHash: currentHash,
		Role:         entities.UserRoleUser,
	}, nil).Once()

	// bcrypt rejects passwords longer than 72 bytes.
	tooLongPassword := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-too-long-password"

	err := uc.ChangePassword(context.Background(), userID, &entities.ChangePasswordInput{
		CurrentPassword: "current-pass",
		NewPassword:     tooLongPassword,
	})
	assert.Error(t, err)
}

func TestAuthUsecase_GetUserByID(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	id := uuid.New()
	user := &entities.User{ID: id, Email: "u@paychain.io"}
	userRepo.On("GetByID", context.Background(), id).Return(user, nil).Once()

	got, err := uc.GetUserByID(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestAuthUsecase_Login_UserRepoError(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	userRepo.On("GetByEmail", context.Background(), "err@mail.com").Return(nil, errors.New("db down")).Once()
	_, err := uc.Login(context.Background(), &entities.LoginInput{
		Email:    "err@mail.com",
		Password: "whatever",
	})
	assert.EqualError(t, err, "db down")
}

func TestAuthUsecase_Login_UseSessionRedisError(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	redispkg.SetClient(redisv9.NewClient(&redisv9.Options{
		Addr:         "127.0.0.1:0",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	}))

	hashed, _ := crypto.HashPassword("correct-password")
	user := &entities.User{
		ID:           uuid.New(),
		Email:        "session@mail.com",
		PasswordHash: hashed,
		Role:         entities.UserRoleUser,
	}
	userRepo.On("GetByEmail", context.Background(), user.Email).Return(user, nil).Once()

	_, err := uc.Login(context.Background(), &entities.LoginInput{
		Email:      user.Email,
		Password:   "correct-password",
		UseSession: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store session in redis")
}

func TestAuthUsecase_Login_UseSessionSuccess(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable: %v", err)
	}
	defer srv.Close()

	redispkg.SetClient(redisv9.NewClient(&redisv9.Options{
		Addr: srv.Addr(),
	}))

	hashed, _ := crypto.HashPassword("correct-password")
	user := &entities.User{
		ID:           uuid.New(),
		Email:        "session-ok@mail.com",
		PasswordHash: hashed,
		Role:         entities.UserRoleUser,
	}
	userRepo.On("GetByEmail", context.Background(), user.Email).Return(user, nil).Once()

	resp, err := uc.Login(context.Background(), &entities.LoginInput{
		Email:      user.Email,
		Password:   "correct-password",
		UseSession: true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.Empty(t, resp.AccessToken)
}

func TestAuthUsecase_Register_ErrorBranches(t *testing.T) {
	t.Run("user repo email lookup error", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

		userRepo.On("GetByEmail", context.Background(), "err@mail.com").Return(nil, errors.New("db down")).Once()
		_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{
			Email:           "err@mail.com",
			Name:            "Err",
			Password:        "Password123!",
			WalletAddress:   "0xabc",
			WalletChainID:   "eip155:8453",
			WalletSignature: "sig",
		})
		assert.EqualError(t, err, "db down")
	})

	t.Run("invalid chain input", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), chainRepo)

		userRepo.On("GetByEmail", context.Background(), "new@mail.com").Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), "bad-chain").Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByChainID", context.Background(), "bad-chain").Return(nil, domainerrors.ErrNotFound).Maybe()

		_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{
			Email:           "new@mail.com",
			Name:            "New",
			Password:        "Password123!",
			WalletAddress:   "0xabc",
			WalletChainID:   "bad-chain",
			WalletSignature: "sig",
		})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})

	t.Run("wallet lookup unexpected error", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), walletRepo, chainRepo)

		chainUUID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), "new2@mail.com").Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, "0xabc").Return(nil, errors.New("wallet repo down")).Once()

		_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{
			Email:           "new2@mail.com",
			Name:            "New2",
			Password:        "Password123!",
			WalletAddress:   "0xabc",
			WalletChainID:   "eip155:8453",
			WalletSignature: "sig",
		})
		assert.EqualError(t, err, "wallet repo down")
	})

	t.Run("wallet already linked to another user", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), walletRepo, chainRepo)

		chainUUID := uuid.New()
		existingUserID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), "new3@mail.com").Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, "0xabc").Return(&entities.Wallet{
			ID:      uuid.New(),
			UserID:  &existingUserID,
			ChainID: chainUUID,
			Address: "0xabc",
		}, nil).Once()

		_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{
			Email:           "new3@mail.com",
			Name:            "New3",
			Password:        "Password123!",
			WalletAddress:   "0xabc",
			WalletChainID:   "eip155:8453",
			WalletSignature: "sig",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), domainerrors.ErrAlreadyExists.Error())
	})

	t.Run("existing wallet without user continues and then user create fails", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		walletRepo := new(MockWalletRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), walletRepo, chainRepo)

		chainUUID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), "create-fail@mail.com").Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, "0xabc").Return(&entities.Wallet{
			ID:      uuid.New(),
			UserID:  nil,
			ChainID: chainUUID,
			Address: "0xabc",
		}, nil).Once()
		userRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.User")).Return(errors.New("create user failed")).Once()

		_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{
			Email:           "create-fail@mail.com",
			Name:            "Create Fail",
			Password:        "Password123!",
			WalletAddress:   "0xabc",
			WalletChainID:   "eip155:8453",
			WalletSignature: "sig",
		})
		assert.EqualError(t, err, "create user failed")
	})

	t.Run("wallet create fails after user create", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		walletRepo := new(MockWalletRepository)
		emailRepo := new(MockEmailVerificationRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

		chainUUID := uuid.New()
		createdUserID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), "wallet-fail@mail.com").Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, "0xabc").Return(nil, domainerrors.ErrNotFound).Once()
		userRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.User")).Return(nil).Run(func(args mock.Arguments) {
			u := args.Get(1).(*entities.User)
			u.ID = createdUserID
		}).Once()
		walletRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.Wallet")).Return(errors.New("wallet create failed")).Once()

		_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{
			Email:           "wallet-fail@mail.com",
			Name:            "Wallet Fail",
			Password:        "Password123!",
			WalletAddress:   "0xabc",
			WalletChainID:   "eip155:8453",
			WalletSignature: "sig",
		})
		assert.EqualError(t, err, "wallet create failed")
	})

	t.Run("email verification create fails", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		walletRepo := new(MockWalletRepository)
		emailRepo := new(MockEmailVerificationRepository)
		chainRepo := new(MockChainRepository)
		uc := newAuthUsecaseForTest(userRepo, emailRepo, walletRepo, chainRepo)

		chainUUID := uuid.New()
		createdUserID := uuid.New()
		userRepo.On("GetByEmail", context.Background(), "verify-fail@mail.com").Return(nil, domainerrors.ErrNotFound).Once()
		chainRepo.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{
			ID:      chainUUID,
			Type:    entities.ChainTypeEVM,
			ChainID: "8453",
		}, nil).Once()
		walletRepo.On("GetByAddress", context.Background(), chainUUID, "0xabc").Return(nil, domainerrors.ErrNotFound).Once()
		userRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.User")).Return(nil).Run(func(args mock.Arguments) {
			u := args.Get(1).(*entities.User)
			u.ID = createdUserID
		}).Once()
		walletRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.Wallet")).Return(nil).Once()
		emailRepo.On("Create", context.Background(), createdUserID, mock.AnythingOfType("string")).Return(errors.New("verification save failed")).Once()

		_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{
			Email:           "verify-fail@mail.com",
			Name:            "Verify Fail",
			Password:        "Password123!",
			WalletAddress:   "0xabc",
			WalletChainID:   "eip155:8453",
			WalletSignature: "sig",
		})
		assert.EqualError(t, err, "verification save failed")
	})
}

func TestAuthUsecase_VerifyEmail_MarkVerifiedError(t *testing.T) {
	userRepo := new(MockUserRepository)
	emailRepo := new(MockEmailVerificationRepository)
	uc := newAuthUsecaseForTest(userRepo, emailRepo, new(MockWalletRepository), new(MockChainRepository))

	emailRepo.On("GetByToken", context.Background(), "ok-token").Return(&entities.User{ID: uuid.New()}, nil).Once()
	emailRepo.On("MarkVerified", context.Background(), "ok-token").Return(errors.New("mark failed")).Once()

	err := uc.VerifyEmail(context.Background(), "ok-token")
	assert.EqualError(t, err, "mark failed")
}

func TestAuthUsecase_RefreshToken_UserLookupError(t *testing.T) {
	userRepo := new(MockUserRepository)
	uc := newAuthUsecaseForTest(userRepo, new(MockEmailVerificationRepository), new(MockWalletRepository), new(MockChainRepository))

	user := &entities.User{
		ID:    uuid.New(),
		Email: "refresh-err@mail.com",
		Role:  entities.UserRoleUser,
	}
	jwtSvc := jwt.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour)
	pair, genErr := jwtSvc.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	assert.NoError(t, genErr)

	userRepo.On("GetByID", context.Background(), user.ID).Return(nil, errors.New("user lookup failed")).Once()
	_, err := uc.RefreshToken(context.Background(), pair.RefreshToken)
	assert.EqualError(t, err, "user lookup failed")
}
