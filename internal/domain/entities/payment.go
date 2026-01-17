package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
)

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// Payment represents a payment entity
type Payment struct {
	ID                  uuid.UUID     `json:"id"`
	SenderID            uuid.UUID     `json:"senderId"`
	MerchantID          null.String   `json:"merchantId,omitempty"`
	ReceiverWalletID    uuid.UUID     `json:"receiverWalletId"`
	SourceTokenID       uuid.UUID     `json:"sourceTokenId"`
	DestTokenID         uuid.UUID     `json:"destTokenId"`
	SourceAmount        string        `json:"sourceAmount"`
	DestAmount          null.String   `json:"destAmount,omitempty"`
	FeeAmount           string        `json:"feeAmount"`
	TotalCharged        string        `json:"totalCharged"`
	BridgeType          string        `json:"bridgeType"`
	Status              PaymentStatus `json:"status"`
	SourceTxHash        null.String   `json:"sourceTxHash,omitempty"`
	DestTxHash          null.String   `json:"destTxHash,omitempty"`
	RefundTxHash        null.String   `json:"refundTxHash,omitempty"`
	CrossChainMessageID null.String   `json:"crossChainMessageId,omitempty"`
	ExpiresAt           null.Time     `json:"expiresAt,omitempty"`
	CreatedAt           time.Time     `json:"createdAt"`
	UpdatedAt           time.Time     `json:"updatedAt"`
	DeletedAt           null.Time     `json:"-"`
}

// CreatePaymentInput represents input for creating a payment
type CreatePaymentInput struct {
	SourceChainID      string `json:"sourceChainId" binding:"required"`
	DestChainID        string `json:"destChainId" binding:"required"`
	SourceTokenAddress string `json:"sourceTokenAddress" binding:"required"`
	DestTokenAddress   string `json:"destTokenAddress" binding:"required"`
	Amount             string `json:"amount" binding:"required"`
	Decimals           int    `json:"decimals" binding:"required"`
	ReceiverAddress    string `json:"receiverAddress" binding:"required"`
	ReceiverMerchantID string `json:"receiverMerchantId,omitempty"`
}

// CreatePaymentResponse represents response for payment creation
type CreatePaymentResponse struct {
	PaymentID      uuid.UUID     `json:"paymentId"`
	Status         PaymentStatus `json:"status"`
	SourceChainID  string        `json:"sourceChainId"`
	DestChainID    string        `json:"destChainId"`
	SourceAmount   string        `json:"sourceAmount"`
	SourceDecimals int           `json:"sourceDecimals"`
	DestAmount     string        `json:"destAmount"`
	DestDecimals   int           `json:"destDecimals"`
	FeeAmount      string        `json:"feeAmount"`
	FeeBreakdown   FeeBreakdown  `json:"feeBreakdown"`
	BridgeType     string        `json:"bridgeType"`
	BridgeReason   string        `json:"bridgeReason"`
	ExpiresAt      time.Time     `json:"expiresAt"`
	SignatureData  interface{}   `json:"signatureData"`
}

// FeeBreakdown represents fee breakdown
type FeeBreakdown struct {
	PlatformFee string `json:"platformFee"`
	BridgeFee   string `json:"bridgeFee"`
	GasFee      string `json:"gasFee"`
}

// PaymentEvent represents a payment event
type PaymentEvent struct {
	ID        uuid.UUID   `json:"id"`
	PaymentID uuid.UUID   `json:"paymentId"`
	EventType string      `json:"eventType"`
	Chain     string      `json:"chain"`
	TxHash    string      `json:"txHash"`
	Metadata  interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time   `json:"createdAt"`
}
