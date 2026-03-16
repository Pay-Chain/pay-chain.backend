package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"payment-kita.backend/pkg/redis"
)

// RateLimitMiddleware limits the number of requests per period
// identifier: a function that returns the unique identifier for rate limiting (e.g. IP or UserID)
// limit: maximum number of requests
// period: time window for the limit
func RateLimitMiddleware(identifier func(*gin.Context) string, limit int64, period time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := identifier(c)
		if id == "" {
			c.Next()
			return
		}

		key := fmt.Sprintf("rate_limit:%s:%s", c.Request.URL.Path, id)
		ctx := c.Request.Context()

		count, err := redis.Incr(ctx, key)
		if err != nil {
			// Fail open if Redis is down, or fail closed?
			// Usually rate limiting is a safety layer, so we should fail open but log.
			c.Next()
			return
		}

		if count == 1 {
			// First request, set expiration
			_, _ = redis.Expire(ctx, key, period)
		}

		if count > limit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
				"code":  "ERR_RATE_LIMIT_EXCEEDED",
			})
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-count))
		
		c.Next()
	}
}

// UserIDIdentifier identifies the user from the context
func UserIDIdentifier(c *gin.Context) string {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return c.ClientIP()
	}
	return fmt.Sprintf("%v", userID)
}

// IPIdentifier identifies the user by IP address
func IPIdentifier(c *gin.Context) string {
	return c.ClientIP()
}
