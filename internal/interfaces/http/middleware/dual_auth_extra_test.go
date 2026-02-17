package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestDualAuthMiddleware_ExtraBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable: %v", err)
	}
	defer srv.Close()
	cli := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	redispkg.SetClient(cli)
	defer cli.Close()
	sessionStore, err := redispkg.NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, apiKeyUsecase, sessionStore))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	userID := uuid.New()
	tokens, _ := jwtService.GenerateTokenPair(userID, "session@paychain.io", "USER")
	if err := sessionStore.CreateSession(context.Background(), "sid-ok", &redispkg.SessionData{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, time.Hour); err != nil {
		t.Fatalf("create session: %v", err)
	}

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	defer func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) }()
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	// Session flow with only signature (without timestamp): should not trigger optional verification branch.
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("x-session-id", "sid-ok")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	req.Header.Set("X-Signature", "provided-without-timestamp")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// API key provided without signature/timestamp must fall through and fail auth.
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Api-Key", "pk_only")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")

	// JWT path signature validation fails due missing active keys.
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	req.Header.Set("X-Signature", "bad")
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	mockApiKeyRepo.On("FindByUserID", mock.Anything, userID).Return([]*entities.ApiKey{}, nil).Once()
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDualAuthMiddleware_SessionHeader_WithNilStore_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	defer func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) }()
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("x-session-id", "sid-any")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		r.ServeHTTP(w, req)
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDualAuthMiddleware_SessionTokenPrecedenceOverBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable: %v", err)
	}
	defer srv.Close()
	cli := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	redispkg.SetClient(cli)
	defer cli.Close()

	sessionStore, err := redispkg.NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	sessionUserID := uuid.New()
	sessionTokens, _ := jwtService.GenerateTokenPair(sessionUserID, "session@paychain.io", "USER")
	bearerTokens, _ := jwtService.GenerateTokenPair(uuid.New(), "bearer@paychain.io", "USER")
	if err := sessionStore.CreateSession(context.Background(), "sid-priority", &redispkg.SessionData{
		AccessToken:  sessionTokens.AccessToken,
		RefreshToken: sessionTokens.RefreshToken,
	}, time.Hour); err != nil {
		t.Fatalf("create session: %v", err)
	}

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	defer func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) }()
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, sessionStore))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("x-session-id", "sid-priority")
	req.Header.Set("Authorization", "Bearer "+bearerTokens.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDualAuthMiddleware_SessionInvalidToken_NoBearerFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable: %v", err)
	}
	defer srv.Close()
	cli := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	redispkg.SetClient(cli)
	defer cli.Close()

	sessionStore, err := redispkg.NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	sessionUserID := uuid.New()
	sessionTokens, _ := jwtService.GenerateTokenPair(sessionUserID, "session@paychain.io", "USER")
	bearerTokens, _ := jwtService.GenerateTokenPair(uuid.New(), "bearer@paychain.io", "USER")
	if err := sessionStore.CreateSession(context.Background(), "sid-priority-invalid", &redispkg.SessionData{
		AccessToken:  sessionTokens.AccessToken + "-broken",
		RefreshToken: sessionTokens.RefreshToken,
	}, time.Hour); err != nil {
		t.Fatalf("create session: %v", err)
	}

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	defer func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) }()
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, sessionStore))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("x-session-id", "sid-priority-invalid")
	req.Header.Set("Authorization", "Bearer "+bearerTokens.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

func TestDualAuthMiddleware_ExpiredToken_FromSessionFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", -1*time.Second, time.Hour)

	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable: %v", err)
	}
	defer srv.Close()
	cli := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	redispkg.SetClient(cli)
	defer cli.Close()

	sessionStore, err := redispkg.NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	userID := uuid.New()
	tokens, _ := jwtService.GenerateTokenPair(userID, "expired@session.io", "USER")
	if err := sessionStore.CreateSession(context.Background(), "sid-expired-dual", &redispkg.SessionData{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, time.Hour); err != nil {
		t.Fatalf("create session: %v", err)
	}

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	defer func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) }()
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, sessionStore))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("x-session-id", "sid-expired-dual")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Token has expired")
}

func TestDualAuthMiddleware_StrictMode_BlocksBearerFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	userID := uuid.New()
	tokens, _ := jwtService.GenerateTokenPair(userID, "strict@paychain.io", "USER")

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	defer func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) }()
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, nil, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestDualAuthMiddleware_TrustedSession_WithBodyAndOptionalSignatureValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockApiKeyRepo := new(MockApiKeyRepository)
	mockUserRepo := new(MockUserRepository)
	encryptionKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	apiKeyUsecase := usecases.NewApiKeyUsecase(mockApiKeyRepo, mockUserRepo, encryptionKey)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable: %v", err)
	}
	defer srv.Close()
	cli := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	redispkg.SetClient(cli)
	defer cli.Close()
	sessionStore, err := redispkg.NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	userID := uuid.New()
	tokens, _ := jwtService.GenerateTokenPair(userID, "trusted@paychain.io", "USER")
	if err := sessionStore.CreateSession(context.Background(), "sid-opt-sig", &redispkg.SessionData{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, time.Hour); err != nil {
		t.Fatalf("create session: %v", err)
	}

	secretKey := "sk_live_opt_sig"
	encryptedSecret, _ := encryptTest(secretKey, encryptionKey)
	mockApiKeyRepo.On("FindByUserID", mock.Anything, userID).Return([]*entities.ApiKey{
		{
			ID:              uuid.New(),
			UserID:          userID,
			SecretEncrypted: encryptedSecret,
			IsActive:        true,
		},
	}, nil).Once()
	mockApiKeyRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.ApiKey")).Return(nil).Once()

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	defer func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) }()
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	r := gin.New()
	r.Use(middleware.DualAuthMiddleware(jwtService, apiKeyUsecase, sessionStore))
	r.POST("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	body := `{"amount":"100"}`
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	bodyHash := sha256Hex([]byte(body))
	stringToSign := fmt.Sprintf("%s%s%s%s", timestamp, "POST", "/test", bodyHash)
	signature := hmacSha256Hex(secretKey, stringToSign)

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("x-session-id", "sid-opt-sig")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", timestamp)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
