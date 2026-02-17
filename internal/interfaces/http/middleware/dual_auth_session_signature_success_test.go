package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/jwt"
	redispkg "pay-chain.backend/pkg/redis"
)

func TestDualAuthMiddleware_TrustedSession_WithOptionalSignatureValidationSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	mr, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable: %v", err)
	}
	defer mr.Close()
	cli := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	redispkg.SetClient(cli)
	defer cli.Close()
	sessionStore, err := redispkg.NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	userID := uuid.New()
	tokens, _ := jwtService.GenerateTokenPair(userID, "trusted@paychain.io", "USER")
	if err := sessionStore.CreateSession(context.Background(), "sid-verify", &redispkg.SessionData{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, time.Hour); err != nil {
		t.Fatalf("create session: %v", err)
	}

	secretKey := "sk_test"
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

	mockApiKeyRepo.On("FindByUserID", mock.Anything, userID).Return(activeKeys, nil).Once()
	mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	defer func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) }()
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, apiKeyUsecase, sessionStore))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("x-session-id", "sid-verify")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", timestamp)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
