package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
)

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "PENDING"
	PaymentStatusProcessing PaymentStatus = "PROCESSING"
	PaymentStatusCompleted  PaymentStatus = "COMPLETED"
	PaymentStatusFailed     PaymentStatus = "FAILED"
	PaymentStatusRefunded   PaymentStatus = "REFUNDED"
)

// PaymentEventType represents payment event type
type PaymentEventType string

const (
	PaymentEventTypeCreated           PaymentEventType = "CREATED"
	PaymentEventTypeDestinationTxHash PaymentEventType = "DESTINATION_TX_HASH"
	PaymentEventTypeCompleted         PaymentEventType = "COMPLETED"
	PaymentEventTypeFailed            PaymentEventType = "FAILED"
)

// Payment represents a payment entity
type Payment struct {
	ID                  uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	SenderID            *uuid.UUID    `json:"senderId"`
	MerchantID          *uuid.UUID    `json:"merchantId,omitempty"`
	BridgeID            *uuid.UUID    `json:"bridgeId,omitempty"`
	SourceChainID       uuid.UUID     `json:"sourceChainId"`
	DestChainID         uuid.UUID     `json:"destChainId"`
	SourceTokenID       *uuid.UUID    `json:"sourceTokenId"`
	DestTokenID         *uuid.UUID    `json:"destTokenId"`
	SourceTokenAddress  string        `json:"sourceTokenAddress"`
	DestTokenAddress    string        `json:"destTokenAddress"`
	SenderAddress       string        `json:"senderAddress"`
	DestAddress         string        `json:"destAddress"`
	SourceAmount        string        `json:"sourceAmount" gorm:"type:decimal(36,18)"`
	DestAmount          null.String   `json:"destAmount,omitempty" gorm:"type:decimal(36,18)"`
	FeeAmount           string        `json:"feeAmount" gorm:"type:decimal(36,18)"`
	TotalCharged        string        `json:"totalCharged" gorm:"type:decimal(36,18)"`
	ReceiverAddress     string        `json:"receiverAddress"`
	Status              PaymentStatus `json:"status"`
	SourceTxHash        null.String   `json:"sourceTxHash,omitempty"`
	DestTxHash          null.String   `json:"destTxHash,omitempty"`
	RefundTxHash        null.String   `json:"refundTxHash,omitempty"`
	CrossChainMessageID null.String   `json:"crossChainMessageId,omitempty"`
	ExpiresAt           *time.Time    `json:"expiresAt,omitempty"`
	CreatedAt           time.Time     `json:"createdAt"`
	UpdatedAt           time.Time     `json:"updatedAt"`
	DeletedAt           *time.Time    `json:"-"`

	// Joins
	SourceChain *Chain         `json:"sourceChain,omitempty" gorm:"foreignKey:SourceChainID"`
	DestChain   *Chain         `json:"destChain,omitempty" gorm:"foreignKey:DestChainID"`
	SourceToken *Token         `json:"sourceToken,omitempty" gorm:"foreignKey:SourceTokenID"`
	DestToken   *Token         `json:"destToken,omitempty" gorm:"foreignKey:DestTokenID"`
	Bridge      *PaymentBridge `json:"bridge,omitempty" gorm:"foreignKey:BridgeID"`
}

// PaymentBridge represents the bridge provider (CCIP, Hyperlane)
type PaymentBridge struct {
	ID   uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	Name string    `json:"name"`
}

// BridgeConfig represents routing config for a source/destination chain pair.
type BridgeConfig struct {
	ID            uuid.UUID      `json:"id"`
	BridgeID      uuid.UUID      `json:"bridgeId"`
	SourceChainID uuid.UUID      `json:"sourceChainId"`
	DestChainID   uuid.UUID      `json:"destChainId"`
	RouterAddress string         `json:"routerAddress"`
	FeePercentage string         `json:"feePercentage"`
	Config        string         `json:"config"`
	IsActive      bool           `json:"isActive"`
	Bridge        *PaymentBridge `json:"bridge,omitempty"`
}

// FeeConfig represents fee rules for specific chain/token pair.
type FeeConfig struct {
	ID                 uuid.UUID  `json:"id"`
	ChainID            uuid.UUID  `json:"chainId"`
	TokenID            uuid.UUID  `json:"tokenId"`
	PlatformFeePercent string     `json:"platformFeePercent"`
	FixedBaseFee       string     `json:"fixedBaseFee"`
	MinFee             string     `json:"minFee"`
	MaxFee             *string    `json:"maxFee,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	DeletedAt          *time.Time `json:"-"`
}

// CreatePaymentInput represents input for creating a payment
type CreatePaymentInput struct {
	SourceChainID      string `json:"sourceChainId" binding:"required"` // UUID or NetworkID? Likely NetworkID in API
	DestChainID        string `json:"destChainId" binding:"required"`   // Likely NetworkID in API
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
	SourceChainID  string        `json:"sourceChainId"` // Network ID
	DestChainID    string        `json:"destChainId"`   // Network ID
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
	TotalFee    string `json:"totalFee"`
	NetAmount   string `json:"netAmount"`
}

// PaymentEvent represents a payment event
type PaymentEvent struct {
	ID          uuid.UUID        `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	PaymentID   uuid.UUID        `json:"paymentId"`
	EventType   PaymentEventType `json:"eventType"`
	ChainID     *uuid.UUID       `json:"chainId,omitempty"`
	TxHash      string           `json:"txHash"`
	BlockNumber int64            `json:"blockNumber,omitempty"`
	Metadata    interface{}      `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt   time.Time        `json:"createdAt"`
}

type CreatePaymentAppInput struct {
	SourceChainID       string `json:"sourceChainId" binding:"required"`
	DestChainID         string `json:"destChainId" binding:"required"`
	SourceTokenAddress  string `json:"sourceTokenAddress" binding:"required"`
	DestTokenAddress    string `json:"destTokenAddress" binding:"required"`
	Amount              string `json:"amount" binding:"required"`
	Decimals            int    `json:"decimals" binding:"required"`
	SenderWalletAddress string `json:"senderWalletAddress" binding:"required"`
	ReceiverAddress     string `json:"receiverAddress" binding:"required"`
}
