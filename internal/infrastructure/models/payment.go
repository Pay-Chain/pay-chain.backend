package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	SenderID            uuid.UUID  `gorm:"type:uuid;not null;index"`
	MerchantID          *uuid.UUID `gorm:"type:uuid;index"` // Nullable
	ReceiverWalletID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	SourceChainID       string     `gorm:"type:varchar(50);not null"`
	DestChainID         string     `gorm:"type:varchar(50);not null"`
	SourceTokenID       uuid.UUID  `gorm:"type:uuid;not null"`
	DestTokenID         uuid.UUID  `gorm:"type:uuid;not null"`
	SourceTokenAddress  string     `gorm:"type:varchar(255);not null"`
	DestTokenAddress    string     `gorm:"type:varchar(255);not null"`
	ReceiverAddress     string     `gorm:"type:varchar(255);not null"`
	SourceAmount        string     `gorm:"type:varchar(100);not null"` // BigInt
	DestAmount          *string    `gorm:"type:varchar(100)"`          // BigInt, nullable
	Decimals            int        `gorm:"not null"`
	FeeAmount           string     `gorm:"type:varchar(100);default:'0'"`
	TotalCharged        string     `gorm:"type:varchar(100);default:'0'"`
	BridgeType          string     `gorm:"type:varchar(50)"`
	Status              string     `gorm:"type:varchar(50);not null;index"`
	SourceTxHash        *string    `gorm:"type:varchar(255);index"`
	DestTxHash          *string    `gorm:"type:varchar(255);index"`
	RefundTxHash        *string    `gorm:"type:varchar(255)"`
	CrossChainMessageID *string    `gorm:"type:varchar(255);index"`
	ExpiresAt           *time.Time
	RefundedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}

type PaymentEvent struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	PaymentID   uuid.UUID `gorm:"type:uuid;not null;index"`
	EventType   string    `gorm:"type:varchar(50);not null;index"`
	Chain       string    `gorm:"type:varchar(50)"`
	TxHash      string    `gorm:"type:varchar(255)"`
	BlockNumber int64
	Metadata    string `gorm:"type:jsonb;default:'{}'"`
	CreatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	Payment Payment `gorm:"foreignKey:PaymentID"`
}
