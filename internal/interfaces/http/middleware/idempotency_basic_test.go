package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/alicebob/miniredis/v2"
	redisv9 "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	redispkg "pay-chain.backend/pkg/redis"
)

func startMiniRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable in this environment: %v", err)
	}
	return srv
}

func TestIdempotencyMiddleware_NoHeaderPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IdempotencyMiddleware())
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestIdempotencyMiddleware_RedisErrorPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redispkg.SetClient(redisv9.NewClient(&redisv9.Options{Addr: "127.0.0.1:0"}))

	r := gin.New()
	r.Use(IdempotencyMiddleware())
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusAccepted) })

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set(IdempotencyHeader, "idem-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)
}

func TestIdempotencyMiddleware_ProcessingConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := startMiniRedis(t)
	t.Cleanup(srv.Close)

	cli := redisv9.NewClient(&redisv9.Options{Addr: srv.Addr()})
	redispkg.SetClient(cli)
	t.Cleanup(func() { _ = cli.Close() })

	storageKey := "idempotency:user-1:key-1"
	srv.Set(storageKey, "processing")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})
	r.Use(IdempotencyMiddleware())
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusCreated) })

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set(IdempotencyHeader, "key-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	require.Contains(t, w.Body.String(), "ERR_IDEMPOTENCY_CONFLICT")
}

func TestIdempotencyMiddleware_CachedHitReturnsBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := startMiniRedis(t)
	t.Cleanup(srv.Close)

	cli := redisv9.NewClient(&redisv9.Options{Addr: srv.Addr()})
	redispkg.SetClient(cli)
	t.Cleanup(func() { _ = cli.Close() })

	storageKey := "idempotency:user-1:key-2"
	srv.Set(storageKey, `{"ok":true}`)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})
	r.Use(IdempotencyMiddleware())
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusCreated) })

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set(IdempotencyHeader, "key-2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "true", w.Header().Get("X-Idempotency-Hit"))
	require.Equal(t, `{"ok":true}`, w.Body.String())
}

func TestIdempotencyMiddleware_StoresAndReplaysSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := startMiniRedis(t)
	t.Cleanup(srv.Close)

	cli := redisv9.NewClient(&redisv9.Options{Addr: srv.Addr()})
	redispkg.SetClient(cli)
	t.Cleanup(func() { _ = cli.Close() })

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})
	r.Use(IdempotencyMiddleware())
	r.POST("/x", func(c *gin.Context) {
		c.String(http.StatusCreated, `{"id":1}`)
	})

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set(IdempotencyHeader, "key-3")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/x", nil)
	req2.Header.Set(IdempotencyHeader, "key-3")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	require.Equal(t, "true", w2.Header().Get("X-Idempotency-Hit"))
	require.Equal(t, `{"id":1}`, w2.Body.String())
}

func TestIdempotencyMiddleware_DeletesKeyOnFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := startMiniRedis(t)
	t.Cleanup(srv.Close)

	cli := redisv9.NewClient(&redisv9.Options{Addr: srv.Addr()})
	redispkg.SetClient(cli)
	t.Cleanup(func() { _ = cli.Close() })

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})
	r.Use(IdempotencyMiddleware())
	r.POST("/x", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "boom")
	})

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set(IdempotencyHeader, "key-4")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	_, err := redispkg.Get(context.Background(), "idempotency:user-1:key-4")
	require.Error(t, err)
	require.Equal(t, redisv9.Nil, err)
}
