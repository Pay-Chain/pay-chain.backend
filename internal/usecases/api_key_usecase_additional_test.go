package usecases_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
)

func TestApiKeyUsecase_NewAndCreate_Branches(t *testing.T) {
	t.Run("constructor supports raw 32-byte key fallback", func(t *testing.T) {
		mockApiKeyRepo := new(MockApiKeyRepository)
		mockUserRepo := new(MockUserRepository)
		raw32 := "abcdefghijklmnopqrstuvwxyzABCDEF" // 32 chars
		uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, raw32)

		userID := uuid.New()
		input := &entities.CreateApiKeyInput{Name: "Raw Key", Permissions: []string{"read"}}
		mockApiKeyRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.ApiKey")).Return(nil).Once()
		resp, err := uc.CreateApiKey(context.Background(), userID, input)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("create fails when encryption key invalid", func(t *testing.T) {
		mockApiKeyRepo := new(MockApiKeyRepository)
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, "bad-key")
		_, err := uc.CreateApiKey(context.Background(), uuid.New(), &entities.CreateApiKeyInput{Name: "x"})
		require.Error(t, err)
	})
}

func TestApiKeyUsecase_ValidateApiKey_ErrorBranches(t *testing.T) {
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)
	ctx := context.Background()

	t.Run("invalid timestamp", func(t *testing.T) {
		_, err := uc.ValidateApiKey(ctx, "pk", "sig", "not-int", "GET", "/v1/x", "hash")
		require.Error(t, err)
	})

	t.Run("expired timestamp", func(t *testing.T) {
		oldTs := fmt.Sprintf("%d", time.Now().Unix()-1000)
		_, err := uc.ValidateApiKey(ctx, "pk", "sig", oldTs, "GET", "/v1/x", "hash")
		require.Error(t, err)
	})

	t.Run("key not found", func(t *testing.T) {
		apiKey := "pk_live_not_found"
		ts := fmt.Sprintf("%d", time.Now().Unix())
		mockApiKeyRepo.On("FindByKeyHash", ctx, sha256Hex([]byte(apiKey))).Return((*entities.ApiKey)(nil), errors.New("not found")).Once()
		_, err := uc.ValidateApiKey(ctx, apiKey, "sig", ts, "GET", "/v1/x", "hash")
		require.Error(t, err)
	})

	t.Run("inactive key", func(t *testing.T) {
		apiKey := "pk_live_inactive"
		ts := fmt.Sprintf("%d", time.Now().Unix())
		mockApiKeyRepo.On("FindByKeyHash", ctx, sha256Hex([]byte(apiKey))).Return(&entities.ApiKey{IsActive: false}, nil).Once()
		_, err := uc.ValidateApiKey(ctx, apiKey, "sig", ts, "GET", "/v1/x", "hash")
		require.Error(t, err)
	})

	t.Run("decrypt failure", func(t *testing.T) {
		apiKey := "pk_live_decrypt_fail"
		ts := fmt.Sprintf("%d", time.Now().Unix())
		mockApiKeyRepo.On("FindByKeyHash", ctx, sha256Hex([]byte(apiKey))).Return(&entities.ApiKey{
			IsActive:        true,
			SecretEncrypted: "not-hex",
		}, nil).Once()
		_, err := uc.ValidateApiKey(ctx, apiKey, "sig", ts, "GET", "/v1/x", "hash")
		require.Error(t, err)
	})

	t.Run("invalid signature", func(t *testing.T) {
		apiKey := "pk_live_sig_fail"
		secretKey := "sk_live_test_sig_fail"
		enc, _ := encryptSecret(secretKey, encryptionKey)
		ts := fmt.Sprintf("%d", time.Now().Unix())
		mockApiKeyRepo.On("FindByKeyHash", ctx, sha256Hex([]byte(apiKey))).Return(&entities.ApiKey{
			IsActive:        true,
			SecretEncrypted: enc,
		}, nil).Once()
		_, err := uc.ValidateApiKey(ctx, apiKey, "bad-signature", ts, "POST", "/v1/payments", "body-hash")
		require.Error(t, err)
	})

	t.Run("user preload missing and fetch user failed", func(t *testing.T) {
		apiKey := "pk_live_user_fetch_fail"
		secretKey := "sk_live_user_fetch_fail"
		enc, _ := encryptSecret(secretKey, encryptionKey)
		ts := fmt.Sprintf("%d", time.Now().Unix())
		method := "POST"
		path := "/v1/payments"
		bodyHash := sha256Hex([]byte(`{"x":1}`))
		signature := hmacSha256Hex(secretKey, fmt.Sprintf("%s%s%s%s", ts, method, path, bodyHash))
		userID := uuid.New()

		mockApiKeyRepo.On("FindByKeyHash", ctx, sha256Hex([]byte(apiKey))).Return(&entities.ApiKey{
			IsActive:        true,
			SecretEncrypted: enc,
			UserID:          userID,
			User:            nil,
		}, nil).Once()
		mockApiKeyRepo.On("Update", ctx, mock.AnythingOfType("*entities.ApiKey")).Return(nil).Once()
		mockUserRepo.On("GetByID", ctx, userID).Return((*entities.User)(nil), errors.New("not found")).Once()

		_, err := uc.ValidateApiKey(ctx, apiKey, signature, ts, method, path, bodyHash)
		require.Error(t, err)
	})

	t.Run("user preload missing and fetch user success", func(t *testing.T) {
		apiKey := "pk_live_user_fetch_ok"
		secretKey := "sk_live_user_fetch_ok"
		enc, _ := encryptSecret(secretKey, encryptionKey)
		ts := fmt.Sprintf("%d", time.Now().Unix())
		method := "POST"
		path := "/v1/payments"
		bodyHash := sha256Hex([]byte(`{"x":2}`))
		signature := hmacSha256Hex(secretKey, fmt.Sprintf("%s%s%s%s", ts, method, path, bodyHash))
		userID := uuid.New()
		user := &entities.User{ID: userID, Email: "ok@paychain.io"}

		mockApiKeyRepo.On("FindByKeyHash", ctx, sha256Hex([]byte(apiKey))).Return(&entities.ApiKey{
			IsActive:        true,
			SecretEncrypted: enc,
			UserID:          userID,
			User:            nil,
		}, nil).Once()
		mockApiKeyRepo.On("Update", ctx, mock.AnythingOfType("*entities.ApiKey")).Return(nil).Once()
		mockUserRepo.On("GetByID", ctx, userID).Return(user, nil).Once()

		resolvedUser, err := uc.ValidateApiKey(ctx, apiKey, signature, ts, method, path, bodyHash)
		require.NoError(t, err)
		require.NotNil(t, resolvedUser)
		require.Equal(t, userID, resolvedUser.ID)
	})
}

