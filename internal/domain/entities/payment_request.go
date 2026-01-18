package entities

import (
	"time"

	"github.com/google/uuid"
)

// PaymentRequestStatus represents the status of a payment request
type PaymentRequestStatus string

const (
	PaymentRequestStatusPending   PaymentRequestStatus = "pending"
	PaymentRequestStatusCompleted PaymentRequestStatus = "completed"
	PaymentRequestStatusExpired   PaymentRequestStatus = "expired"
	PaymentRequestStatusCancelled PaymentRequestStatus = "cancelled"
)

// PaymentRequest represents a merchant's payment request
type PaymentRequest struct {
	ID           uuid.UUID            `json:"id"`
	MerchantID   uuid.UUID            `json:"merchantId"`
	WalletID     uuid.UUID            `json:"walletId"`
	ChainID      string               `json:"chainId"` // CAIP-2 format
	TokenAddress string               `json:"tokenAddress"`
	Amount       string               `json:"amount"` // In smallest unit
	Decimals     int                  `json:"decimals"`
	Description  string               `json:"description,omitempty"`
	Status       PaymentRequestStatus `json:"status"`
	ExpiresAt    time.Time            `json:"expiresAt"`
	TxHash       string               `json:"txHash,omitempty"`
	PayerAddress string               `json:"payerAddress,omitempty"`
	CompletedAt  *time.Time           `json:"completedAt,omitempty"`
	CreatedAt    time.Time            `json:"createdAt"`
	UpdatedAt    time.Time            `json:"updatedAt"`
	DeletedAt    *time.Time           `json:"-"`
}

// PaymentRequestTxData contains the transaction data for paying a request
type PaymentRequestTxData struct {
	RequestID       string `json:"requestId"`
	ContractAddress string `json:"contractAddress"`
	ChainID         string `json:"chainId"` // CAIP-2 format
	Amount          string `json:"amount"`  // In smallest unit
	Decimals        int    `json:"decimals"`  
	
	// EVM specific
	Hex string `json:"hex,omitempty"`
	
	// SVM specific
	Base64 string `json:"base64,omitempty"`
}

// RpcEndpoint represents an RPC endpoint for a chain
type RpcEndpoint struct {
	ID          uuid.UUID  `json:"id"`
	ChainID     int        `json:"chainId"`
	URL         string     `json:"url"`
	Priority    int        `json:"priority"`
	IsActive    bool       `json:"isActive"`
	LastErrorAt *time.Time `json:"lastErrorAt,omitempty"`
	ErrorCount  int        `json:"errorCount"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// BackgroundJob represents a background job for async processing
type BackgroundJob struct {
	ID           uuid.UUID  `json:"id"`
	JobType      string     `json:"jobType"`
	Payload      string     `json:"payload"` // JSON
	Status       string     `json:"status"`  // pending, processing, completed, failed
	Attempts     int        `json:"attempts"`
	MaxAttempts  int        `json:"maxAttempts"`
	ScheduledAt  time.Time  `json:"scheduledAt"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}
