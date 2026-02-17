package middleware

import (
	"context"
	"errors"
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
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
)

type internalApiKeyRepoStub struct{}

func (internalApiKeyRepoStub) Create(context.Context, *entities.ApiKey) error { return nil }
func (internalApiKeyRepoStub) FindByKeyHash(context.Context, string) (*entities.ApiKey, error) {
	return nil, errors.New("not found")
}
func (internalApiKeyRepoStub) FindByUserID(context.Context, uuid.UUID) ([]*entities.ApiKey, error) {
	return []*entities.ApiKey{}, nil
}
func (internalApiKeyRepoStub) FindByID(context.Context, uuid.UUID) (*entities.ApiKey, error) {
	return nil, errors.New("not found")
}
func (internalApiKeyRepoStub) Update(context.Context, *entities.ApiKey) error { return nil }
func (internalApiKeyRepoStub) Delete(context.Context, uuid.UUID) error        { return nil }

type internalUserRepoStub struct{}

func (internalUserRepoStub) Create(context.Context, *entities.User) error { return nil }
func (internalUserRepoStub) GetByID(context.Context, uuid.UUID) (*entities.User, error) {
	return nil, errors.New("not found")
}
func (internalUserRepoStub) GetByEmail(context.Context, string) (*entities.User, error) {
	return nil, errors.New("not found")
}
func (internalUserRepoStub) Update(context.Context, *entities.User) error            { return nil }
func (internalUserRepoStub) UpdatePassword(context.Context, uuid.UUID, string) error { return nil }
func (internalUserRepoStub) SoftDelete(context.Context, uuid.UUID) error             { return nil }
func (internalUserRepoStub) List(context.Context, string) ([]*entities.User, error)  { return nil, nil }

func TestAuthAndDualMiddleware_InternalCoveragePaths(t *testing.T) {
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
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) })

	t.Run("auth middleware trusted session and expired bearer", func(t *testing.T) {
		validJWT := jwt.NewJWTService("secret", time.Hour, time.Hour)
		pair, _ := validJWT.GenerateTokenPair(uuid.New(), "trusted@paychain.io", "USER")
		_ = sessionStore.CreateSession(context.Background(), "sid-internal-auth", &redis.SessionData{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		}, time.Minute)

		_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")
		r := gin.New()
		r.Use(AuthMiddleware(validJWT, sessionStore))
		r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		req.Header.Set("X-Session-Id", "sid-internal-auth")
		req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204 got %d", w.Code)
		}

		expiredJWT := jwt.NewJWTService("secret", -1*time.Second, time.Hour)
		expiredPair, _ := expiredJWT.GenerateTokenPair(uuid.New(), "expired@paychain.io", "USER")
		_ = os.Setenv("INTERNAL_PROXY_SECRET", "")
		r2 := gin.New()
		r2.Use(AuthMiddleware(expiredJWT, nil))
		r2.GET("/expired", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		req = httptest.NewRequest(http.MethodGet, "/expired", nil)
		req.Header.Set("Authorization", "Bearer "+expiredPair.AccessToken)
		w = httptest.NewRecorder()
		r2.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 got %d", w.Code)
		}
	})

	t.Run("dual auth trusted session body restore and optional signature branch", func(t *testing.T) {
		_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")
		j := jwt.NewJWTService("secret", time.Hour, time.Hour)
		pair, _ := j.GenerateTokenPair(uuid.New(), "dual@paychain.io", "USER")
		_ = sessionStore.CreateSession(context.Background(), "sid-internal-dual", &redis.SessionData{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		}, time.Minute)

		apiKeyUsecase := usecases.NewApiKeyUsecase(
			internalApiKeyRepoStub{},
			internalUserRepoStub{},
			"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
		)

		r := gin.New()
		r.Use(DualAuthMiddleware(j, apiKeyUsecase, sessionStore))
		r.POST("/dual", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodPost, "/dual", strings.NewReader(`{"a":1}`))
		req.Header.Set("x-session-id", "sid-internal-dual")
		req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
		req.Header.Set("X-Signature", "present")
		req.Header.Set("X-Timestamp", "1700000000")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		// Signature verification should fail, but it still exercises optional signature path.
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 got %d", w.Code)
		}
	})
}

func TestAuthAndDualMiddleware_InternalCoveragePaths_StrictBranchAssertions(t *testing.T) {
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

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) })

	t.Run("auth trusted session and expired bearer", func(t *testing.T) {
		validJWT := jwt.NewJWTService("secret", time.Hour, time.Hour)
		validPair, err := validJWT.GenerateTokenPair(uuid.New(), "trusted2@paychain.io", "USER")
		require.NoError(t, err)
		require.NoError(t, sessionStore.CreateSession(context.Background(), "sid-internal-auth-2", &redis.SessionData{
			AccessToken:  validPair.AccessToken,
			RefreshToken: validPair.RefreshToken,
		}, time.Minute))

		_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")
		r := gin.New()
		r.Use(AuthMiddleware(validJWT, sessionStore))
		r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		req.Header.Set("X-Session-Id", "sid-internal-auth-2")
		req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)

		expiredJWT := jwt.NewJWTService("secret", -1*time.Second, time.Hour)
		expiredPair, err := expiredJWT.GenerateTokenPair(uuid.New(), "expired2@paychain.io", "USER")
		require.NoError(t, err)

		_ = os.Setenv("INTERNAL_PROXY_SECRET", "")
		r2 := gin.New()
		r2.Use(AuthMiddleware(expiredJWT, nil))
		r2.GET("/expired", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		req = httptest.NewRequest(http.MethodGet, "/expired", nil)
		req.Header.Set("Authorization", "Bearer "+expiredPair.AccessToken)
		w = httptest.NewRecorder()
		r2.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "Token has expired")
	})

	t.Run("dual auth session with optional signature verification", func(t *testing.T) {
		_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")
		j := jwt.NewJWTService("secret", time.Hour, time.Hour)
		pair, err := j.GenerateTokenPair(uuid.New(), "dual2@paychain.io", "USER")
		require.NoError(t, err)
		require.NoError(t, sessionStore.CreateSession(context.Background(), "sid-internal-dual-2", &redis.SessionData{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		}, time.Minute))

		apiKeyUsecase := usecases.NewApiKeyUsecase(
			internalApiKeyRepoStub{},
			internalUserRepoStub{},
			"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
		)

		r := gin.New()
		r.Use(DualAuthMiddleware(j, apiKeyUsecase, sessionStore))
		r.POST("/dual", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		// Cover trusted-session path without optional verification.
		req := httptest.NewRequest(http.MethodPost, "/dual", strings.NewReader(`{"a":1}`))
		req.Header.Set("x-session-id", "sid-internal-dual-2")
		req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)

		// Cover optional verification branch (signature + timestamp present).
		req = httptest.NewRequest(http.MethodPost, "/dual", strings.NewReader(`{"a":2}`))
		req.Header.Set("x-session-id", "sid-internal-dual-2")
		req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
		req.Header.Set("X-Signature", "present")
		req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "Invalid Signature for JWT user")
	})
}
