package middleware_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/usecases"
)

func TestApiKeyPartnerMiddleware_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	mockMerchantRepo := new(MockMerchantRepository)

	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	userID := uuid.New()
	merchantID := uuid.New()
	apiKey := "pk_partner_test"
	secretKey := "sk_partner_test"
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
			Email: "merchant@example.com",
			Role:  entities.UserRoleUser,
		},
	}

	body := `{"invoice_currency":"IDR"}`
	bodyHash := sha256Hex([]byte(body))
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	stringToSign := fmt.Sprintf("%s.%s.%s.%s", timestamp, "POST", "/partner/quotes", bodyHash)
	signature := hmacSha256Hex(secretKey, stringToSign)

	mockApiKeyRepo.On("FindByKeyHash", mock.Anything, keyHash).Return(keyEntity, nil)
	mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockMerchantRepo.On("GetByUserID", mock.Anything, userID).Return(&entities.Merchant{ID: merchantID, UserID: userID}, nil)

	r := gin.New()
	r.Use(middleware.ApiKeyPartnerMiddleware(apiKeyUsecase, mockMerchantRepo))
	r.POST("/partner/quotes", func(c *gin.Context) {
		gotUserID, _ := c.Get(middleware.UserIDKey)
		gotMerchantID, _ := c.Get(middleware.MerchantIDKey)
		gotIsMerchant, _ := c.Get(middleware.IsMerchantAuthenticatedKey)
		c.JSON(http.StatusOK, gin.H{
			"userId":     gotUserID,
			"merchantId": gotMerchantID,
			"isMerchant": gotIsMerchant,
		})
	})

	req, _ := http.NewRequest("POST", "/partner/quotes", strings.NewReader(body))
	req.Header.Set(middleware.PartnerAPIKeyHeader, apiKey)
	req.Header.Set(middleware.PartnerAPITimestampHeader, timestamp)
	req.Header.Set(middleware.PartnerAPISignatureHeader, signature)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, userID.String(), resp["userId"])
	assert.Equal(t, merchantID.String(), resp["merchantId"])
	assert.Equal(t, true, resp["isMerchant"])
}

func TestApiKeyPartnerMiddleware_MissingHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.ApiKeyPartnerMiddleware(nil, new(MockMerchantRepository)))
	r.POST("/partner/quotes", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("POST", "/partner/quotes", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing partner auth headers")
}

func TestApiKeyPartnerMiddleware_RequiresMerchant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	mockMerchantRepo := new(MockMerchantRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	userID := uuid.New()
	apiKey := "pk_partner_test"
	secretKey := "sk_partner_test"
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
			Email: "merchant@example.com",
			Role:  entities.UserRoleUser,
		},
	}

	body := `{"invoice_currency":"IDR"}`
	bodyHash := sha256Hex([]byte(body))
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	stringToSign := fmt.Sprintf("%s.%s.%s.%s", timestamp, "POST", "/partner/quotes", bodyHash)
	signature := hmacSha256Hex(secretKey, stringToSign)

	mockApiKeyRepo.On("FindByKeyHash", mock.Anything, keyHash).Return(keyEntity, nil)
	mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockMerchantRepo.On("GetByUserID", mock.Anything, userID).Return(nil, fmt.Errorf("merchant not found"))

	r := gin.New()
	r.Use(middleware.ApiKeyPartnerMiddleware(apiKeyUsecase, mockMerchantRepo))
	r.POST("/partner/quotes", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("POST", "/partner/quotes", strings.NewReader(body))
	req.Header.Set(middleware.PartnerAPIKeyHeader, apiKey)
	req.Header.Set(middleware.PartnerAPITimestampHeader, timestamp)
	req.Header.Set(middleware.PartnerAPISignatureHeader, signature)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "merchant account required")
}

