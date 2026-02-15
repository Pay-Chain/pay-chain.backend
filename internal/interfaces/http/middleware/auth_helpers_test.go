package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestContextGetters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	_, ok := GetUserID(c)
	require.False(t, ok)
	_, ok = GetUserEmail(c)
	require.False(t, ok)
	_, ok = GetUserRole(c)
	require.False(t, ok)

	id := uuid.New()
	c.Set(UserIDKey, id)
	c.Set(UserEmailKey, "user@paychain.io")
	c.Set(UserRoleKey, "ADMIN")

	gotID, ok := GetUserID(c)
	require.True(t, ok)
	require.Equal(t, id, gotID)
	gotEmail, ok := GetUserEmail(c)
	require.True(t, ok)
	require.Equal(t, "user@paychain.io", gotEmail)
	gotRole, ok := GetUserRole(c)
	require.True(t, ok)
	require.Equal(t, "ADMIN", gotRole)
}

func TestRequireRolePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("unauthorized_when_no_role", func(t *testing.T) {
		r := gin.New()
		r.Use(RequireRole("ADMIN"))
		r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("forbidden_when_role_not_allowed", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(UserRoleKey, "USER")
			c.Next()
		})
		r.Use(RequireAdmin())
		r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("success_with_allowed_role", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(UserRoleKey, "SUB_ADMIN")
			c.Next()
		})
		r.Use(RequireAdminOrSubAdmin())
		r.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestIsTrustedProxyRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	prev := os.Getenv("INTERNAL_PROXY_SECRET")
	t.Cleanup(func() {
		_ = os.Setenv("INTERNAL_PROXY_SECRET", prev)
	})

	_ = os.Setenv("INTERNAL_PROXY_SECRET", "")
	require.True(t, IsTrustedProxyRequest(c))

	_ = os.Setenv("INTERNAL_PROXY_SECRET", "secret-123")
	c.Request.Header.Set("X-Internal-Proxy-Secret", "wrong")
	require.False(t, IsTrustedProxyRequest(c))
	c.Request.Header.Set("X-Internal-Proxy-Secret", "secret-123")
	require.True(t, IsTrustedProxyRequest(c))
}
