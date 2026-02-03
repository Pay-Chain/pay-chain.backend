package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Merchant struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID             uuid.UUID `gorm:"type:uuid;not null;index"`
	BusinessName       string    `gorm:"type:varchar(255);not null"`
	BusinessEmail      string    `gorm:"type:varchar(255);not null"`
	MerchantType       string    `gorm:"type:varchar(50);not null"`
	Status             string    `gorm:"type:varchar(50);not null;default:'pending'"`
	TaxID              string    `gorm:"type:varchar(50)"`
	BusinessAddress    string    `gorm:"type:text"`
	Documents          string    `gorm:"type:jsonb;default:'{}'"`
	FeeDiscountPercent float64   `gorm:"type:decimal(5,2);default:0"`
	VerifiedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}
