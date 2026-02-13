package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
)

const (
	// AuthorizationHeader is the header key for authorization
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the prefix for bearer tokens
	BearerPrefix = "Bearer "
	// UserIDKey is the context key for user ID
	UserIDKey = "userId"
	// UserEmailKey is the context key for user email
	UserEmailKey = "userEmail"
	// UserRoleKey is the context key for user role
	UserRoleKey = "userRole"
)

// AuthMiddleware creates a new authentication middleware
func AuthMiddleware(jwtService *jwt.JWTService, sessionStore *redis.SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""
		strictSessionMode := os.Getenv("INTERNAL_PROXY_SECRET") != ""

		// 1. Check for X-Session-Id header (from trusted proxy)
		sessionID := c.GetHeader("X-Session-Id")
		if sessionID != "" && IsTrustedProxyRequest(c) {
			session, err := sessionStore.GetSession(c.Request.Context(), sessionID)
			if err == nil && session != nil {
				tokenString = session.AccessToken
			}
		}

		// 2. Legacy fallback to Authorization header when strict mode is disabled.
		if tokenString == "" && !strictSessionMode {
			authHeader := c.GetHeader(AuthorizationHeader)
			if authHeader != "" && strings.HasPrefix(authHeader, BearerPrefix) {
				tokenString = strings.TrimPrefix(authHeader, BearerPrefix)
			}
		}

		if tokenString == "" {
			log.Printf("[AuthMiddleware] Request to %s failed: No valid session or token found", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			log.Printf("[AuthMiddleware] Token validation failed for %s: %v", c.Request.URL.Path, err)
			if err == jwt.ErrExpiredToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Token has expired",
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			return
		}

		log.Printf("[AuthMiddleware] Authenticated user %s with role %s for %s", claims.Email, claims.Role, c.Request.URL.Path)

		// Set user info in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserEmailKey, claims.Email)
		c.Set(UserRoleKey, claims.Role)

		c.Next()
	}
}

func IsTrustedProxyRequest(c *gin.Context) bool {
	secret := os.Getenv("INTERNAL_PROXY_SECRET")
	if secret == "" {
		return true // backward compatible for local/dev without configured secret
	}
	return c.GetHeader("X-Internal-Proxy-Secret") == secret
}

// GetUserID gets the user ID from context
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return uuid.Nil, false
	}
	return userID.(uuid.UUID), true
}

// GetUserEmail gets the user email from context
func GetUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get(UserEmailKey)
	if !exists {
		return "", false
	}
	return email.(string), true
}

// GetUserRole gets the user role from context
func GetUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get(UserRoleKey)
	if !exists {
		return "", false
	}
	return role.(string), true
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := GetUserRole(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "User role not found",
			})
			return
		}

		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions",
		})
	}
}

// RequireAdmin creates a middleware that requires admin role
func RequireAdmin() gin.HandlerFunc {
	return RequireRole("ADMIN")
}

// RequireAdminOrSubAdmin creates a middleware that requires admin or sub_admin role
func RequireAdminOrSubAdmin() gin.HandlerFunc {
	return RequireRole("ADMIN", "SUB_ADMIN")
}