func TestApiKeyPartnerMiddleware_RejectsExpiredTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	mockMerchantRepo := new(MockMerchantRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	userID := uuid.New()
	apiKey := "pk_partner_test"
	secretKey := "sk_partner_test"
	keyHash := sha256Hex([]byte(apiKey))
	encryptedSecret, _ := encryptTest(secretKey, encryptionKey)

	keyEntity := &entities.ApiKey{
		ID:              uuid.New(),
		UserID:          userID,
		KeyHash:         keyHash,
		SecretEncrypted: encryptedSecret,
		IsActive:        true,
		User:            &entities.User{ID: userID, Email: "merchant@example.com", Role: entities.UserRoleUser},
	}

	body := `{"invoice_currency":"IDR"}`
	bodyHash := sha256Hex([]byte(body))
	timestamp := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
	stringToSign := fmt.Sprintf("%s.%s.%s.%s", timestamp, "POST", "/partner/quotes", bodyHash)
	signature := hmacSha256Hex(secretKey, stringToSign)

	mockApiKeyRepo.On("FindByKeyHash", mock.Anything, keyHash).Return(keyEntity, nil)

	r := gin.New()
	r.Use(middleware.ApiKeyPartnerMiddleware(apiKeyUsecase, mockMerchantRepo))
	r.POST("/partner/quotes", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("POST", "/partner/quotes", strings.NewReader(body))
	req.Header.Set(middleware.PartnerAPIKeyHeader, apiKey)
	req.Header.Set(middleware.PartnerAPITimestampHeader, timestamp)
	req.Header.Set(middleware.PartnerAPISignatureHeader, signature)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid partner API key or signature")
}

func TestApiKeyPartnerMiddleware_RejectsInvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	mockMerchantRepo := new(MockMerchantRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	userID := uuid.New()
	apiKey := "pk_partner_test"
	secretKey := "sk_partner_test"
	keyHash := sha256Hex([]byte(apiKey))
	encryptedSecret, _ := encryptTest(secretKey, encryptionKey)

	keyEntity := &entities.ApiKey{
		ID:              uuid.New(),
		UserID:          userID,
		KeyHash:         keyHash,
		SecretEncrypted: encryptedSecret,
		IsActive:        true,
		User:            &entities.User{ID: userID, Email: "merchant@example.com", Role: entities.UserRoleUser},
	}

	body := `{"invoice_currency":"IDR"}`
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	mockApiKeyRepo.On("FindByKeyHash", mock.Anything, keyHash).Return(keyEntity, nil)

	r := gin.New()
	r.Use(middleware.ApiKeyPartnerMiddleware(apiKeyUsecase, mockMerchantRepo))
	r.POST("/partner/quotes", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("POST", "/partner/quotes", strings.NewReader(body))
	req.Header.Set(middleware.PartnerAPIKeyHeader, apiKey)
	req.Header.Set(middleware.PartnerAPITimestampHeader, timestamp)
	req.Header.Set(middleware.PartnerAPISignatureHeader, hmacSha256Hex(secretKey, "wrong.payload"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid partner API key or signature")
}

func TestApiKeyPartnerMiddleware_RejectsTamperedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	mockMerchantRepo := new(MockMerchantRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)

	userID := uuid.New()
	apiKey := "pk_partner_test"
	secretKey := "sk_partner_test"
	keyHash := sha256Hex([]byte(apiKey))
	encryptedSecret, _ := encryptTest(secretKey, encryptionKey)

	keyEntity := &entities.ApiKey{
		ID:              uuid.New(),
		UserID:          userID,
		KeyHash:         keyHash,
		SecretEncrypted: encryptedSecret,
		IsActive:        true,
		User:            &entities.User{ID: userID, Email: "merchant@example.com", Role: entities.UserRoleUser},
	}

	signedBody := `{"invoice_currency":"IDR"}`
	actualBody := `{"invoice_currency":"USD"}`
	bodyHash := sha256Hex([]byte(signedBody))
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	stringToSign := fmt.Sprintf("%s.%s.%s.%s", timestamp, "POST", "/partner/quotes", bodyHash)
	signature := hmacSha256Hex(secretKey, stringToSign)

	mockApiKeyRepo.On("FindByKeyHash", mock.Anything, keyHash).Return(keyEntity, nil)

	r := gin.New()
	r.Use(middleware.ApiKeyPartnerMiddleware(apiKeyUsecase, mockMerchantRepo))
	r.POST("/partner/quotes", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("POST", "/partner/quotes", strings.NewReader(actualBody))
	req.Header.Set(middleware.PartnerAPIKeyHeader, apiKey)
	req.Header.Set(middleware.PartnerAPITimestampHeader, timestamp)
	req.Header.Set(middleware.PartnerAPISignatureHeader, signature)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid partner API key or signature")
}
