package entities

import (
	"time"

	"github.com/google/uuid"
)

// Wallet represents a user's wallet
type Wallet struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	UserID     *uuid.UUID `json:"userId,omitempty"`
	MerchantID *uuid.UUID `json:"merchantId,omitempty"`
	ChainID    uuid.UUID  `json:"chainId"`
	Address    string     `json:"address"`
	Type       string     `json:"type"` // EOA, SMART_CONTRACT
	IsPrimary  bool       `json:"isPrimary" gorm:"default:false"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	DeletedAt  *time.Time `json:"-"`

	// Joins
	User     *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Merchant *Merchant `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
	Chain    *Chain    `json:"chain,omitempty" gorm:"foreignKey:ChainID"`
}

// ConnectWalletInput represents input for connecting a wallet
type ConnectWalletInput struct {
	ChainID   string `json:"chainId" binding:"required"` // The Network ID (e.g. "1")
	Address   string `json:"address" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	Message   string `json:"message" binding:"required"`
}
