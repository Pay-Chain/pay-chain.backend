package models

import (
	"time"

	"github.com/google/uuid"
)

type PaymentQuote struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	MerchantID            uuid.UUID `gorm:"type:uuid;not null;index"`
	InvoiceCurrency       string    `gorm:"type:varchar(32);not null"`
	InvoiceAmount         string    `gorm:"type:decimal(78,0);not null"`
	SelectedChainID       string    `gorm:"type:varchar(64);not null"`
	SelectedTokenAddress  string    `gorm:"type:varchar(128);not null"`
	SelectedTokenSymbol   string    `gorm:"type:varchar(32);not null"`
	SelectedTokenDecimals int       `gorm:"not null"`
	QuotedAmount          string    `gorm:"type:decimal(78,0);not null"`
	QuoteRate             string    `gorm:"type:decimal(78,18);not null"`
	PriceSource           string    `gorm:"type:varchar(128);not null"`
	Route                 string    `gorm:"type:text;not null"`
	SlippageBps           int       `gorm:"not null;default:0"`
	RateTimestamp         time.Time `gorm:"not null"`
	ExpiresAt             time.Time `gorm:"not null;index"`
	Status                string    `gorm:"type:varchar(32);not null;index"`
	UsedAt                *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (PaymentQuote) TableName() string {
	return "payment_quotes"
}

type PartnerPaymentSession struct {
	ID                    uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	MerchantID            uuid.UUID  `gorm:"type:uuid;not null;index"`
	QuoteID               *uuid.UUID `gorm:"type:uuid;index"`
	PaymentRequestID      *uuid.UUID `gorm:"type:uuid;index"`
	InvoiceCurrency       string     `gorm:"type:varchar(32);not null"`
	InvoiceAmount         string     `gorm:"type:decimal(78,0);not null"`
	SelectedChainID       string     `gorm:"type:varchar(64);not null"`
	SelectedTokenAddress  string     `gorm:"type:varchar(128);not null"`
	SelectedTokenSymbol   string     `gorm:"type:varchar(32);not null"`
	SelectedTokenDecimals int        `gorm:"not null"`
	DestChain             string     `gorm:"type:varchar(64);not null"`
	DestToken             string     `gorm:"type:varchar(128);not null"`
	DestWallet            string     `gorm:"type:varchar(128);not null"`
	PaymentAmount         string     `gorm:"type:decimal(78,0);not null"`
	PaymentAmountDecimals int        `gorm:"not null"`
	Status                string     `gorm:"type:varchar(32);not null;index"`
	ChannelUsed           *string    `gorm:"type:varchar(32)"`
	PaymentCode           string     `gorm:"type:text;not null"`
	PaymentURL            string     `gorm:"type:text;not null"`
	InstructionTo         string     `gorm:"type:varchar(128)"`
	InstructionValue      string     `gorm:"type:varchar(128)"`
	InstructionDataHex    string     `gorm:"type:text"`
	InstructionDataBase58 string     `gorm:"type:text"`
	InstructionDataBase64 string     `gorm:"type:text"`
	QuoteRate             *string    `gorm:"type:decimal(78,18)"`
	QuoteSource           *string    `gorm:"type:varchar(128)"`
	QuoteRoute            *string    `gorm:"type:text"`
	QuoteExpiresAt        *time.Time
	QuoteSnapshotJSON     string    `gorm:"type:jsonb"`
	ExpiresAt             time.Time `gorm:"not null;index"`
	PaidTxHash            *string   `gorm:"type:varchar(128)"`
	PaidChainID           *string   `gorm:"type:varchar(64)"`
	PaidTokenAddress      *string   `gorm:"type:varchar(128)"`
	PaidAmount            *string   `gorm:"type:decimal(78,0)"`
	PaidSenderAddress     *string   `gorm:"type:varchar(128)"`
	CompletedAt           *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (PartnerPaymentSession) TableName() string {
	return "partner_payment_sessions"
}
