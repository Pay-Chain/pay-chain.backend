package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"payment-kita.backend/pkg/logger"
)

// LoggerMiddleware logs HTTP requests using the structured logger
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// CORS preflight can be noisy and does not represent business traffic.
		if method == "OPTIONS" {
			return
		}

		// Calculate latency
		end := time.Now()
		latency := end.Sub(start)

		if raw != "" {
			path = path + "?" + raw
		}

		// Log using our structured logger
		// The RequestID is expected to be in c.Request.Context() by RequestIDMiddleware
		logger.LogRequest(c.Request.Context(), method, path, c.Writer.Status(), latency, c.ClientIP())
	}
}
