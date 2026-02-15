package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	redisv9 "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	redispkg "pay-chain.backend/pkg/redis"
)

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
