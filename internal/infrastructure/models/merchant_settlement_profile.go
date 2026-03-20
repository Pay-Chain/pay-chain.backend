package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MerchantSettlementProfile struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	MerchantID        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	InvoiceCurrency   string    `gorm:"type:varchar(32);not null"`
	DestChain         string    `gorm:"type:varchar(64);not null"`
	DestToken         string    `gorm:"type:varchar(128);not null"`
	DestWallet        string    `gorm:"type:varchar(128);not null"`
	BridgeTokenSymbol string    `gorm:"type:varchar(32);not null;default:'USDC'"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

func (MerchantSettlementProfile) TableName() string {
	return "merchant_settlement_profiles"
}
