package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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

type authServiceStub struct {
	registerFn      func(ctx context.Context, input *entities.CreateUserInput) (*entities.User, string, error)
	loginFn         func(ctx context.Context, input *entities.LoginInput) (*entities.AuthResponse, error)
	verifyEmailFn   func(ctx context.Context, token string) error
	refreshTokenFn  func(ctx context.Context, refreshToken string) (*jwt.TokenPair, error)
	getUserByIDFn   func(ctx context.Context, id uuid.UUID) (*entities.User, error)
	getTokenExpFn   func(token string) (int64, error)
	changePassFn    func(ctx context.Context, userID uuid.UUID, input *entities.ChangePasswordInput) error
}

func (s authServiceStub) Register(ctx context.Context, input *entities.CreateUserInput) (*entities.User, string, error) {
	return s.registerFn(ctx, input)
}
func (s authServiceStub) Login(ctx context.Context, input *entities.LoginInput) (*entities.AuthResponse, error) {
	return s.loginFn(ctx, input)
}
func (s authServiceStub) VerifyEmail(ctx context.Context, token string) error {
	return s.verifyEmailFn(ctx, token)
}
func (s authServiceStub) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	return s.refreshTokenFn(ctx, refreshToken)
}
func (s authServiceStub) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	return s.getUserByIDFn(ctx, id)
}
func (s authServiceStub) GetTokenExpiry(token string) (int64, error) {
	return s.getTokenExpFn(token)
}
func (s authServiceStub) ChangePassword(ctx context.Context, userID uuid.UUID, input *entities.ChangePasswordInput) error {
	return s.changePassFn(ctx, userID, input)
}

type sessionStoreStub struct {
	createFn func(ctx context.Context, sessionID string, data *redis.SessionData, expiration time.Duration) error
	getFn    func(ctx context.Context, sessionID string) (*redis.SessionData, error)
	deleteFn func(ctx context.Context, sessionID string) error
}

func (s sessionStoreStub) CreateSession(ctx context.Context, sessionID string, data *redis.SessionData, expiration time.Duration) error {
	return s.createFn(ctx, sessionID, data, expiration)
}
func (s sessionStoreStub) GetSession(ctx context.Context, sessionID string) (*redis.SessionData, error) {
	return s.getFn(ctx, sessionID)
}
func (s sessionStoreStub) DeleteSession(ctx context.Context, sessionID string) error {
	return s.deleteFn(ctx, sessionID)
}

