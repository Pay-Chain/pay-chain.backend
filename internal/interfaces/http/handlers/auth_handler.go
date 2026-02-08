package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authUsecase *usecases.AuthUsecase
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authUsecase *usecases.AuthUsecase) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
	}
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var input entities.CreateUserInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, verificationToken, err := h.authUsecase.Register(c.Request.Context(), &input)
	if err != nil {
		if err == domainerrors.ErrAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already registered",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register user",
		})
		return
	}

	// TODO: Send verification email with token
	_ = verificationToken

	c.JSON(http.StatusCreated, gin.H{
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authResponse, err := h.authUsecase.Login(c.Request.Context(), &input)
	if err != nil {
		if err == domainerrors.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to login",
		})
		return
	}

	// Set tokens in cookies
	c.SetCookie("token", authResponse.AccessToken, 3600*24, "/", "", false, true)
	c.SetCookie("refresh_token", authResponse.RefreshToken, 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  authResponse.AccessToken,
		"refreshToken": authResponse.RefreshToken,
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.authUsecase.VerifyEmail(c.Request.Context(), input.Token); err != nil {
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid or expired verification token",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify email",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully",
	})
}

// RefreshToken handles token refresh
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	log.Printf("[AuthHandler] RefreshToken: Request received. Content-Length: %d", c.Request.ContentLength)

	var refreshToken string

	// 1. Try to get from JSON body if present
	if c.Request.ContentLength > 0 {
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

	// 2. Fallback to cookie if not in body
	if refreshToken == "" {
		if cookie, err := c.Cookie("refresh_token"); err == nil {
			refreshToken = cookie
			log.Println("[AuthHandler] RefreshToken: Token found in cookie")
		} else {
			log.Printf("[AuthHandler] RefreshToken: No cookie found: %v", err)
		}
	}

	// 3. Last fallback: Check Authorization header as Bearer token if appropriate
	// (Though refresh tokens usually aren't sent as Bearer, some clients might)

	if refreshToken == "" {
		log.Println("[AuthHandler] RefreshToken: Error - No refresh token provided in body or cookie")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Refresh token is required",
		})
		return
	}

	tokenPair, err := h.authUsecase.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or expired refresh token",
		})
		return
	}

	// Set new tokens in cookies
	c.SetCookie("token", tokenPair.AccessToken, 3600*24, "/", "", false, true)
	c.SetCookie("refresh_token", tokenPair.RefreshToken, 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  tokenPair.AccessToken,
		"refreshToken": tokenPair.RefreshToken,
	})
}

// GetMe returns current authenticated user details
// GET /api/v1/auth/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	log.Printf("[AuthHandler] GetMe called for path: %s", c.Request.URL.Path)

	val, exists := c.Get(middleware.UserIDKey)
	if !exists {
		log.Printf("[AuthHandler] GetMe failed: userId not found in context (Middleware key: %s)", middleware.UserIDKey)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := val.(uuid.UUID)
	if !ok {
		log.Printf("[AuthHandler] GetMe failed: userID in context is not a UUID (got %T)", val)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	log.Printf("[AuthHandler] GetMe fetching user for ID: %s", userID)
	user, err := h.authUsecase.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[AuthHandler] GetMe failed to fetch user: %v", err)
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	log.Printf("[AuthHandler] GetMe success for user: %s (%s)", user.Name, user.Email)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        user.ID,
			"email":     user.Email,
			"name":      user.Name,
			"role":      user.Role,
			"kycStatus": user.KYCStatus,
		},
	})
}
