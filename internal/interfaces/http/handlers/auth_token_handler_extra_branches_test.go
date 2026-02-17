package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
)

func TestAuthHandler_RefreshToken_UsesLegacyCookieSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_PROXY_SECRET", "")

	var seenSessionID string
	h := NewAuthHandler(
		authServiceStub{
			registerFn:    func(context.Context, *entities.CreateUserInput) (*entities.User, string, error) { return nil, "", errors.New("unused") },
			loginFn:       func(context.Context, *entities.LoginInput) (*entities.AuthResponse, error) { return nil, errors.New("unused") },
			verifyEmailFn: func(context.Context, string) error { return errors.New("unused") },
			refreshTokenFn: func(_ context.Context, refreshToken string) (*jwt.TokenPair, error) {
				if refreshToken != "ok-refresh" {
					return nil, errors.New("unexpected refresh token")
				}
				return &jwt.TokenPair{AccessToken: "new-a", RefreshToken: "new-r"}, nil
			},
			getUserByIDFn: func(context.Context, uuid.UUID) (*entities.User, error) { return nil, errors.New("unused") },
			getTokenExpFn: func(string) (int64, error) { return 0, errors.New("unused") },
			changePassFn:  func(context.Context, uuid.UUID, *entities.ChangePasswordInput) error { return errors.New("unused") },
		},
		sessionStoreStub{
			createFn: func(_ context.Context, sessionID string, data *redis.SessionData, _ time.Duration) error {
				seenSessionID = sessionID
				if data.AccessToken == "" || data.RefreshToken == "" {
					return errors.New("missing tokens")
				}
				return nil
			},
			getFn:    func(context.Context, string) (*redis.SessionData, error) { return nil, errors.New("unused") },
			deleteFn: func(context.Context, string) error { return nil },
		},
	)

	r := gin.New()
	r.POST("/auth/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(`{"refreshToken":"ok-refresh"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "cookie-session-123"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if seenSessionID != "cookie-session-123" {
		t.Fatalf("expected session_id from cookie, got %q", seenSessionID)
	}
}

func TestAuthHandler_RefreshToken_StrictMode_UntrustedProxySkipsSessionLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_PROXY_SECRET", "strict-secret")

	h := NewAuthHandler(
		authServiceStub{
			registerFn:    func(context.Context, *entities.CreateUserInput) (*entities.User, string, error) { return nil, "", errors.New("unused") },
			loginFn:       func(context.Context, *entities.LoginInput) (*entities.AuthResponse, error) { return nil, errors.New("unused") },
			verifyEmailFn: func(context.Context, string) error { return errors.New("unused") },
			refreshTokenFn: func(context.Context, string) (*jwt.TokenPair, error) {
				return nil, errors.New("should not be called")
			},
			getUserByIDFn: func(context.Context, uuid.UUID) (*entities.User, error) { return nil, errors.New("unused") },
			getTokenExpFn: func(string) (int64, error) { return 0, errors.New("unused") },
			changePassFn:  func(context.Context, uuid.UUID, *entities.ChangePasswordInput) error { return errors.New("unused") },
		},
		sessionStoreStub{
			createFn: func(context.Context, string, *redis.SessionData, time.Duration) error { return errors.New("unused") },
			getFn:    func(context.Context, string) (*redis.SessionData, error) { return &redis.SessionData{RefreshToken: "x"}, nil },
			deleteFn: func(context.Context, string) error { return nil },
		},
	)

	r := gin.New()
	r.POST("/auth/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.Header.Set("X-Session-Id", "session-from-header")
	// intentionally do not send X-Internal-Proxy-Secret to trigger untrusted-proxy branch
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTokenHandler_ListSupportedTokens_SearchQueryBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTokenHandler(newTokenRepoStub(), newChainRepoStub())
	r := gin.New()
	r.GET("/tokens", h.ListSupportedTokens)

	req := httptest.NewRequest(http.MethodGet, "/tokens?search=idr&page=2&limit=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_RefreshToken_LegacyBodyWithoutTokenAndNoCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_PROXY_SECRET", "")

	h := NewAuthHandler(
		authServiceStub{
			registerFn:    func(context.Context, *entities.CreateUserInput) (*entities.User, string, error) { return nil, "", errors.New("unused") },
			loginFn:       func(context.Context, *entities.LoginInput) (*entities.AuthResponse, error) { return nil, errors.New("unused") },
			verifyEmailFn: func(context.Context, string) error { return errors.New("unused") },
			refreshTokenFn: func(context.Context, string) (*jwt.TokenPair, error) {
				return nil, errors.New("should not be called")
			},
			getUserByIDFn: func(context.Context, uuid.UUID) (*entities.User, error) { return nil, errors.New("unused") },
			getTokenExpFn: func(string) (int64, error) { return 0, errors.New("unused") },
			changePassFn:  func(context.Context, uuid.UUID, *entities.ChangePasswordInput) error { return errors.New("unused") },
		},
		sessionStoreStub{
			createFn: func(context.Context, string, *redis.SessionData, time.Duration) error { return errors.New("unused") },
			getFn:    func(context.Context, string) (*redis.SessionData, error) { return nil, errors.New("unused") },
			deleteFn: func(context.Context, string) error { return nil },
		},
	)

	r := gin.New()
	r.POST("/auth/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(`{"refreshToken":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}
