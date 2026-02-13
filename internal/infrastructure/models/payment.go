package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	SenderID            uuid.UUID  `gorm:"type:uuid;not null;index"` // References Users? Or generic text? Assuming Users for now, but usually Sender is UserID
	MerchantID          *uuid.UUID `gorm:"type:uuid;index"`
	BridgeID            *uuid.UUID `gorm:"type:uuid;index"`
	SourceChainID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	DestChainID         uuid.UUID  `gorm:"type:uuid;not null;index"`
	SourceTokenID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	DestTokenID         uuid.UUID  `gorm:"type:uuid;not null;index"`
	SourceAmount        string     `gorm:"type:decimal(36,18);not null"`
	DestAmount          *string    `gorm:"type:decimal(36,18)"`
	FeeAmount           string     `gorm:"type:decimal(36,18);default:0"`
	TotalCharged        string     `gorm:"type:decimal(36,18);default:0"`
	ReceiverAddress     string     `gorm:"type:varchar(255);not null"`
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

	// Relations
	SourceChain Chain          `gorm:"foreignKey:SourceChainID;references:ID"`
	DestChain   Chain          `gorm:"foreignKey:DestChainID;references:ID"`
	SourceToken Token          `gorm:"foreignKey:SourceTokenID;references:ID"`
	DestToken   Token          `gorm:"foreignKey:DestTokenID;references:ID"`
	Merchant    *Merchant      `gorm:"foreignKey:MerchantID;references:ID"`
	Bridge      *PaymentBridge `gorm:"foreignKey:BridgeID;references:ID"`
}

type PaymentEvent struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	PaymentID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	EventType   string     `gorm:"type:varchar(50);not null;index"`
	ChainID     *uuid.UUID `gorm:"type:uuid;index"`
	TxHash      string     `gorm:"type:varchar(255)"`
	BlockNumber int64
	Metadata    string `gorm:"type:jsonb;default:'{}'"`
	CreatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	Payment Payment `gorm:"foreignKey:PaymentID"`
}

type PaymentBridge struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	Name      string    `gorm:"type:varchar(50);unique;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type BridgeConfig struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	BridgeID      uuid.UUID `gorm:"type:uuid;not null;index"`
	SourceChainID uuid.UUID `gorm:"type:uuid;not null;index"`
	DestChainID   uuid.UUID `gorm:"type:uuid;not null;index"`
	RouterAddress string    `gorm:"type:varchar(255)"`
	FeePercentage string    `gorm:"type:decimal(5,4);default:0"`
	Config        string    `gorm:"type:jsonb"`
	IsActive      bool      `gorm:"default:true"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	Bridge PaymentBridge `gorm:"foreignKey:BridgeID;references:ID"`
}

type FeeConfig struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	ChainID            uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenID            uuid.UUID `gorm:"type:uuid;not null;index"`
	PlatformFeePercent string    `gorm:"type:decimal(5,4);default:0"`
	FixedBaseFee       string    `gorm:"type:decimal(36,18);default:0"`
	MinFee             string    `gorm:"type:decimal(36,18);default:0"`
	MaxFee             *string   `gorm:"type:decimal(36,18)"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}
