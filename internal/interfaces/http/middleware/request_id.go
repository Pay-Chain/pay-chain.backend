package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

// RequestIDMiddleware generates a unique ID for each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for existing header
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}

		// Set in context
		c.Set(RequestIDKey, id)

		// Also set in Go Context for logger compatibility
		// We use string "request_id" key as defined in pkg/logger
		// This allows logger.WithContext(c.Request.Context()) to find it
		ctx := context.WithValue(c.Request.Context(), "request_id", id)
		c.Request = c.Request.WithContext(ctx)

		// Pass to next handler
		c.Next()
	}
}
