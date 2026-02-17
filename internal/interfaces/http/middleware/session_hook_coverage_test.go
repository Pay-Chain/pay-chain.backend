package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
)

func TestAuthMiddleware_SessionHookAndExpiredBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origLoadSession := loadSessionFromStore
	origSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() {
		loadSessionFromStore = origLoadSession
		_ = os.Setenv("INTERNAL_PROXY_SECRET", origSecret)
	})

	validJWT := jwt.NewJWTService("secret", time.Hour, time.Hour)
	validPair, err := validJWT.GenerateTokenPair(uuid.New(), "session-hook@paychain.io", "USER")
	require.NoError(t, err)

	loadSessionFromStore = func(context.Context, *redis.SessionStore, string) (*redis.SessionData, error) {
		return &redis.SessionData{AccessToken: validPair.AccessToken}, nil
	}

	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")
	r := gin.New()
	r.Use(AuthMiddleware(validJWT, &redis.SessionStore{}))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-hook")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	expiredJWT := jwt.NewJWTService("secret", -1*time.Second, time.Hour)
	expiredPair, err := expiredJWT.GenerateTokenPair(uuid.New(), "expired-hook@paychain.io", "USER")
	require.NoError(t, err)
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")

	r2 := gin.New()
	r2.Use(AuthMiddleware(expiredJWT, nil))
	r2.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredPair.AccessToken)
	w = httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Token has expired")
}

func TestDualAuthMiddleware_SessionHookAndOptionalSignatureBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origLoadSession := loadSessionFromStore
	origSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() {
		loadSessionFromStore = origLoadSession
		_ = os.Setenv("INTERNAL_PROXY_SECRET", origSecret)
	})

	j := jwt.NewJWTService("secret", time.Hour, time.Hour)
	pair, err := j.GenerateTokenPair(uuid.New(), "dual-session-hook@paychain.io", "USER")
	require.NoError(t, err)

	loadSessionFromStore = func(context.Context, *redis.SessionStore, string) (*redis.SessionData, error) {
		return &redis.SessionData{AccessToken: pair.AccessToken}, nil
	}

	apiKeyUsecase := usecases.NewApiKeyUsecase(
		internalApiKeyRepoStub{},
		internalUserRepoStub{},
		"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
	)

	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")
	r := gin.New()
	r.Use(DualAuthMiddleware(j, apiKeyUsecase, &redis.SessionStore{}))
	r.POST("/dual", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	// Cover trusted-session path and successful body restore.
	req := httptest.NewRequest(http.MethodPost, "/dual", strings.NewReader(`{"a":1}`))
	req.Header.Set("x-session-id", "sid-hook")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	// Cover optional signature verification branch in trusted-session flow.
	req = httptest.NewRequest(http.MethodPost, "/dual", strings.NewReader(`{"a":2}`))
	req.Header.Set("x-session-id", "sid-hook")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	req.Header.Set("X-Signature", "bad-signature")
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Invalid Signature for JWT user")
}
