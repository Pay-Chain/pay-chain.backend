package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
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
	var input struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	tokenPair, err := h.authUsecase.RefreshToken(c.Request.Context(), input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or expired refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  tokenPair.AccessToken,
		"refreshToken": tokenPair.RefreshToken,
	})
}
