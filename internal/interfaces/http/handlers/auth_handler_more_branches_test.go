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
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
)

func TestAuthHandler_GetMe_ErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	h := NewAuthHandler(
		authServiceStub{
			registerFn:    func(context.Context, *entities.CreateUserInput) (*entities.User, string, error) { return nil, "", errors.New("unused") },
			loginFn:       func(context.Context, *entities.LoginInput) (*entities.AuthResponse, error) { return nil, errors.New("unused") },
			verifyEmailFn: func(context.Context, string) error { return errors.New("unused") },
			refreshTokenFn: func(context.Context, string) (*jwt.TokenPair, error) {
				return nil, errors.New("unused")
			},
			getUserByIDFn: func(_ context.Context, id uuid.UUID) (*entities.User, error) {
				if id == userID {
					return nil, domainerrors.ErrNotFound
				}
				return nil, errors.New("db down")
			},
			getTokenExpFn: func(string) (int64, error) { return 0, errors.New("unused") },
			changePassFn:  func(context.Context, uuid.UUID, *entities.ChangePasswordInput) error { return errors.New("unused") },
		},
		sessionStoreStub{
			createFn: func(context.Context, string, *redis.SessionData, time.Duration) error { return nil },
			getFn:    func(context.Context, string) (*redis.SessionData, error) { return nil, nil },
			deleteFn: func(context.Context, string) error { return nil },
		},
	)

	r := gin.New()
	r.GET("/auth/me/type", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, "not-uuid")
		h.GetMe(c)
	})
	r.GET("/auth/me/notfound", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		h.GetMe(c)
	})
	r.GET("/auth/me/internal", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, uuid.New())
		h.GetMe(c)
	})

	for path, code := range map[string]int{
		"/auth/me/type":     http.StatusInternalServerError,
		"/auth/me/notfound": http.StatusNotFound,
		"/auth/me/internal": http.StatusInternalServerError,
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != code {
			t.Fatalf("path %s expected %d got %d body=%s", path, code, w.Code, w.Body.String())
		}
	}
}

func TestAuthHandler_RefreshToken_LegacyFallbackBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_PROXY_SECRET", "")
	h := NewAuthHandler(
		authServiceStub{
			registerFn:    func(context.Context, *entities.CreateUserInput) (*entities.User, string, error) { return nil, "", errors.New("unused") },
			loginFn:       func(context.Context, *entities.LoginInput) (*entities.AuthResponse, error) { return nil, errors.New("unused") },
			verifyEmailFn: func(context.Context, string) error { return errors.New("unused") },
			refreshTokenFn: func(_ context.Context, refreshToken string) (*jwt.TokenPair, error) {
				if refreshToken == "bad" {
					return nil, errors.New("bad refresh")
				}
				return &jwt.TokenPair{AccessToken: "access", RefreshToken: "refresh"}, nil
			},
			getUserByIDFn: func(context.Context, uuid.UUID) (*entities.User, error) { return nil, errors.New("unused") },
			getTokenExpFn: func(string) (int64, error) { return 0, errors.New("unused") },
			changePassFn:  func(context.Context, uuid.UUID, *entities.ChangePasswordInput) error { return errors.New("unused") },
		},
		sessionStoreStub{
			createFn: func(_ context.Context, sessionID string, data *redis.SessionData, _ time.Duration) error {
				if sessionID == "force-error" {
					return errors.New("redis down")
				}
				if data.RefreshToken == "" {
					return errors.New("missing refresh token")
				}
				return nil
			},
			getFn:    func(context.Context, string) (*redis.SessionData, error) { return nil, errors.New("unused") },
			deleteFn: func(context.Context, string) error { return nil },
		},
	)

	r := gin.New()
	r.POST("/auth/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader([]byte(`{"refreshToken":"ok"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "ok"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader([]byte(`{"refreshToken":"ok"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "force-error"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader([]byte(`{"refreshToken":"bad"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