func TestAuthHandler_RegisterLoginVerifyAndMe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	var createdSessionID string

	h := NewAuthHandler(
		authServiceStub{
			registerFn: func(_ context.Context, input *entities.CreateUserInput) (*entities.User, string, error) {
				if input.Email == "exists@x.com" {
					return nil, "", domainerrors.ErrAlreadyExists
				}
				return &entities.User{ID: userID, Email: input.Email, Name: input.Name, Role: entities.UserRoleUser, KYCStatus: entities.KYCNotStarted}, "verify-token", nil
			},
			loginFn: func(_ context.Context, input *entities.LoginInput) (*entities.AuthResponse, error) {
				if input.Email == "bad@x.com" {
					return nil, domainerrors.ErrInvalidCredentials
				}
				if input.Email == "err@x.com" {
					return nil, errors.New("login boom")
				}
				return &entities.AuthResponse{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					User:         &entities.User{ID: userID, Email: "user@x.com", Name: "User", Role: entities.UserRoleUser, KYCStatus: entities.KYCNotStarted},
				}, nil
			},
			verifyEmailFn: func(_ context.Context, token string) error {
				if token == "missing" {
					return domainerrors.ErrNotFound
				}
				return nil
			},
			getUserByIDFn: func(_ context.Context, id uuid.UUID) (*entities.User, error) {
				if id == userID {
					return &entities.User{ID: userID, Email: "user@x.com", Name: "User", Role: entities.UserRoleUser, KYCStatus: entities.KYCNotStarted}, nil
				}
				return nil, domainerrors.ErrNotFound
			},
			refreshTokenFn: func(_ context.Context, refreshToken string) (*jwt.TokenPair, error) {
				return nil, errors.New("unused")
			},
			getTokenExpFn: func(token string) (int64, error) { return 0, errors.New("unused") },
			changePassFn:  func(_ context.Context, _ uuid.UUID, _ *entities.ChangePasswordInput) error { return nil },
		},
		sessionStoreStub{
			createFn: func(_ context.Context, sessionID string, data *redis.SessionData, expiration time.Duration) error {
				createdSessionID = sessionID
				if data.AccessToken == "" || data.RefreshToken == "" {
					return errors.New("invalid session data")
				}
				return nil
			},
			getFn: func(_ context.Context, _ string) (*redis.SessionData, error) { return nil, errors.New("unused") },
			deleteFn: func(_ context.Context, _ string) error { return nil },
		},
	)

	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/verify-email", h.VerifyEmail)
	r.GET("/auth/me", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		h.GetMe(c)
	})

	// Register success
	regBody := []byte(`{"email":"user@x.com","name":"User","password":"secret123","walletAddress":"0xabc","walletChainId":"eip155:8453","walletSignature":"sig"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	// Register conflict mapping
	regBody = []byte(`{"email":"exists@x.com","name":"User","password":"secret123","walletAddress":"0xabc","walletChainId":"eip155:8453","walletSignature":"sig"}`)
	req = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}

	// Login success
	loginBody := []byte(`{"email":"user@x.com","password":"secret123"}`)
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if createdSessionID == "" {
		t.Fatalf("expected session id generated")
	}

	// Login invalid credentials mapping
	loginBody = []byte(`{"email":"bad@x.com","password":"secret123"}`)
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}

	// Login generic error
	loginBody = []byte(`{"email":"err@x.com","password":"secret123"}`)
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	// Verify email success
	req = httptest.NewRequest(http.MethodPost, "/auth/verify-email", bytes.NewReader([]byte(`{"token":"ok"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Verify email not found mapping
	req = httptest.NewRequest(http.MethodPost, "/auth/verify-email", bytes.NewReader([]byte(`{"token":"missing"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}

	// Get me success
	req = httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_RefreshChangePasswordLogoutAndSessionExpiry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	prevSecret := os.Getenv("INTERNAL_PROXY_SECRET")
	defer func() { _ = os.Setenv("INTERNAL_PROXY_SECRET", prevSecret) }()
	_ = os.Setenv("INTERNAL_PROXY_SECRET", "proxy-secret")

	h := NewAuthHandler(
		authServiceStub{
			registerFn:    func(context.Context, *entities.CreateUserInput) (*entities.User, string, error) { return nil, "", errors.New("unused") },
			loginFn:       func(context.Context, *entities.LoginInput) (*entities.AuthResponse, error) { return nil, errors.New("unused") },
			verifyEmailFn: func(context.Context, string) error { return errors.New("unused") },
			refreshTokenFn: func(_ context.Context, refreshToken string) (*jwt.TokenPair, error) {
				if refreshToken == "bad-refresh" {
					return nil, errors.New("bad refresh")
				}
				return &jwt.TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil
			},
			getUserByIDFn: func(context.Context, uuid.UUID) (*entities.User, error) { return nil, errors.New("unused") },
			getTokenExpFn: func(token string) (int64, error) {
				if token == "bad-token" {
					return 0, errors.New("bad token")
				}
				return 1710000000, nil
			},
			changePassFn: func(_ context.Context, userID uuid.UUID, input *entities.ChangePasswordInput) error {
				if input.CurrentPassword == "wrong" {
					return domainerrors.NewAppError(http.StatusUnauthorized, domainerrors.CodeInvalidCredentials, "Current password is incorrect", domainerrors.ErrInvalidCredentials)
				}
				return nil
			},
		},
		sessionStoreStub{
			createFn: func(_ context.Context, sessionID string, data *redis.SessionData, expiration time.Duration) error {
				if data.RefreshToken == "" || data.AccessToken == "" {
					return errors.New("missing session tokens")
				}
				if sessionID == "force-create-error" {
					return errors.New("session create error")
				}
				return nil
			},
			getFn: func(_ context.Context, sessionID string) (*redis.SessionData, error) {
				switch sessionID {
				case "sid-ok":
					return &redis.SessionData{AccessToken: "good-token", RefreshToken: "good-refresh"}, nil
				case "sid-bad-refresh":
					return &redis.SessionData{AccessToken: "good-token", RefreshToken: "bad-refresh"}, nil
				case "sid-bad-token":
					return &redis.SessionData{AccessToken: "bad-token", RefreshToken: "good-refresh"}, nil
				default:
					return nil, errors.New("session not found")
				}
			},
			deleteFn: func(_ context.Context, _ string) error { return nil },
		},
	)

	r := gin.New()
	r.POST("/auth/refresh", h.RefreshToken)
	r.POST("/auth/change-password", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		h.ChangePassword(c)
	})
	r.POST("/auth/logout", h.Logout)
	r.GET("/auth/session-expiry", h.GetSessionExpiry)

	// Refresh success via trusted proxy + header
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	req.Header.Set("X-Session-Id", "sid-ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Refresh unauthorized when refresh token invalid
	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	req.Header.Set("X-Session-Id", "sid-bad-refresh")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}

	// Change password success
	req = httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader([]byte(`{"currentPassword":"oldpass123","newPassword":"newpass123"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Change password same password validation
	req = httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader([]byte(`{"currentPassword":"samepass1","newPassword":"samepass1"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}

	// Logout always success
	req = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid-ok"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Session expiry success
	req = httptest.NewRequest(http.MethodGet, "/auth/session-expiry", nil)
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	req.Header.Set("X-Session-Id", "sid-ok")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Session expiry invalid session token
	req = httptest.NewRequest(http.MethodGet, "/auth/session-expiry", nil)
	req.Header.Set("X-Internal-Proxy-Secret", "proxy-secret")
	req.Header.Set("X-Session-Id", "sid-bad-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}
