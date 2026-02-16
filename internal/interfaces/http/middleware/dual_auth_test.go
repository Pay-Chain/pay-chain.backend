package middleware_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/jwt"
)

type errorReadCloser struct{}

func (errorReadCloser) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (errorReadCloser) Close() error             { return nil }

// Mock objects already defined in usecases_test can be used if they were exported,
// but they are in different packages. I'll define minimal ones here or use the ones from usecases package if I can.
// Actually, I'll just redefine minimal mocks here for repo.

type MockApiKeyRepository struct{ mock.Mock }

func (m *MockApiKeyRepository) Create(ctx context.Context, k *entities.ApiKey) error {
	return m.Called(ctx, k).Error(0)
}
func (m *MockApiKeyRepository) FindByKeyHash(ctx context.Context, h string) (*entities.ApiKey, error) {
	args := m.Called(ctx, h)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ApiKey), args.Error(1)
}
func (m *MockApiKeyRepository) FindByUserID(ctx context.Context, id uuid.UUID) ([]*entities.ApiKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.ApiKey), args.Error(1)
}
func (m *MockApiKeyRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.ApiKey, error) {
	return nil, nil
}
func (m *MockApiKeyRepository) Update(ctx context.Context, k *entities.ApiKey) error {
	return m.Called(ctx, k).Error(0)
}
func (m *MockApiKeyRepository) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type MockUserRepository struct{ mock.Mock }

func (m *MockUserRepository) Create(ctx context.Context, u *entities.User) error { return nil }
func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	return nil, nil
}
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	return nil, nil
}
func (m *MockUserRepository) Update(ctx context.Context, u *entities.User) error { return nil }
func (m *MockUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return nil
}
func (m *MockUserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockUserRepository) List(ctx context.Context, s string) ([]*entities.User, error) {
	return nil, nil
}

func TestDualAuthMiddleware_ApiKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)
	// sessionStore would need redis, let's keep it nil for now if the middleware handles it

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, apiKeyUsecase, nil))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get(middleware.UserIDKey)
		c.JSON(http.StatusOK, gin.H{"userId": userID})
	})

	userID := uuid.New()
	apiKey := "pk_test"
	secretKey := "sk_test"
	keyHash := sha256Hex([]byte(apiKey))

	encryptedSecret, _ := encryptTest(secretKey, encryptionKey)

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
	body := ""
	bodyHash := sha256Hex([]byte(body))
	stringToSign := fmt.Sprintf("%s%s%s%s", timestamp, "GET", "/test", bodyHash)
	signature := hmacSha256Hex(secretKey, stringToSign)

	mockApiKeyRepo.On("FindByKeyHash", mock.Anything, keyHash).Return(keyEntity, nil)
	mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", timestamp)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), userID.String())
}

func TestDualAuthMiddleware_JWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, nil))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get(middleware.UserIDKey)
		c.JSON(http.StatusOK, gin.H{"userId": userID})
	})

	userID := uuid.New()
	email := "test@example.com"
	role := "USER"

	tokens, _ := jwtService.GenerateTokenPair(userID, email, role)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Signature and timestamp are required")
}

func TestDualAuthMiddleware_JWTWithSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, apiKeyUsecase, nil))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get(middleware.UserIDKey)
		c.JSON(http.StatusOK, gin.H{"userId": userID})
	})

	userID := uuid.New()
	email := "test@example.com"
	role := "USER"
	secretKey := "sk_test"

	tokens, _ := jwtService.GenerateTokenPair(userID, email, role)

	encryptedSecret, _ := encryptTest(secretKey, encryptionKey)
	activeKeys := []*entities.ApiKey{
		{
			ID:              uuid.New(),
			UserID:          userID,
			SecretEncrypted: encryptedSecret,
			IsActive:        true,
		},
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	bodyHash := sha256Hex([]byte(""))
	stringToSign := fmt.Sprintf("%s%s%s%s", timestamp, "GET", "/test", bodyHash)
	signature := hmacSha256Hex(secretKey, stringToSign)

	mockApiKeyRepo.On("FindByUserID", mock.Anything, userID).Return(activeKeys, nil)
	mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", timestamp)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), userID.String())
}

func TestDualAuthMiddleware_RequestBodyReadError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, nil))
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	req.Body = errorReadCloser{}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to read request body")
}

func TestDualAuthMiddleware_ApiKeyInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, apiKeyUsecase, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	apiKey := "pk_bad"
	keyHash := sha256Hex([]byte(apiKey))
	mockApiKeyRepo.On("FindByKeyHash", mock.Anything, keyHash).Return(nil, errors.New("not found")).Once()

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("X-Signature", "bad")
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid API Key or Signature")
}

func TestDualAuthMiddleware_JWTInvalidToken_AndNoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")

	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestDualAuthMiddleware_StrictSessionModeWithoutTrustedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "secret")
	defer os.Unsetenv("INTERNAL_PROXY_SECRET")

	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)
	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	userID := uuid.New()
	tokens, _ := jwtService.GenerateTokenPair(userID, "strict@test.com", "USER")

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestDualAuthMiddleware_JWTExpired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", -1*time.Second, time.Hour)

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	userID := uuid.New()
	tokens, _ := jwtService.GenerateTokenPair(userID, "expired@test.com", "USER")
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Token has expired")
}

func TestDualAuthMiddleware_JWTInvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, apiKeyUsecase, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	userID := uuid.New()
	tokens, _ := jwtService.GenerateTokenPair(userID, "sig@test.com", "USER")
	mockApiKeyRepo.On("FindByUserID", mock.Anything, userID).Return([]*entities.ApiKey{}, nil).Once()

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	req.Header.Set("X-Signature", "bad-signature")
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid Signature for JWT user")
}

// Helpers
func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSha256Hex(secret, data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
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
