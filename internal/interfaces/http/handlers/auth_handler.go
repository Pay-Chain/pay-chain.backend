package handlers

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
	"pay-chain.backend/pkg/utils"
)

type AuthService interface {
	Register(ctx context.Context, input *entities.CreateUserInput) (*entities.User, string, error)
	Login(ctx context.Context, input *entities.LoginInput) (*entities.AuthResponse, error)
	VerifyEmail(ctx context.Context, token string) error
	RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetTokenExpiry(token string) (int64, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, input *entities.ChangePasswordInput) error
}

type SessionStore interface {
	CreateSession(ctx context.Context, sessionID string, data *redis.SessionData, expiration time.Duration) error
	GetSession(ctx context.Context, sessionID string) (*redis.SessionData, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authUsecase  AuthService
	sessionStore SessionStore
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authUsecase AuthService, sessionStore SessionStore) *AuthHandler {
	return &AuthHandler{
		authUsecase:  authUsecase,
		sessionStore: sessionStore,
	}
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var input entities.CreateUserInput

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	user, verificationToken, err := h.authUsecase.Register(c.Request.Context(), &input)
	if err != nil {
		if err == domainerrors.ErrAlreadyExists {
			response.Error(c, domainerrors.Conflict("Email already registered"))
			return
		}
		response.Error(c, err)
		return
	}

	// TODO: Send verification email with token
	_ = verificationToken

	response.Success(c, http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email for verification.",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var input entities.LoginInput

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	authResponse, err := h.authUsecase.Login(c.Request.Context(), &input)
	if err != nil {
		if err == domainerrors.ErrInvalidCredentials {
			response.Error(c, domainerrors.NewAppError(http.StatusUnauthorized, domainerrors.CodeInvalidCredentials, "Invalid email or password", domainerrors.ErrInvalidCredentials))
			return
		}
		response.Error(c, err)
		return
	}

	// Generate Session ID
	sessionID := utils.GenerateUUIDv7().String()

	// Store in Redis (encrypted)
	// Refresh expiry is longer, so use that for session TTL
	sessionData := &redis.SessionData{
		AccessToken:  authResponse.AccessToken,
		RefreshToken: authResponse.RefreshToken,
	}
	// We need config for expiry? Or use hardcoded defaults matching JWT?
	// The implementation plan says "Use RefreshToken expiry".
	// We don't have config here. But authResponse doesn't have expiry.
	// Let's assume 7 days (standard) or we should have passed config to Handler?
	// Better: pass config or just use a safe default 7 days.
	// Or even better: `authUsecase` should return expiry?
	// For now, let's use 7 days (604800 seconds).
	err = h.sessionStore.CreateSession(c.Request.Context(), sessionID, sessionData, 7*24*time.Hour)
	if err != nil {
		response.Error(c, domainerrors.InternalError(err))
		return
	}

	// Set session_id cookie (HttpOnly)
	c.SetCookie("session_id", sessionID, 3600*24*7, "/", "", false, true)

	// Remove old token cookies if they exist (cleanup)
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.Success(c, http.StatusOK, gin.H{
		"sessionId": sessionID, // Proxy needs this? If proxy reads cookie, maybe not needed in body. But plan says "Return sessionID to the frontend proxy".
		"user": gin.H{
			"id":        authResponse.User.ID,
			"email":     authResponse.User.Email,
			"name":      authResponse.User.Name,
			"role":      authResponse.User.Role,
			"kycStatus": authResponse.User.KYCStatus,
		},
	})
}

// VerifyEmail handles email verification
// POST /api/v1/auth/verify-email
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var input struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	if err := h.authUsecase.VerifyEmail(c.Request.Context(), input.Token); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.BadRequest("Invalid or expired verification token"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Email verified successfully",
	})
}

