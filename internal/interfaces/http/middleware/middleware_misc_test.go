package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	loggerpkg "pay-chain.backend/pkg/logger"
)

func TestResponseWriter_Write(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	w := responseWriter{
		ResponseWriter: c.Writer,
		body:           &bytes.Buffer{},
	}

	n, err := w.Write([]byte("ok"))
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "ok", w.body.String())
	require.Equal(t, "ok", rec.Body.String())
}

func TestRequestIDMiddleware_GeneratesAndUsesHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("generates request id when header missing", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.GET("/x", func(c *gin.Context) {
			id, ok := c.Get(RequestIDKey)
			require.True(t, ok)
			require.NotEmpty(t, id.(string))
			ctxVal := c.Request.Context().Value("request_id")
			require.NotNil(t, ctxVal)
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("uses provided request id header", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.GET("/x", func(c *gin.Context) {
			id, _ := c.Get(RequestIDKey)
			require.Equal(t, "req-123", id.(string))
			ctxVal := c.Request.Context().Value("request_id")
			require.Equal(t, "req-123", ctxVal)
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("X-Request-ID", "req-123")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestLoggerMiddleware_Executes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	loggerpkg.Init("test")
	r := gin.New()
	r.Use(LoggerMiddleware())
	r.GET("/x", func(c *gin.Context) {
		c.String(http.StatusCreated, "created")
	})

	req := httptest.NewRequest(http.MethodGet, "/x?foo=bar", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "created", rec.Body.String())
}
