package usecases_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
)

func TestApiKeyUsecase_CreateApiKey(t *testing.T) {
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

	uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	userID := uuid.New()
	input := &entities.CreateApiKeyInput{
		Name:        "Test Key",
		Permissions: []string{"read", "write"},
	}

	ctx := context.Background()

	mockApiKeyRepo.On("Create", ctx, mock.AnythingOfType("*entities.ApiKey")).Return(nil)

	resp, err := uc.CreateApiKey(ctx, userID, input)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Test Key", resp.Name)
	assert.NotEmpty(t, resp.ApiKey)
	assert.NotEmpty(t, resp.SecretKey)

	mockApiKeyRepo.AssertExpectations(t)
}

func TestApiKeyUsecase_ValidateApiKey(t *testing.T) {
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

	uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	ctx := context.Background()
	userID := uuid.New()
	apiKey := "pk_live_test"
	secretKey := "sk_live_test"

	// Create a dummy key entity
	keyHash := sha256Hex([]byte(apiKey))

	// We need to encrypt the secretKey to mock the repository return
	encryptedSecret, _ := encryptSecret(secretKey, encryptionKey)

	keyEntity := &entities.ApiKey{
		ID:              uuid.New(),
		UserID:          userID,
		KeyHash:         keyHash,
		SecretEncrypted: encryptedSecret,
		IsActive:        true,
		User: &entities.User{
			ID:    userID,
			Email: "test@example.com",
			Role:  entities.UserRoleUser,
		},
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	method := "POST"
	path := "/v1/payments"
	body := `{"amount":"100"}`
	bodyHash := sha256Hex([]byte(body))

	stringToSign := fmt.Sprintf("%s%s%s%s", timestamp, method, path, bodyHash)
	signature := hmacSha256Hex(secretKey, stringToSign)

	mockApiKeyRepo.On("FindByKeyHash", ctx, keyHash).Return(keyEntity, nil)
	mockApiKeyRepo.On("Update", ctx, mock.AnythingOfType("*entities.ApiKey")).Return(nil)

	user, err := uc.ValidateApiKey(ctx, apiKey, signature, timestamp, method, path, bodyHash)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)

	mockApiKeyRepo.AssertExpectations(t)
}

func TestApiKeyUsecase_ValidateSignatureForJWT(t *testing.T) {
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

	uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	ctx := context.Background()
	userID := uuid.New()
	secretKey := "sk_live_test"

	encryptedSecret, _ := encryptSecret(secretKey, encryptionKey)

	activeKeys := []*entities.ApiKey{
		{
			ID:              uuid.New(),
			UserID:          userID,
			SecretEncrypted: encryptedSecret,
			IsActive:        true,
		},
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	method := "POST"
	path := "/v1/payments"
	bodyHash := sha256Hex([]byte(`{"amount":"100"}`))

	stringToSign := fmt.Sprintf("%s%s%s%s", timestamp, method, path, bodyHash)
	signature := hmacSha256Hex(secretKey, stringToSign)

	mockApiKeyRepo.On("FindByUserID", ctx, userID).Return(activeKeys, nil)
	mockApiKeyRepo.On("Update", ctx, mock.AnythingOfType("*entities.ApiKey")).Return(nil)

	err := uc.ValidateSignatureForJWT(ctx, userID, signature, timestamp, method, path, bodyHash)

	assert.NoError(t, err)

	mockApiKeyRepo.AssertExpectations(t)
}

func TestApiKeyUsecase_ListApiKeys(t *testing.T) {
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")

	ctx := context.Background()
	userID := uuid.New()

	expectedKeys := []*entities.ApiKey{
		{ID: uuid.New(), Name: "Key 1"},
		{ID: uuid.New(), Name: "Key 2"},
	}

	mockApiKeyRepo.On("FindByUserID", ctx, userID).Return(expectedKeys, nil)

	keys, err := uc.ListApiKeys(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, "Key 1", keys[0].Name)

	mockApiKeyRepo.AssertExpectations(t)
}

func TestApiKeyUsecase_RevokeApiKey(t *testing.T) {
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")

	ctx := context.Background()
	userID := uuid.New()
	keyID := uuid.New()

	keyEntity := &entities.ApiKey{
		ID:     keyID,
		UserID: userID,
	}

	mockApiKeyRepo.On("FindByID", ctx, keyID).Return(keyEntity, nil)
	mockApiKeyRepo.On("Delete", ctx, keyID).Return(nil)

	err := uc.RevokeApiKey(ctx, userID, keyID)

	assert.NoError(t, err)

	mockApiKeyRepo.AssertExpectations(t)
}

// Helpers copied/adapted from usecase to facilitate testing
func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSha256Hex(secret, data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// Minimal encryption helper for mocking (AES-GCM as used in usecase)
// This is a bit complex to replicate perfectly but we need it to mock the repository
// Alternatively we could change the usecase to take a Decrypter interface,
// but for simplicity we'll just implement a compatible encrypt here or
// use a very specific mock value.
// Let's use the actual encryption logic from the usecase if it was exported, but it's not.
// I'll just mock the decrypt return in the usecase if I could, but it calls u.decrypt internally.
// So I MUST provide a valid hex string that decrypts to secretKey.

func encryptSecret(plaintext string, keyHex string) (string, error) {
	// For testing, we can actually use the usecase's logic if we had access.
	// and would have stored the encrypted version in the repo.
	// But here we want to pre-load a mock response.
	// Since we can't call uc.encrypt, let's just make a small hack:
	// Use CreateApiKey to get an encrypted secret indirectly if possible?
	// No, it doesn't return the encrypted version.

	// Actually, I'll just use the same logic here.
	// Copy-pasting from api_key_usecase.go (minimal version)

	return encryptTest(plaintext, keyHex)
}

func encryptTest(plaintext string, keyHex string) (string, error) {
	key, _ := hex.DecodeString(keyHex)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}
