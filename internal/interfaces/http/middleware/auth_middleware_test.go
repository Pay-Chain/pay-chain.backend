package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/pkg/jwt"
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
