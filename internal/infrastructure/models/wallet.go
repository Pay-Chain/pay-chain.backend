package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Wallet struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID     *uuid.UUID `gorm:"type:uuid;index"` // Nullable
	MerchantID *uuid.UUID `gorm:"type:uuid;index"` // Nullable
	ChainID    int        `gorm:"not null;index"`  // integer FK to chains.id
	Address    string     `gorm:"type:varchar(255);not null;index"`
	IsPrimary  bool       `gorm:"default:false"`
	CreatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
