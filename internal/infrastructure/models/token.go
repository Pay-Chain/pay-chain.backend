package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Token struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ChainID         uuid.UUID `gorm:"-"` // Ignored for tokens table
	Symbol          string    `gorm:"type:varchar(20);not null"`
	Name            string    `gorm:"type:varchar(100);not null"`
	Decimals        int       `gorm:"not null"`
	ContractAddress string    `gorm:"-"` // Ignored for tokens table
	Type            string    `gorm:"type:varchar(20);not null;default:'ERC20'"`
	LogoURL         string    `gorm:"type:text"`
	IsActive        bool      `gorm:"-"` // Ignored for tokens table
	IsStablecoin    bool      `gorm:"default:false"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Associations - ignored for tokens table write
	Chain Chain `gorm:"-"`
}

type SupportedToken struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ChainID         uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenID         uuid.UUID `gorm:"type:uuid;not null;index"`
	ContractAddress string    `gorm:"type:varchar(255);not null"`
	IsActive        bool      `gorm:"default:true"`
	MinAmount       string    `gorm:"type:varchar(100)"` // BigInt as string
	MaxAmount       *string   `gorm:"type:varchar(100)"` // BigInt as string, nullable
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Associations
	Token Token `gorm:"foreignKey:TokenID"`
	Chain Chain `gorm:"foreignKey:ChainID;references:ID"`
}
