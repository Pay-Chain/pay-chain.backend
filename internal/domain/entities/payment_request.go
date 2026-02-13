package entities

import (
	"time"

	"github.com/google/uuid"
)

// PaymentRequestStatus represents the status of a payment request
type PaymentRequestStatus string

const (
	PaymentRequestStatusPending   PaymentRequestStatus = "PENDING"
	PaymentRequestStatusCompleted PaymentRequestStatus = "COMPLETED"
	PaymentRequestStatusExpired   PaymentRequestStatus = "EXPIRED"
	PaymentRequestStatusCancelled PaymentRequestStatus = "CANCELLED"
)

// PaymentRequest represents a merchant's payment request
type PaymentRequest struct {
	ID            uuid.UUID            `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	MerchantID    uuid.UUID            `json:"merchantId"`
	ChainID       uuid.UUID            `json:"chainId"`   // Internal UUID
	NetworkID     string               `json:"networkId"` // External Chain ID (CAIP-2)
	TokenID       uuid.UUID            `json:"tokenId"`   // Internal UUID
	TokenAddress  string               `json:"tokenAddress"`
	WalletAddress string               `json:"walletAddress" gorm:"column:wallet_address;not null"`
	PayerAddress  string               `json:"payerAddress,omitempty"`
	Amount        string               `json:"amount" gorm:"type:decimal(36,18)"`
	Decimals      int                  `json:"decimals"`
	Description   string               `json:"description,omitempty"`
	Status        PaymentRequestStatus `json:"status"`
	ExpiresAt     time.Time            `json:"expiresAt"`
	TxHash        string               `json:"txHash,omitempty"`
	CompletedAt   *time.Time           `json:"completedAt,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
	UpdatedAt     time.Time            `json:"updatedAt"`
	DeletedAt     *time.Time           `json:"-"`

	// Joins
	Merchant *Merchant `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
	Chain    *Chain    `json:"chain,omitempty" gorm:"foreignKey:ChainID"`
	Token    *Token    `json:"token,omitempty" gorm:"foreignKey:TokenID"`
}

// PaymentRequestTxData contains the transaction data for paying a request
type PaymentRequestTxData struct {
	RequestID       string `json:"requestId"`
	ContractAddress string `json:"contractAddress"`
	ChainID         string `json:"chainId"` // Network ID
	Amount          string `json:"amount"`
	Decimals        int    `json:"decimals"`
	To              string `json:"to,omitempty"`
	ProgramID       string `json:"programId,omitempty"`

	// EVM specific
	Hex string `json:"hex,omitempty"`

	// SVM specific
	Base64 string `json:"base64,omitempty"` // backward compatibility
	Base58 string `json:"base58,omitempty"`
}

// JobStatus represents background job status
type JobStatus string

const (
	JobStatusPending    JobStatus = "PENDING"
	JobStatusProcessing JobStatus = "PROCESSING"
	JobStatusCompleted  JobStatus = "COMPLETED"
	JobStatusFailed     JobStatus = "FAILED"
)

// BackgroundJob represents a background job for async processing
type BackgroundJob struct {
	ID           uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	JobType      string      `json:"jobType"`
	Payload      interface{} `json:"payload" gorm:"type:jsonb"` // JSON
	Status       JobStatus   `json:"status"`
	Attempts     int         `json:"attempts"`
	MaxAttempts  int         `json:"maxAttempts"`
	ScheduledAt  time.Time   `json:"scheduledAt"`
	StartedAt    *time.Time  `json:"startedAt,omitempty"`
	CompletedAt  *time.Time  `json:"completedAt,omitempty"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	DeletedAt    *time.Time  `json:"-"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
}
