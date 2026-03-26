package entities

import (
	"time"

	"github.com/google/uuid"
)

type PaymentQuoteStatus string

const (
	PaymentQuoteStatusActive    PaymentQuoteStatus = "ACTIVE"
	PaymentQuoteStatusUsed      PaymentQuoteStatus = "USED"
	PaymentQuoteStatusExpired   PaymentQuoteStatus = "EXPIRED"
	PaymentQuoteStatusCancelled PaymentQuoteStatus = "CANCELLED"
)

type PartnerPaymentSessionStatus string

const (
	PartnerPaymentSessionStatusPending   PartnerPaymentSessionStatus = "PENDING"
	PartnerPaymentSessionStatusCompleted PartnerPaymentSessionStatus = "COMPLETED"
	PartnerPaymentSessionStatusExpired   PartnerPaymentSessionStatus = "EXPIRED"
	PartnerPaymentSessionStatusFailed    PartnerPaymentSessionStatus = "FAILED"
	PartnerPaymentSessionStatusCancelled PartnerPaymentSessionStatus = "CANCELLED"
)

type PaymentQuote struct {
	ID                    uuid.UUID          `json:"id"`
	MerchantID            uuid.UUID          `json:"merchantId"`
	InvoiceCurrency       string             `json:"invoiceCurrency"`
	InvoiceAmount         string             `json:"invoiceAmount"`
	SelectedChainID       string             `json:"selectedChainId"`
	SelectedTokenAddress  string             `json:"selectedTokenAddress"`
	SelectedTokenSymbol   string             `json:"selectedTokenSymbol"`
	SelectedTokenDecimals int                `json:"selectedTokenDecimals"`
	QuotedAmount          string             `json:"quotedAmount"`
	QuoteRate             string             `json:"quoteRate"`
	PriceSource           string             `json:"priceSource"`
	Route                 string             `json:"route"`
	SlippageBps           int                `json:"slippageBps"`
	RateTimestamp         time.Time          `json:"rateTimestamp"`
	ExpiresAt             time.Time          `json:"expiresAt"`
	Status                PaymentQuoteStatus `json:"status"`
	UsedAt                *time.Time         `json:"usedAt,omitempty"`
	CreatedAt             time.Time          `json:"createdAt"`
	UpdatedAt             time.Time          `json:"updatedAt"`
}

type PartnerPaymentSession struct {
	ID                    uuid.UUID                   `json:"id"`
	MerchantID            uuid.UUID                   `json:"merchantId"`
	QuoteID               *uuid.UUID                  `json:"quoteId,omitempty"`
	PaymentRequestID      *uuid.UUID                  `json:"paymentRequestId,omitempty"`
	InvoiceCurrency       string                      `json:"invoiceCurrency"`
	InvoiceAmount         string                      `json:"invoiceAmount"`
	SelectedChainID       string                      `json:"selectedChainId"`
	SelectedTokenAddress  string                      `json:"selectedTokenAddress"`
	SelectedTokenSymbol   string                      `json:"selectedTokenSymbol"`
	SelectedTokenDecimals int                         `json:"selectedTokenDecimals"`
	DestChain             string                      `json:"destChain"`
	DestToken             string                      `json:"destToken"`
	DestWallet            string                      `json:"destWallet"`
	PaymentAmount         string                      `json:"paymentAmount"`
	PaymentAmountDecimals int                         `json:"paymentAmountDecimals"`
	Status                PartnerPaymentSessionStatus `json:"status"`
	ChannelUsed           *string                     `json:"channelUsed,omitempty"`
	PaymentCode           string                      `json:"paymentCode"`
	PaymentURL            string                      `json:"paymentUrl"`
	InstructionTo         string                      `json:"instructionTo,omitempty"`
	InstructionValue      string                      `json:"instructionValue,omitempty"`
	InstructionDataHex    string                      `json:"instructionDataHex,omitempty"`
	InstructionDataBase58 string                      `json:"instructionDataBase58,omitempty"`
	InstructionDataBase64 string                      `json:"instructionDataBase64,omitempty"`
	InstructionApprovalTo string                      `json:"instructionApprovalTo,omitempty"`
	InstructionApprovalDataHex string                 `json:"instructionApprovalDataHex,omitempty"`
	QuoteRate             *string                     `json:"quoteRate,omitempty"`
	QuoteSource           *string                     `json:"quoteSource,omitempty"`
	QuoteRoute            *string                     `json:"quoteRoute,omitempty"`
	QuoteExpiresAt        *time.Time                  `json:"quoteExpiresAt,omitempty"`
	QuoteSnapshotJSON     string                      `json:"quoteSnapshotJson,omitempty"`
	ExpiresAt             time.Time                   `json:"expiresAt"`
	PaidTxHash            *string                     `json:"paidTxHash,omitempty"`
	PaidChainID           *string                     `json:"paidChainId,omitempty"`
	PaidTokenAddress      *string                     `json:"paidTokenAddress,omitempty"`
	PaidAmount            *string                     `json:"paidAmount,omitempty"`
	PaidSenderAddress     *string                     `json:"paidSenderAddress,omitempty"`
	CompletedAt           *time.Time                  `json:"completedAt,omitempty"`
	CreatedAt             time.Time                   `json:"createdAt"`
	UpdatedAt             time.Time                   `json:"updatedAt"`
}
