package usecases

import (
	"context"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

type apiKeyRepoMiniStub struct {
	createFn        func(context.Context, *entities.ApiKey) error
	findByKeyHashFn func(context.Context, string) (*entities.ApiKey, error)
	findByUserIDFn  func(context.Context, uuid.UUID) ([]*entities.ApiKey, error)
	findByIDFn      func(context.Context, uuid.UUID) (*entities.ApiKey, error)
	updateFn        func(context.Context, *entities.ApiKey) error
	deleteFn        func(context.Context, uuid.UUID) error
}

func (s *apiKeyRepoMiniStub) Create(ctx context.Context, apiKey *entities.ApiKey) error {
	if s.createFn != nil {
		return s.createFn(ctx, apiKey)
	}
	return nil
}
func (s *apiKeyRepoMiniStub) FindByKeyHash(ctx context.Context, keyHash string) (*entities.ApiKey, error) {
	if s.findByKeyHashFn != nil {
		return s.findByKeyHashFn(ctx, keyHash)
	}
	return nil, errors.New("not found")
}
func (s *apiKeyRepoMiniStub) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.ApiKey, error) {
	if s.findByUserIDFn != nil {
		return s.findByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (s *apiKeyRepoMiniStub) FindByID(ctx context.Context, id uuid.UUID) (*entities.ApiKey, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id)
	}
	return nil, errors.New("not found")
}
func (s *apiKeyRepoMiniStub) Update(ctx context.Context, apiKey *entities.ApiKey) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, apiKey)
	}
	return nil
}
func (s *apiKeyRepoMiniStub) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

type userRepoMiniStub struct{}

func (userRepoMiniStub) Create(context.Context, *entities.User) error                   { return nil }
func (userRepoMiniStub) GetByID(context.Context, uuid.UUID) (*entities.User, error)     { return &entities.User{ID: uuid.New()}, nil }
func (userRepoMiniStub) GetByEmail(context.Context, string) (*entities.User, error)      { return nil, errors.New("not found") }
func (userRepoMiniStub) Update(context.Context, *entities.User) error                    { return nil }
func (userRepoMiniStub) UpdatePassword(context.Context, uuid.UUID, string) error         { return nil }
func (userRepoMiniStub) SoftDelete(context.Context, uuid.UUID) error                     { return nil }
func (userRepoMiniStub) List(context.Context, string) ([]*entities.User, error)          { return nil, nil }

func TestApiKeyUsecase_CreateApiKey_RandomFailureBranches(t *testing.T) {
	validKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	repo := &apiKeyRepoMiniStub{}
	uc := NewApiKeyUsecase(repo, userRepoMiniStub{}, validKey)

	orig := apiKeyRandRead
	t.Cleanup(func() { apiKeyRandRead = orig })

	t.Run("first random call failed", func(t *testing.T) {
		apiKeyRandRead = func(_ []byte) (int, error) { return 0, errors.New("rng down") }
		_, err := uc.CreateApiKey(context.Background(), uuid.New(), &entities.CreateApiKeyInput{Name: "x"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to generate key")
	})

	t.Run("second random call failed", func(t *testing.T) {
		call := 0
		apiKeyRandRead = func(b []byte) (int, error) {
			call++
			if call == 2 {
				return 0, errors.New("rng down 2")
			}
			for i := range b {
				b[i] = byte(i + 1)
			}
			return len(b), nil
		}
		_, err := uc.CreateApiKey(context.Background(), uuid.New(), &entities.CreateApiKeyInput{Name: "y"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to generate secret")
	})
}

func TestApiKeyUsecase_ValidateApiKey_DecryptInvalidCipherKeyBranch(t *testing.T) {
	// invalid key on constructor path => decrypt hits aes.NewCipher key-size error branch
	uc := NewApiKeyUsecase(
		&apiKeyRepoMiniStub{
			findByKeyHashFn: func(context.Context, string) (*entities.ApiKey, error) {
				return &entities.ApiKey{ID: uuid.New(), UserID: uuid.New(), IsActive: true, SecretEncrypted: "00"}, nil
			},
		},
		userRepoMiniStub{},
		"invalid-encryption-key",
	)

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	_, err := uc.ValidateApiKey(context.Background(), "pk_live_abc", "sig", timestamp, "GET", "/x", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decrypt secret")
}

type errNonceReader struct{}

func (errNonceReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestApiKeyUsecase_EncryptDecrypt_ErrorBranches(t *testing.T) {
	validKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	uc := NewApiKeyUsecase(&apiKeyRepoMiniStub{}, userRepoMiniStub{}, validKey)

	t.Run("encrypt nonce read error", func(t *testing.T) {
		origReader := apiKeyRandReader
		t.Cleanup(func() { apiKeyRandReader = origReader })
		apiKeyRandReader = errNonceReader{}

		_, err := uc.encrypt("plain")
		require.Error(t, err)
	})

	t.Run("decrypt malformed ciphertext", func(t *testing.T) {
		_, err := uc.decrypt("00")
		require.Error(t, err)
		require.Contains(t, err.Error(), "malformed ciphertext")
	})

	t.Run("decrypt auth open error", func(t *testing.T) {
		enc, err := uc.encrypt("hello")
		require.NoError(t, err)

		ucOther := NewApiKeyUsecase(&apiKeyRepoMiniStub{}, userRepoMiniStub{}, "ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100")
		_, err = ucOther.decrypt(enc)
		require.Error(t, err)
	})

	t.Run("decrypt success", func(t *testing.T) {
		enc, err := uc.encrypt("roundtrip")
		require.NoError(t, err)
		plain, err := uc.decrypt(enc)
		require.NoError(t, err)
		require.Equal(t, "roundtrip", plain)
	})

	t.Run("encrypt invalid key size", func(t *testing.T) {
		ucBad := NewApiKeyUsecase(&apiKeyRepoMiniStub{}, userRepoMiniStub{}, "bad-key")
		_, err := ucBad.encrypt("plain")
		require.Error(t, err)
	})

	t.Run("encrypt gcm construction error", func(t *testing.T) {
		origNewGCM := newGCMCipher
		t.Cleanup(func() { newGCMCipher = origNewGCM })
		newGCMCipher = func(cipher.Block) (cipher.AEAD, error) {
			return nil, errors.New("gcm failed")
		}

		_, err := uc.encrypt("plain")
		require.Error(t, err)
		require.Contains(t, err.Error(), "gcm failed")
	})

	t.Run("decrypt invalid hex", func(t *testing.T) {
		_, err := uc.decrypt("zzzz")
		require.Error(t, err)
	})

	t.Run("decrypt invalid key size", func(t *testing.T) {
		ucBad := NewApiKeyUsecase(&apiKeyRepoMiniStub{}, userRepoMiniStub{}, "bad-key")
		_, err := ucBad.decrypt("00")
		require.Error(t, err)
	})

	t.Run("decrypt gcm construction error", func(t *testing.T) {
		origNewGCM := newGCMCipher
		t.Cleanup(func() { newGCMCipher = origNewGCM })
		newGCMCipher = func(cipher.Block) (cipher.AEAD, error) {
			return nil, errors.New("gcm failed decrypt")
		}

		_, err := uc.decrypt("000102030405060708090a0b0c0d0e0f")
		require.Error(t, err)
		require.Contains(t, err.Error(), "gcm failed decrypt")
	})
}