// RefreshToken handles token refresh
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	log.Printf("[AuthHandler] RefreshToken: Request received. Content-Length: %d", c.Request.ContentLength)

	var refreshToken string
	strictSessionMode := os.Getenv("INTERNAL_PROXY_SECRET") != ""

	// 1. Try to get from Redis session (session_id header/cookie)
	sessionID := c.GetHeader("X-Session-Id")
	if sessionID == "" && !strictSessionMode {
		sessionID, _ = c.Cookie("session_id")
	}
	if sessionID != "" && middleware.IsTrustedProxyRequest(c) {
		if session, sessErr := h.sessionStore.GetSession(c.Request.Context(), sessionID); sessErr == nil && session != nil {
			refreshToken = session.RefreshToken
			log.Println("[AuthHandler] RefreshToken: Token loaded from Redis session")
		}
	}

	// 2. Fallback to JSON body if present (legacy mode).
	if refreshToken == "" && !strictSessionMode && c.Request.ContentLength > 0 {
		var input struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := c.ShouldBindJSON(&input); err == nil {
			refreshToken = input.RefreshToken
			if refreshToken != "" {
				log.Println("[AuthHandler] RefreshToken: Token found in JSON body")
			}
		} else {
			log.Printf("[AuthHandler] RefreshToken: Failed to bind JSON: %v", err)
		}
	}

	// 3. Fallback to cookie if not in body/session (legacy mode).
	if refreshToken == "" && !strictSessionMode {
		if cookie, err := c.Cookie("refresh_token"); err == nil {
			refreshToken = cookie
			log.Println("[AuthHandler] RefreshToken: Token found in cookie")
		} else {
			log.Printf("[AuthHandler] RefreshToken: No cookie found: %v", err)
		}
	}

	if refreshToken == "" {
		log.Println("[AuthHandler] RefreshToken: Error - No refresh token provided in body/cookie/session")
		response.Error(c, domainerrors.BadRequest("Refresh token is required"))
		return
	}

	tokenPair, err := h.authUsecase.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		response.Error(c, domainerrors.NewAppError(http.StatusUnauthorized, domainerrors.CodeUnauthorized, "Invalid or expired refresh token", err))
		return
	}

	// Update session in Redis
	// Reuse sessionID from header/cookie if available.
	if sessionID == "" && !strictSessionMode {
		if cookieSessionID, cookieErr := c.Cookie("session_id"); cookieErr == nil && cookieSessionID != "" {
			sessionID = cookieSessionID
		}
	}
	if sessionID == "" {
		sessionID = utils.GenerateUUIDv7().String()
	}

	newData := &redis.SessionData{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}
	err = h.sessionStore.CreateSession(c.Request.Context(), sessionID, newData, 7*24*time.Hour)
	if err != nil {
		response.Error(c, domainerrors.InternalError(err))
		return
	}

	// Set new session cookie
	c.SetCookie("session_id", sessionID, 3600*24*7, "/", "", false, true)

	// Clear old cookies
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.Success(c, http.StatusOK, gin.H{
		"sessionId": sessionID,
	})
}

// GetMe returns current authenticated user details
// GET /api/v1/auth/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	log.Printf("[AuthHandler] GetMe called for path: %s", c.Request.URL.Path)

	val, exists := c.Get(middleware.UserIDKey)
	if !exists {
		log.Printf("[AuthHandler] GetMe failed: userId not found in context (Middleware key: %s)", middleware.UserIDKey)
		response.Error(c, domainerrors.Unauthorized("Unauthorized"))
		return
	}

	userID, ok := val.(uuid.UUID)
	if !ok {
		log.Printf("[AuthHandler] GetMe failed: userID in context is not a UUID (got %T)", val)
		response.Error(c, domainerrors.InternalError(nil))
		return
	}

	log.Printf("[AuthHandler] GetMe fetching user for ID: %s", userID)
	user, err := h.authUsecase.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[AuthHandler] GetMe failed to fetch user: %v", err)
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("User not found"))
			return
		}
		response.Error(c, err)
		return
	}

	log.Printf("[AuthHandler] GetMe success for user: %s (%s)", user.Name, user.Email)
	response.Success(c, http.StatusOK, gin.H{
		"user": gin.H{
			"id":        user.ID,
			"email":     user.Email,
			"name":      user.Name,
			"role":      user.Role,
			"kycStatus": user.KYCStatus,
		},
	})
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, err := c.Cookie("session_id")
	if err == nil && sessionID != "" {
		_ = h.sessionStore.DeleteSession(c.Request.Context(), sessionID)
	}

	// Clear cookies
	c.SetCookie("session_id", "", -1, "/", "", false, true)
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.Success(c, http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// ChangePassword handles changing password for authenticated user.
// POST /api/v1/auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	val, exists := c.Get(middleware.UserIDKey)
	if !exists {
		response.Error(c, domainerrors.Unauthorized("Unauthorized"))
		return
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		response.Error(c, domainerrors.InternalError(nil))
		return
	}

	var input entities.ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}
	if input.CurrentPassword == input.NewPassword {
		response.Error(c, domainerrors.BadRequest("New password must be different from current password"))
		return
	}

	if err := h.authUsecase.ChangePassword(c.Request.Context(), userID, &input); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// GetSessionExpiry returns current access token expiry from Redis session.
// GET /api/v1/auth/session-expiry
func (h *AuthHandler) GetSessionExpiry(c *gin.Context) {
	sessionID := c.GetHeader("X-Session-Id")
	strictSessionMode := os.Getenv("INTERNAL_PROXY_SECRET") != ""
	if sessionID == "" && !strictSessionMode {
		sessionID, _ = c.Cookie("session_id")
	}
	if sessionID == "" {
		response.Error(c, domainerrors.Unauthorized("No session"))
		return
	}
	if !middleware.IsTrustedProxyRequest(c) {
		response.Error(c, domainerrors.Forbidden("Invalid proxy request"))
		return
	}

	session, err := h.sessionStore.GetSession(c.Request.Context(), sessionID)
	if err != nil || session == nil || session.AccessToken == "" {
		response.Error(c, domainerrors.Unauthorized("Invalid session"))
		return
	}

	exp, err := h.authUsecase.GetTokenExpiry(session.AccessToken)
	if err != nil {
		response.Error(c, domainerrors.Unauthorized("Invalid session token"))
		return
	}

	response.Success(c, http.StatusOK, gin.H{"exp": exp})
}
