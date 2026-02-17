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

func TestAuthMiddleware_SessionBranches(t *testing.T) {
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

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prev) })
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	r := gin.New()
	r.Use(AuthMiddleware(jwtService, sessionStore))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	// Session header exists but request not trusted.
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-any")
	req.Header.Set("X-Internal-Proxy-Secret", "wrong-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Trusted request but session not found (GetSession error path).
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-missing")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Non-strict mode with non-bearer Authorization should still fail.
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Basic abc")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Session with invalid token should hit invalid-token branch.
	pair, genErr := jwtService.GenerateTokenPair(uuid.New(), "user@x.com", "USER")
	require.NoError(t, genErr)
	require.NoError(t, sessionStore.CreateSession(t.Context(), "sid-invalid-token", &redis.SessionData{
		AccessToken:  pair.AccessToken + "-broken",
		RefreshToken: pair.RefreshToken,
	}, time.Minute))

	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-invalid-token")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Invalid token")
}

func TestAuthMiddleware_ExpiredToken_FromSessionFlow(t *testing.T) {
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

	jwtService := jwt.NewJWTService("secret", -1*time.Second, time.Hour)
	pair, genErr := jwtService.GenerateTokenPair(uuid.New(), "expired@x.com", "USER")
	require.NoError(t, genErr)
	require.NoError(t, sessionStore.CreateSession(t.Context(), "sid-expired-session", &redis.SessionData{
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
	req.Header.Set("X-Session-Id", "sid-expired-session")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Token has expired")
}

func TestAuthMiddleware_StrictMode_UntrustedSessionAndBearerBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := jwt.NewJWTService("secret", time.Hour, time.Hour*24)

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

	pair, genErr := jwtService.GenerateTokenPair(uuid.New(), "strict@x.com", "USER")
	require.NoError(t, genErr)
	require.NoError(t, sessionStore.CreateSession(t.Context(), "sid-strict-untrusted", &redis.SessionData{
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
	req.Header.Set("X-Session-Id", "sid-strict-untrusted")
	req.Header.Set("X-Internal-Proxy-Secret", "wrong-secret")
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Authentication required")
}

func TestAuthMiddleware_StrictMode_TrustedSessionSuccess_AndExpiredBearerBranch(t *testing.T) {
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

	validJWT := jwt.NewJWTService("secret", time.Hour, time.Hour*24)
	pair, genErr := validJWT.GenerateTokenPair(uuid.New(), "session-success@x.com", "USER")
	require.NoError(t, genErr)
	require.NoError(t, sessionStore.CreateSession(t.Context(), "sid-trusted-ok", &redis.SessionData{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}, time.Minute))

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prev) })
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	r := gin.New()
	r.Use(AuthMiddleware(validJWT, sessionStore))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	// Cover auth.go:36-40 (trusted session success path)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Session-Id", "sid-trusted-ok")
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	// Cover auth.go:62-67 (expired token branch) via bearer path in non-strict mode.
	expiredJWT := jwt.NewJWTService("secret", -1*time.Second, time.Hour)
	expiredPair, err := expiredJWT.GenerateTokenPair(uuid.New(), "expired-bearer@x.com", "USER")
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
