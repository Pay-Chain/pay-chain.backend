package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/crypto"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/utils"
)

type authUserRepoStub struct {
	getByEmailFn     func(context.Context, string) (*entities.User, error)
	createFn         func(context.Context, *entities.User) error
	getByIDFn        func(context.Context, uuid.UUID) (*entities.User, error)
	updatePasswordFn func(context.Context, uuid.UUID, string) error
}

func (s *authUserRepoStub) Create(ctx context.Context, user *entities.User) error {
	if s.createFn != nil {
		return s.createFn(ctx, user)
	}
	return nil
}
func (s *authUserRepoStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *authUserRepoStub) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	if s.getByEmailFn != nil {
		return s.getByEmailFn(ctx, email)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *authUserRepoStub) Update(context.Context, *entities.User) error { return nil }
func (s *authUserRepoStub) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	if s.updatePasswordFn != nil {
		return s.updatePasswordFn(ctx, id, passwordHash)
	}
	return nil
}
func (s *authUserRepoStub) SoftDelete(context.Context, uuid.UUID) error            { return nil }
func (s *authUserRepoStub) List(context.Context, string) ([]*entities.User, error) { return nil, nil }

type authEmailRepoStub struct {
	createFn     func(context.Context, uuid.UUID, string) error
	getByTokenFn func(context.Context, string) (*entities.User, error)
	markVerifyFn func(context.Context, string) error
}

func (s *authEmailRepoStub) Create(ctx context.Context, userID uuid.UUID, token string) error {
	if s.createFn != nil {
		return s.createFn(ctx, userID, token)
	}
	return nil
}
func (s *authEmailRepoStub) GetByToken(ctx context.Context, token string) (*entities.User, error) {
	if s.getByTokenFn != nil {
		return s.getByTokenFn(ctx, token)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *authEmailRepoStub) MarkVerified(ctx context.Context, token string) error {
	if s.markVerifyFn != nil {
		return s.markVerifyFn(ctx, token)
	}
	return nil
}

type authWalletRepoStub struct {
	getByAddressFn func(context.Context, uuid.UUID, string) (*entities.Wallet, error)
	createFn       func(context.Context, *entities.Wallet) error
	getByUserIDFn  func(context.Context, uuid.UUID) ([]*entities.Wallet, error)
}

func (s *authWalletRepoStub) GetByAddress(ctx context.Context, chainID uuid.UUID, address string) (*entities.Wallet, error) {
	if s.getByAddressFn != nil {
		return s.getByAddressFn(ctx, chainID, address)
	}
	return nil, domainerrors.ErrNotFound
}
func (s *authWalletRepoStub) Create(ctx context.Context, wallet *entities.Wallet) error {
	if s.createFn != nil {
		return s.createFn(ctx, wallet)
	}
	return nil
}
func (s *authWalletRepoStub) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error) {
	if s.getByUserIDFn != nil {
		return s.getByUserIDFn(ctx, userID)
	}
	return []*entities.Wallet{}, nil
}
func (s *authWalletRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Wallet, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *authWalletRepoStub) Update(context.Context, *entities.Wallet) error         { return nil }
func (s *authWalletRepoStub) Delete(context.Context, uuid.UUID) error                { return nil }
func (s *authWalletRepoStub) SetPrimary(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (s *authWalletRepoStub) SoftDelete(context.Context, uuid.UUID) error            { return nil }

type authChainRepoStub struct {
	chain *entities.Chain
}

func (s *authChainRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Chain, error) {
	return s.chain, nil
}
func (s *authChainRepoStub) GetByChainID(context.Context, string) (*entities.Chain, error) {
	if s.chain == nil {
		return nil, domainerrors.ErrNotFound
	}
	return s.chain, nil
}
func (s *authChainRepoStub) GetByCAIP2(context.Context, string) (*entities.Chain, error) {
	if s.chain == nil {
		return nil, domainerrors.ErrNotFound
	}
	return s.chain, nil
}
func (s *authChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) { return nil, nil }
func (s *authChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *authChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *authChainRepoStub) Create(context.Context, *entities.Chain) error       { return nil }
func (s *authChainRepoStub) Update(context.Context, *entities.Chain) error       { return nil }
func (s *authChainRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }
func (s *authChainRepoStub) CreateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *authChainRepoStub) UpdateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *authChainRepoStub) DeleteRPC(context.Context, uuid.UUID) error          { return nil }
func (s *authChainRepoStub) GetRPCByID(context.Context, uuid.UUID) (*entities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}

func newAuthUsecaseHook(t *testing.T, userRepo *authUserRepoStub, emailRepo *authEmailRepoStub, walletRepo *authWalletRepoStub, chainRepo *authChainRepoStub) *AuthUsecase {
	t.Helper()
	jwtSvc := jwt.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour)
	return NewAuthUsecase(userRepo, emailRepo, walletRepo, chainRepo, jwtSvc)
}

