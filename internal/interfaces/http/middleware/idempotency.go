package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"payment-kita.backend/pkg/redis"
)

const (
	// IdempotencyKeyHeader is the header name for idempotency key
	IdempotencyKeyHeader = "X-PK-Idempotency-Key"
	// IdempotencyTTL is the time-to-live for idempotency keys (24 hours)
	IdempotencyTTL = 24 * time.Hour
	// MaxIdempotencyKeyLength is the maximum allowed length for idempotency key
	MaxIdempotencyKeyLength = 256
)

// IdempotencyMiddleware creates a middleware that prevents duplicate requests
// by tracking request fingerprints in Redis
func IdempotencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get idempotency key from header
		idempotencyKey := c.GetHeader(IdempotencyKeyHeader)

		// If no idempotency key, continue without idempotency check
		if idempotencyKey == "" {
			c.Next()
			return
		}

		// Validate key length
		if len(idempotencyKey) > MaxIdempotencyKeyLength {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Idempotency key too long",
				"max_length": MaxIdempotencyKeyLength,
			})
			c.Abort()
			return
		}

		// Generate fingerprint from request
		fingerprint := generateRequestFingerprint(c, idempotencyKey)

		// Try to acquire lock in Redis
		lockKey := "idem:lock:" + fingerprint
		acquired, err := redis.SetNX(c.Request.Context(), lockKey, "1", IdempotencyTTL)
		if err != nil {
			// Redis error, fail open to avoid blocking legitimate requests
			c.Next()
			return
		}

		if !acquired {
			// Request is being processed or was recently processed
			// Try to get cached response
			cacheKey := "idem:cache:" + fingerprint
			cachedResponse, err := redis.Get(c.Request.Context(), cacheKey)
			if err == nil && cachedResponse != "" {
				// Return cached response
				c.Data(http.StatusOK, "application/json", []byte(cachedResponse))
				c.Abort()
				return
			}

			// Return conflict error
			c.JSON(http.StatusConflict, gin.H{
				"error": "Duplicate request detected",
				"message": "A request with this idempotency key is already being processed",
				"retry_after": 5, // seconds
			})
			c.Abort()
			return
		}

		// Store original response writer
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Cache the response for future duplicate requests
		if writer.status >= 200 && writer.status < 300 {
			// Only cache successful responses
			cacheKey := "idem:cache:" + fingerprint
			if len(writer.body) > 0 {
				// Cache response for TTL duration
				redis.SetEX(c.Request.Context(), cacheKey, string(writer.body), IdempotencyTTL)
			}
		}

		// Release lock
		redis.Del(c.Request.Context(), lockKey)
	}
}

// generateRequestFingerprint creates a unique fingerprint for the request
func generateRequestFingerprint(c *gin.Context, idempotencyKey string) string {
	// Combine idempotency key with request path and merchant ID (if available)
	merchantID := ""
	if val, exists := c.Get("MerchantID"); exists {
		merchantID = val.(string)
	}

	userID := ""
	if val, exists := c.Get("UserID"); exists {
		userID = val.(string)
	}

	// Create fingerprint from: idempotency_key + path + merchant_id + user_id
	fingerprintData := idempotencyKey + "|" + c.Request.URL.Path + "|" + merchantID + "|" + userID

	// Hash the fingerprint
	hash := sha256.Sum256([]byte(fingerprintData))
	return hex.EncodeToString(hash[:])
}

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body   []byte
	status int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body = append(w.body, []byte(s)...)
	return w.ResponseWriter.WriteString(s)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// ValidateIdempotencyKey validates the format of an idempotency key
func ValidateIdempotencyKey(key string) error {
	if key == "" {
		return nil // Empty key is valid (no idempotency)
	}

	if len(key) > MaxIdempotencyKeyLength {
		return ErrIdempotencyKeyTooLong
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	for _, c := range key {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_') {
			return ErrIdempotencyKeyInvalidChar
		}
	}

	return nil
}

// GenerateIdempotencyKey generates a new idempotency key
func GenerateIdempotencyKey() string {
	hash := sha256.Sum256([]byte(time.Now().String() + strconv.Itoa(time.Now().Nanosecond())))
	return "idem_" + hex.EncodeToString(hash[:16]) // 32 char key
}
