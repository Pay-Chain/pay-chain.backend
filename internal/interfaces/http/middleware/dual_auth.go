package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
)

// DualAuthMiddleware handles both JWT and API Key authentication
func DualAuthMiddleware(jwtService *jwt.JWTService, apiKeyUsecase *usecases.ApiKeyUsecase, sessionStore *redis.SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		strictSessionMode := os.Getenv("INTERNAL_PROXY_SECRET") != ""
		apiKey := c.GetHeader("X-Api-Key")
		authHeader := c.GetHeader("Authorization")
		signature := c.GetHeader("X-Signature")
		timestamp := c.GetHeader("X-Timestamp")

		// Read body for signature verification
		var bodyBytes []byte
		var err error
		if c.Request.Body != nil {
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
				return
			}
			// Restore body
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		bodyHash := sha256Hex(bodyBytes)

		// Path A: API Key + Signature
		if apiKey != "" && signature != "" && timestamp != "" {
			user, err := apiKeyUsecase.ValidateApiKey(
				c.Request.Context(),
				apiKey,
				signature,
				timestamp,
				c.Request.Method,
				c.Request.URL.RequestURI(), // Includes query params
				bodyHash,
			)

			if err != nil {
				// Avoid leaking specific error details unless useful
				// u.ValidateApiKey returns domain errors which are safe
				// BUT we should map them to HTTP status.
				// For now, assume Unauthorized unless Internal.
				log.Printf("[DualAuth] API Key validation failed: %v", err)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Key or Signature"})
				return
			}

			// Set context
			c.Set(UserIDKey, user.ID)
			c.Set(UserEmailKey, user.Email)
			c.Set(UserRoleKey, string(user.Role))
			c.Next()
			return
		}

		// Path B: JWT + Signature (required)
		tokenString := ""
		tokenFromTrustedSession := false

		// Check for Session ID first (trusted proxy)
		sessionID := c.GetHeader("x-session-id")
		if sessionID != "" && IsTrustedProxyRequest(c) {
			// Retrieve from Redis
			session, err := sessionStore.GetSession(c.Request.Context(), sessionID)
			if err == nil && session != nil {
				tokenString = session.AccessToken
				tokenFromTrustedSession = true
			}
		}

		// Legacy fallback (non-strict mode only)
		if tokenString == "" && !strictSessionMode && authHeader != "" && strings.HasPrefix(authHeader, BearerPrefix) {
			tokenString = strings.TrimPrefix(authHeader, BearerPrefix)
		}

		if tokenString != "" {
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				log.Printf("[DualAuth] JWT validation failed: %v", err)
				code := http.StatusUnauthorized
				msg := "Invalid token"
				if err == jwt.ErrExpiredToken {
					msg = "Token has expired"
				}
				c.AbortWithStatusJSON(code, gin.H{"error": msg})
				return
			}

			// Signature is required for direct JWT requests, but optional for trusted
			// proxy session flow (session_id -> Redis access token).
			if !tokenFromTrustedSession {
				if signature == "" || timestamp == "" {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Signature and timestamp are required for JWT requests"})
					return
				}

				err = apiKeyUsecase.ValidateSignatureForJWT(
					c.Request.Context(),
					claims.UserID,
					signature,
					timestamp,
					c.Request.Method,
					c.Request.URL.RequestURI(),
					bodyHash,
				)
				if err != nil {
					log.Printf("[DualAuth] JWT Signature validation failed: %v", err)
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Signature for JWT user"})
					return
				}
			} else if signature != "" && timestamp != "" {
				// If provided, still verify even in session flow.
				err = apiKeyUsecase.ValidateSignatureForJWT(
					c.Request.Context(),
					claims.UserID,
					signature,
					timestamp,
					c.Request.Method,
					c.Request.URL.RequestURI(),
					bodyHash,
				)
				if err != nil {
					log.Printf("[DualAuth] JWT Signature validation failed (session flow): %v", err)
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Signature for JWT user"})
					return
				}
			}

			// Set context
			c.Set(UserIDKey, claims.UserID)
			c.Set(UserEmailKey, claims.Email)
			c.Set(UserRoleKey, claims.Role)
			c.Next()
			return
		}

		// Neither auth method present or valid
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required (JWT or API Key)",
		})
	}
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
