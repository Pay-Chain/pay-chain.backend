package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Token struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	ChainID         uuid.UUID `gorm:"type:uuid;not null;index"`
	Symbol          string    `gorm:"type:varchar(20);not null"`
	Name            string    `gorm:"type:varchar(100);not null"`
	Decimals        int       `gorm:"not null"`
	ContractAddress string    `gorm:"type:varchar(255);index"` // Nullable for native
	Type            string    `gorm:"type:varchar(20);not null;default:'ERC20'"`
	LogoURL         string    `gorm:"type:text"`
	IsActive        bool      `gorm:"default:true"`
	IsNative        bool      `gorm:"default:false"`
	IsStablecoin    bool      `gorm:"default:false"`
	MinAmount       string    `gorm:"type:decimal(36,18);default:0"`
	MaxAmount       *string   `gorm:"type:decimal(36,18)"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Associations
	Chain Chain `gorm:"foreignKey:ChainID;references:ID"`
}