func TestAuthUsecase_Hook_RegisterHashAndTokenErrors(t *testing.T) {
	chainID := uuid.New()
	chain := &entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}

	t.Run("hash password error", func(t *testing.T) {
		orig := authHashPassword
		t.Cleanup(func() { authHashPassword = orig })
		authHashPassword = func(string) (string, error) { return "", errors.New("hash fail") }

		uc := newAuthUsecaseHook(t,
			&authUserRepoStub{getByEmailFn: func(context.Context, string) (*entities.User, error) { return nil, domainerrors.ErrNotFound }},
			&authEmailRepoStub{},
			&authWalletRepoStub{getByAddressFn: func(context.Context, uuid.UUID, string) (*entities.Wallet, error) {
				return nil, domainerrors.ErrNotFound
			}},
			&authChainRepoStub{chain: chain},
		)

		_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{Email: "a@x.com", Name: "A", Password: "pw", WalletAddress: "0xabc", WalletChainID: "eip155:8453", WalletSignature: "sig"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "hash fail")
	})

	t.Run("generate verification token error", func(t *testing.T) {
		orig := authGenerateVerificationToken
		t.Cleanup(func() { authGenerateVerificationToken = orig })
		authGenerateVerificationToken = func() (string, error) { return "", errors.New("token fail") }

		uc := newAuthUsecaseHook(t,
			&authUserRepoStub{
				getByEmailFn: func(context.Context, string) (*entities.User, error) { return nil, domainerrors.ErrNotFound },
				createFn: func(_ context.Context, user *entities.User) error {
					user.ID = uuid.New()
					return nil
				},
			},
			&authEmailRepoStub{},
			&authWalletRepoStub{
				getByAddressFn: func(context.Context, uuid.UUID, string) (*entities.Wallet, error) {
					return nil, domainerrors.ErrNotFound
				},
				createFn: func(context.Context, *entities.Wallet) error { return nil },
			},
			&authChainRepoStub{chain: chain},
		)

		_, _, err := uc.Register(context.Background(), &entities.CreateUserInput{Email: "b@x.com", Name: "B", Password: "pw", WalletAddress: "0xabc", WalletChainID: "eip155:8453", WalletSignature: "sig"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "token fail")
	})
}

func TestAuthUsecase_Hook_LoginBranches(t *testing.T) {
	hashed, err := crypto.HashPassword("correct-password")
	require.NoError(t, err)
	user := &entities.User{ID: uuid.New(), Email: "u@x.com", PasswordHash: hashed, Role: entities.UserRoleUser}
	chain := &entities.Chain{ID: uuid.New(), ChainID: "8453", Type: entities.ChainTypeEVM}

	t.Run("generate token pair error", func(t *testing.T) {
		orig := authGenerateTokenPair
		t.Cleanup(func() { authGenerateTokenPair = orig })
		authGenerateTokenPair = func(*jwt.JWTService, uuid.UUID, string, string) (*jwt.TokenPair, error) {
			return nil, errors.New("token pair fail")
		}

		uc := newAuthUsecaseHook(t,
			&authUserRepoStub{getByEmailFn: func(context.Context, string) (*entities.User, error) { return user, nil }},
			&authEmailRepoStub{},
			&authWalletRepoStub{},
			&authChainRepoStub{chain: chain},
		)

		_, err := uc.Login(context.Background(), &entities.LoginInput{Email: user.Email, Password: "correct-password"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "token pair fail")
	})

	t.Run("session json marshal error", func(t *testing.T) {
		origMarshal := authJSONMarshal
		origSet := authRedisSet
		t.Cleanup(func() {
			authJSONMarshal = origMarshal
			authRedisSet = origSet
		})
		authJSONMarshal = func(v interface{}) ([]byte, error) {
			_ = v
			return nil, errors.New("marshal fail")
		}
		authRedisSet = func(context.Context, string, interface{}, time.Duration) error { return nil }

		uc := newAuthUsecaseHook(t,
			&authUserRepoStub{getByEmailFn: func(context.Context, string) (*entities.User, error) { return user, nil }},
			&authEmailRepoStub{},
			&authWalletRepoStub{},
			&authChainRepoStub{chain: chain},
		)

		_, err := uc.Login(context.Background(), &entities.LoginInput{Email: user.Email, Password: "correct-password", UseSession: true})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to marshal session data")
	})

	t.Run("session success branch without redis server", func(t *testing.T) {
		origMarshal := authJSONMarshal
		origSet := authRedisSet
		t.Cleanup(func() {
			authJSONMarshal = origMarshal
			authRedisSet = origSet
		})
		authJSONMarshal = func(v interface{}) ([]byte, error) { return []byte(`{"ok":true}`), nil }
		authRedisSet = func(context.Context, string, interface{}, time.Duration) error { return nil }

		uc := newAuthUsecaseHook(t,
			&authUserRepoStub{getByEmailFn: func(context.Context, string) (*entities.User, error) { return user, nil }},
			&authEmailRepoStub{},
			&authWalletRepoStub{},
			&authChainRepoStub{chain: chain},
		)

		resp, err := uc.Login(context.Background(), &entities.LoginInput{Email: user.Email, Password: "correct-password", UseSession: true})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, strings.TrimSpace(resp.SessionID) != "")
		require.Equal(t, user.ID, resp.User.ID)
	})
}
