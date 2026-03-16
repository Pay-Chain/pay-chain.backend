package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	go_redis "github.com/redis/go-redis/v9"

	"github.com/gin-gonic/gin"
	"payment-kita.backend/pkg/redis"
)

const (
	IdempotencyHeader = "Idempotency-Key"
	// LockDuration is the time we hold the lock while processing
	LockDuration = 30 * time.Second
	// RetentionDuration is how long we keep the response
	RetentionDuration = 24 * time.Hour
)

var (
	redisGet   = redis.Get
	redisSet   = redis.Set
	redisSetNX = redis.SetNX
	redisDel   = redis.Del
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// IdempotencyMiddleware ensures that the same request is not processed twice
func IdempotencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Check if Idempotency-Key header is present
		key := c.GetHeader(IdempotencyHeader)
		if key == "" {
			c.Next()
			return
		}

		// 2. Generate a unique key for storage (Prefix + UserID + Key)
		userID, exists := c.Get(UserIDKey)
		if !exists {
			// Fallback to user_id for tests or legacy
			userID, exists = c.Get("user_id")
		}

		if !exists {
			c.Next()
			return
		}
		storageKey := fmt.Sprintf("idempotency:%v:%s", userID, key)

		// 3. Check Redis for existing key
		ctx := c.Request.Context()

		// Simple approach:
		val, err := redisGet(ctx, storageKey)
		if err == nil {
			// Key exists
			if val == "processing" {
				// 409 Conflict - Request in progress
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{
					"error": "Request already in progress",
					"code":  "ERR_IDEMPOTENCY_CONFLICT",
				})
				return
			}

			// Return cached response
			var cached struct {
				Status int    `json:"status"`
				Body   string `json:"body"`
			}
			if err := json.Unmarshal([]byte(val), &cached); err == nil {
				c.Header("Content-Type", "application/json")
				c.Header("X-Idempotency-Hit", "true")
				c.String(cached.Status, cached.Body)
				c.Abort()
				return
			}

			// Fallback if unmarshal fails (e.g. legacy data)
			c.Header("Content-Type", "application/json")
			c.Header("X-Idempotency-Hit", "true")
			c.String(http.StatusOK, val)
			c.Abort()
			return
		} else if !errors.Is(err, go_redis.Nil) {
			// Redis error (non-nill)
			c.Next()
			return
		}

		// 4. Set "processing" state
		success, err := redisSetNX(ctx, storageKey, "processing", LockDuration)
		if err != nil || !success {
			// Race condition or locked
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": "Request in progress",
			})
			return
		}

		// 5. Wrap ResponseWriter to capture body
		w := &responseWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w

		// 6. Process Request
		c.Next()

		// 7. Store Result if Successful (2xx)
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			// Store the status and body
			cachedResponse := struct {
				Status int    `json:"status"`
				Body   string `json:"body"`
			}{
				Status: c.Writer.Status(),
				Body:   w.body.String(),
			}
			data, _ := json.Marshal(cachedResponse)
			_ = redisSet(ctx, storageKey, string(data), RetentionDuration)
		} else {
			// Remove key so retry is possible
			_ = redisDel(ctx, storageKey)
		}
	}
}
