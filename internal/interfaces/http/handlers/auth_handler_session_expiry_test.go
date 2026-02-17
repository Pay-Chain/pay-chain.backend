package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
)

func TestAuthHandler_GetSessionExpiry_NoSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_PROXY_SECRET", "secret")

	h := &AuthHandler{}
	r := gin.New()
	r.GET("/api/v1/auth/session-expiry", h.GetSessionExpiry)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session-expiry", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "No session")
}

func TestAuthHandler_GetSessionExpiry_InvalidProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_PROXY_SECRET", "secret")

	h := &AuthHandler{}
	r := gin.New()
	r.GET("/api/v1/auth/session-expiry", h.GetSessionExpiry)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session-expiry", nil)
	req.Header.Set("X-Session-Id", "sid-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "Invalid proxy request")
}

func TestAuthHandler_GetSessionExpiry_CookieFallbackAndInvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_PROXY_SECRET", "")

	userID := uuid.New()
	jwtSvc := jwt.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour)
	pair, err := jwtSvc.GenerateTokenPair(userID, "cookie@paychain.io", string(entities.UserRoleUser))
	require.NoError(t, err)

	h := NewAuthHandler(
		authServiceStub{
			registerFn:    func(context.Context, *entities.CreateUserInput) (*entities.User, string, error) { return nil, "", errors.New("unused") },
			loginFn:       func(context.Context, *entities.LoginInput) (*entities.AuthResponse, error) { return nil, errors.New("unused") },
			verifyEmailFn: func(context.Context, string) error { return errors.New("unused") },
			refreshTokenFn: func(context.Context, string) (*jwt.TokenPair, error) {
				return nil, errors.New("unused")
			},
			getUserByIDFn: func(context.Context, uuid.UUID) (*entities.User, error) { return nil, errors.New("unused") },
			getTokenExpFn: func(token string) (int64, error) {
				if token == pair.AccessToken {
					return 1710000000, nil
				}
				return 0, errors.New("invalid token")
			},
			changePassFn: func(context.Context, uuid.UUID, *entities.ChangePasswordInput) error { return errors.New("unused") },
		},
		sessionStoreStub{
			createFn: func(context.Context, string, *redis.SessionData, time.Duration) error { return nil },
			getFn: func(_ context.Context, sessionID string) (*redis.SessionData, error) {
				if sessionID == "cookie-sid-ok" {
					return &redis.SessionData{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken}, nil
				}
				return nil, errors.New("not found")
			},
			deleteFn: func(context.Context, string) error { return nil },
		},
	)

	r := gin.New()
	r.GET("/api/v1/auth/session-expiry", h.GetSessionExpiry)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session-expiry", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "cookie-sid-ok"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"exp\":1710000000")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/session-expiry", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "cookie-sid-missing"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Invalid session")
}
