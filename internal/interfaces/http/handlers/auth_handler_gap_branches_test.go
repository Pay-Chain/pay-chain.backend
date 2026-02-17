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
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
)

func TestAuthHandler_Login_Refresh_ChangePassword_GapBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_PROXY_SECRET", "")

	userID := uuid.New()
	h := NewAuthHandler(
		authServiceStub{
			registerFn:    func(context.Context, *entities.CreateUserInput) (*entities.User, string, error) { return nil, "", errors.New("unused") },
			loginFn:       func(context.Context, *entities.LoginInput) (*entities.AuthResponse, error) { return &entities.AuthResponse{AccessToken: "a", RefreshToken: "r", User: &entities.User{ID: userID}}, nil },
			verifyEmailFn: func(context.Context, string) error { return errors.New("unused") },
			refreshTokenFn: func(_ context.Context, token string) (*jwt.TokenPair, error) {
				return &jwt.TokenPair{AccessToken: "a2", RefreshToken: "r2"}, nil
			},
			getUserByIDFn: func(context.Context, uuid.UUID) (*entities.User, error) { return nil, errors.New("unused") },
			getTokenExpFn: func(string) (int64, error) { return 0, errors.New("unused") },
			changePassFn:  func(context.Context, uuid.UUID, *entities.ChangePasswordInput) error { return nil },
		},
		sessionStoreStub{
			createFn: func(_ context.Context, sessionID string, _ *redis.SessionData, _ time.Duration) error {
				if sessionID != "" {
					return errors.New("create session failed")
				}
				return nil
			},
			getFn: func(context.Context, string) (*redis.SessionData, error) { return nil, errors.New("unused") },
			deleteFn: func(context.Context, string) error { return nil },
		},
	)

	r := gin.New()
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.RefreshToken)
	r.POST("/auth/change-password", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		h.ChangePassword(c)
	})

	// Cover login session store CreateSession error branch.
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"u@x.com","password":"p"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// Cover refresh JSON bind error branch (invalid JSON with positive content-length).
	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(`{"refreshToken":`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Cover change-password bind error branch.
	req = httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewBufferString(`{"currentPassword":`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

