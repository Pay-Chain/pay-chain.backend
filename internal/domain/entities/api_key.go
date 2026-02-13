package entities

import (
	"time"

	"github.com/google/uuid"
)

// ApiKey represents an API key for a user
type ApiKey struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	UserID          uuid.UUID  `json:"userId" gorm:"type:uuid;not null"`
	Name            string     `json:"name" gorm:"type:varchar(100);not null"`
	KeyPrefix       string     `json:"keyPrefix" gorm:"type:varchar(20);not null"`
	KeyHash         string     `json:"keyHash" gorm:"type:varchar(64);uniqueIndex;not null"`
	SecretEncrypted string     `json:"secretEncrypted" gorm:"type:text;not null"`
	SecretMasked    string     `json:"secretMasked" gorm:"type:varchar(20);not null"`
	Permissions     []string   `json:"permissions" gorm:"type:jsonb;default:'[]'"`
	IsActive        bool       `json:"isActive" gorm:"default:true"`
	LastUsedAt      *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	DeletedAt       *time.Time `json:"-" gorm:"index"`

	// Relationships
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

type CreateApiKeyInput struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
}

type CreateApiKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	ApiKey    string    `json:"apiKey"`
	SecretKey string    `json:"secretKey"`
	CreatedAt time.Time `json:"createdAt"`
}
