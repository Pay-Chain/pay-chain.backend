package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
)

// UserRole represents user roles
type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleSubAdmin UserRole = "sub_admin"
	UserRolePartner  UserRole = "partner"
	UserRoleUser     UserRole = "user"
)

// KYCStatus represents KYC verification status
type KYCStatus string

const (
	KYCNotStarted       KYCStatus = "not_started"
	KYCIDCardVerified   KYCStatus = "id_card_verified"
	KYCFaceVerified     KYCStatus = "face_verified"
	KYCLivenessVerified KYCStatus = "liveness_verified"
	KYCFullyVerified    KYCStatus = "fully_verified"
)

// User represents a user entity
type User struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	PasswordHash  string    `json:"-"`
	Role          UserRole  `json:"role"`
	KYCStatus     KYCStatus `json:"kycStatus"`
	KYCVerifiedAt null.Time `json:"kycVerifiedAt,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	DeletedAt     null.Time `json:"-"`
}

// CreateUserInput represents input for creating a user
type CreateUserInput struct {
	Email           string `json:"email" binding:"required,email"`
	Name            string `json:"name" binding:"required,min=2,max=100"`
	Password        string `json:"password" binding:"required,min=8"`
	WalletAddress   string `json:"walletAddress" binding:"required"`
	WalletChainID   string `json:"walletChainId" binding:"required"`
	WalletSignature string `json:"walletSignature" binding:"required"`
}

// LoginInput represents input for user login
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         *User  `json:"user"`
}