func TestApiKeyUsecase_ValidateSignatureForJWT_ErrorBranches(t *testing.T) {
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)
	ctx := context.Background()
	userID := uuid.New()

	err := uc.ValidateSignatureForJWT(ctx, userID, "sig", "not-int", "GET", "/x", "h")
	require.Error(t, err)

	oldTs := fmt.Sprintf("%d", time.Now().Unix()-1000)
	err = uc.ValidateSignatureForJWT(ctx, userID, "sig", oldTs, "GET", "/x", "h")
	require.Error(t, err)

	ts := fmt.Sprintf("%d", time.Now().Unix())
	mockApiKeyRepo.On("FindByUserID", ctx, userID).Return(([]*entities.ApiKey)(nil), errors.New("db failed")).Once()
	err = uc.ValidateSignatureForJWT(ctx, userID, "sig", ts, "GET", "/x", "h")
	require.Error(t, err)

	mockApiKeyRepo.On("FindByUserID", ctx, userID).Return([]*entities.ApiKey{}, nil).Once()
	err = uc.ValidateSignatureForJWT(ctx, userID, "sig", ts, "GET", "/x", "h")
	require.Error(t, err)

	inactive := []*entities.ApiKey{{IsActive: false}}
	mockApiKeyRepo.On("FindByUserID", ctx, userID).Return(inactive, nil).Once()
	err = uc.ValidateSignatureForJWT(ctx, userID, "sig", ts, "GET", "/x", "h")
	require.Error(t, err)

	badDecrypt := []*entities.ApiKey{{IsActive: true, SecretEncrypted: "not-hex"}}
	mockApiKeyRepo.On("FindByUserID", ctx, userID).Return(badDecrypt, nil).Once()
	err = uc.ValidateSignatureForJWT(ctx, userID, "sig", ts, "GET", "/x", "h")
	require.Error(t, err)
}

func TestApiKeyUsecase_RevokeApiKey_Branches(t *testing.T) {
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	ctx := context.Background()
	userID := uuid.New()
	keyID := uuid.New()

	mockApiKeyRepo.On("FindByID", ctx, keyID).Return((*entities.ApiKey)(nil), errors.New("not found")).Once()
	err := uc.RevokeApiKey(ctx, userID, keyID)
	assert.Error(t, err)

	anotherUserID := uuid.New()
	mockApiKeyRepo.On("FindByID", ctx, keyID).Return(&entities.ApiKey{ID: keyID, UserID: anotherUserID}, nil).Once()
	err = uc.RevokeApiKey(ctx, userID, keyID)
	assert.Error(t, err)
}

