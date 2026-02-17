package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
)

func TestAuthMiddleware_BearerFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Minute, time.Hour)

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() {
		_ = os.Setenv("INTERNAL_PROXY_SECRET", prev)
	})
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")

	r := gin.New()
	r.Use(AuthMiddleware(jwtService, nil))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer invalid")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid token", func(t *testing.T) {
		pair, err := jwtService.GenerateTokenPair(uuid.New(), "u@paychain.io", "USER")
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestAuthMiddleware_StrictModeBlocksBearerFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Minute, time.Hour)

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() {
		_ = os.Setenv("INTERNAL_PROXY_SECRET", prev)
	})
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	pair, err := jwtService.GenerateTokenPair(uuid.New(), "u@paychain.io", "USER")
	require.NoError(t, err)

	r := gin.New()
	r.Use(AuthMiddleware(jwtService, nil))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_StrictModeSessionFlowAndExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Session flow in strict mode with trusted proxy headers.
	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable: %v", err)
	}
	defer srv.Close()

	cli := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	redis.SetClient(cli)
	defer cli.Close()

	sessionStore, err := redis.NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	jwtService := jwt.NewJWTService("secret", time.Minute, time.Hour)
	pair, err := jwtService.GenerateTokenPair(uuid.New(), "u@paychain.io", "USER")
	require.NoError(t, err)
	require.NoError(t, sessionStore.CreateSession(t.Context(), "sid-ok", &redis.SessionData{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}, time.Minute))

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prev) })
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	r := gin.New()
	r.Use(AuthMiddleware(jwtService, sessionStore))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	req.Header.Set("X-Session-Id", "sid-ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	// Expired token path in non-strict mode via Authorization header.
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")
	expiredJWT := jwt.NewJWTService("secret", -1*time.Second, time.Hour)
	expiredPair, err := expiredJWT.GenerateTokenPair(uuid.New(), "u@paychain.io", "USER")
	require.NoError(t, err)

	r2 := gin.New()
	r2.Use(AuthMiddleware(expiredJWT, sessionStore))
	r2.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredPair.AccessToken)
	w = httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Token has expired")
}
