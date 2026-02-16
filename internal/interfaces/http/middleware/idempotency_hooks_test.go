package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyMiddleware_WithHookedRedis(t *testing.T) {
	origGet := redisGet
	origSet := redisSet
	origSetNX := redisSetNX
	origDel := redisDel
	t.Cleanup(func() {
		redisGet = origGet
		redisSet = origSet
		redisSetNX = origSetNX
		redisDel = origDel
	})

	t.Run("processing conflict", func(t *testing.T) {
		redisGet = func(context.Context, string) (string, error) { return "processing", nil }
		redisSetNX = func(context.Context, string, interface{}, time.Duration) (bool, error) { return false, nil }
		redisSet = func(context.Context, string, interface{}, time.Duration) error { return nil }
		redisDel = func(context.Context, string) error { return nil }

		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
		r.Use(IdempotencyMiddleware())
		r.POST("/x", func(c *gin.Context) { c.String(http.StatusCreated, `{"id":1}`) })

		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.Header.Set(IdempotencyHeader, "key-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("cached response", func(t *testing.T) {
		redisGet = func(context.Context, string) (string, error) { return `{"ok":true}`, nil }
		redisSetNX = func(context.Context, string, interface{}, time.Duration) (bool, error) { return true, nil }
		redisSet = func(context.Context, string, interface{}, time.Duration) error { return nil }
		redisDel = func(context.Context, string) error { return nil }

		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
		r.Use(IdempotencyMiddleware())
		r.POST("/x", func(c *gin.Context) { c.Status(http.StatusCreated) })

		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.Header.Set(IdempotencyHeader, "key-2")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "true", w.Header().Get("X-Idempotency-Hit"))
	})

	t.Run("setnx conflict and fail path cleanup", func(t *testing.T) {
		setCalled := false
		delCalled := false
		redisGet = func(context.Context, string) (string, error) { return "", errors.New("redis: nil") }
		redisSetNX = func(context.Context, string, interface{}, time.Duration) (bool, error) { return true, nil }
		redisSet = func(context.Context, string, interface{}, time.Duration) error { setCalled = true; return nil }
		redisDel = func(context.Context, string) error { delCalled = true; return nil }

		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
		r.Use(IdempotencyMiddleware())
		r.POST("/ok", func(c *gin.Context) { c.String(http.StatusCreated, `{"id":9}`) })
		r.POST("/fail", func(c *gin.Context) { c.String(http.StatusBadRequest, "bad") })

		reqOK := httptest.NewRequest(http.MethodPost, "/ok", nil)
		reqOK.Header.Set(IdempotencyHeader, "key-3")
		wOK := httptest.NewRecorder()
		r.ServeHTTP(wOK, reqOK)
		require.Equal(t, http.StatusCreated, wOK.Code)
		require.True(t, setCalled)

		reqFail := httptest.NewRequest(http.MethodPost, "/fail", nil)
		reqFail.Header.Set(IdempotencyHeader, "key-4")
		wFail := httptest.NewRecorder()
		r.ServeHTTP(wFail, reqFail)
		require.Equal(t, http.StatusBadRequest, wFail.Code)
		require.True(t, delCalled)
	})

	t.Run("redis read error passthrough", func(t *testing.T) {
		redisGet = func(context.Context, string) (string, error) { return "", errors.New("redis down") }
		redisSetNX = func(context.Context, string, interface{}, time.Duration) (bool, error) { return false, errors.New("boom") }
		redisSet = func(context.Context, string, interface{}, time.Duration) error { return nil }
		redisDel = func(context.Context, string) error { return nil }

		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
		r.Use(IdempotencyMiddleware())
		r.POST("/x", func(c *gin.Context) { c.Status(http.StatusAccepted) })

		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.Header.Set(IdempotencyHeader, "key-5")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("setnx error returns conflict", func(t *testing.T) {
		redisGet = func(context.Context, string) (string, error) { return "", errors.New("redis: nil") }
		redisSetNX = func(context.Context, string, interface{}, time.Duration) (bool, error) { return false, errors.New("boom") }
		redisSet = func(context.Context, string, interface{}, time.Duration) error { return nil }
		redisDel = func(context.Context, string) error { return nil }

		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
		r.Use(IdempotencyMiddleware())
		r.POST("/x", func(c *gin.Context) { c.Status(http.StatusAccepted) })

		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.Header.Set(IdempotencyHeader, "key-6")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusConflict, w.Code)
	})
}
