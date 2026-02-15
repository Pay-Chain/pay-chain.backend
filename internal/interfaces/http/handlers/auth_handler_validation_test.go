package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/interfaces/http/middleware"
)

func TestAuthHandler_RegisterLoginVerify_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &AuthHandler{}
	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/verify-email", h.VerifyEmail)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_RefreshToken_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_PROXY_SECRET", "secret")

	h := &AuthHandler{}
	r := gin.New()
	r.POST("/auth/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Refresh token is required")
}

func TestAuthHandler_GetMe_And_ChangePassword_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &AuthHandler{}
	r := gin.New()

	r.GET("/auth/me", h.GetMe)
	r.POST("/auth/change-password/noctx", h.ChangePassword)
	r.POST("/auth/change-password/badctx", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, "not-uuid")
		h.ChangePassword(c)
	})
	r.POST("/auth/change-password/same", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, uuid.New())
		h.ChangePassword(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/auth/change-password/noctx", strings.NewReader(`{"currentPassword":"a","newPassword":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/auth/change-password/badctx", strings.NewReader(`{"currentPassword":"a","newPassword":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/auth/change-password/same", strings.NewReader(`{"currentPassword":"same-password-123","newPassword":"same-password-123"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "New password must be different")
}
