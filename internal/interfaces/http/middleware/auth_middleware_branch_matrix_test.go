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

func TestAuthMiddleware_SessionAndBearerBranchMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

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

	userID := uuid.New()
	pair, err := jwtService.GenerateTokenPair(userID, "matrix@paychain.io", "USER")
	require.NoError(t, err)

	// session with empty access token to force bearer fallback when non-strict.
	require.NoError(t, sessionStore.CreateSession(t.Context(), "sid-empty", &redis.SessionData{
		AccessToken:  "",
		RefreshToken: pair.RefreshToken,
	}, time.Minute))

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prev) })
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")

	r := gin.New()
	r.Use(AuthMiddleware(jwtService, sessionStore))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	// Trusted session id found but access token empty -> fallback to bearer in non-strict mode.
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-empty")
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	// Same empty session without bearer -> unauthorized.
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-empty")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_SessionHeader_WithNilStore_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Minute, time.Hour)

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prev) })
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	r := gin.New()
	r.Use(AuthMiddleware(jwtService, nil))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-any")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	w := httptest.NewRecorder()

	require.NotPanics(t, func() {
		r.ServeHTTP(w, req)
	})
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_SessionTokenPrecedenceOverBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Minute, time.Hour)

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

	userID := uuid.New()
	sessionPair, err := jwtService.GenerateTokenPair(userID, "session@paychain.io", "USER")
	require.NoError(t, err)
	bearerPair, err := jwtService.GenerateTokenPair(uuid.New(), "bearer@paychain.io", "USER")
	require.NoError(t, err)

	require.NoError(t, sessionStore.CreateSession(t.Context(), "sid-precedence", &redis.SessionData{
		AccessToken:  sessionPair.AccessToken,
		RefreshToken: sessionPair.RefreshToken,
	}, time.Minute))

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prev) })
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")

	r := gin.New()
	r.Use(AuthMiddleware(jwtService, sessionStore))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-precedence")
	req.Header.Set("Authorization", "Bearer "+bearerPair.AccessToken+"-invalid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthMiddleware_SessionInvalidToken_NoBearerFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Minute, time.Hour)

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

	userID := uuid.New()
	sessionPair, err := jwtService.GenerateTokenPair(userID, "session@paychain.io", "USER")
	require.NoError(t, err)
	bearerPair, err := jwtService.GenerateTokenPair(uuid.New(), "bearer@paychain.io", "USER")
	require.NoError(t, err)

	require.NoError(t, sessionStore.CreateSession(t.Context(), "sid-invalid-precedence", &redis.SessionData{
		AccessToken:  sessionPair.AccessToken + "-tampered",
		RefreshToken: sessionPair.RefreshToken,
	}, time.Minute))

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prev) })
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")

	r := gin.New()
	r.Use(AuthMiddleware(jwtService, sessionStore))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-invalid-precedence")
	req.Header.Set("Authorization", "Bearer "+bearerPair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Invalid token")
}
