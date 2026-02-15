package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
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
