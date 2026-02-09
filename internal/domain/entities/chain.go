package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
)

// ChainType represents blockchain type
type ChainType string

const (
	ChainTypeEVM       ChainType = "EVM"
	ChainTypeSVM       ChainType = "SVM"
	ChainTypeSubstrate ChainType = "SUBSTRATE"
)

// Chain represents a blockchain
type Chain struct {
	ID                 uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	NetworkID          string    `json:"networkId" gorm:"uniqueIndex;not null"` // External Chain ID (e.g. "1", "solana:5ey...")
	Namespace          string    `json:"namespace"`
	Name               string    `json:"name"`
	ChainType          ChainType `json:"chainType"`
	RPCURL             string    `json:"rpcUrl"` // Deprecated: use RPCURLs[0]
	RPCURLs            []string  `json:"rpcUrls" gorm:"type:text[]"`
	ExplorerURL        string    `json:"explorerUrl,omitempty"`
	Symbol             string    `json:"symbol,omitempty"`
	LogoURL            string    `json:"logoUrl,omitempty"`
	ContractAddress    string    `json:"contractAddress,omitempty"`
	CCIPRouterAddress  string    `json:"ccipRouterAddress,omitempty"`
	HyperbridgeGateway string    `json:"hyperbridgeGateway,omitempty"`
	StateMachineID     string    `json:"stateMachineId,omitempty"`
	IsActive           bool      `json:"isActive"`
	CreatedAt          time.Time `json:"createdAt"`
}

// ChainRPC represents a blockchain RPC endpoint
type ChainRPC struct {
	ID          uuid.UUID  `json:"id"`
	ChainID     uuid.UUID  `json:"chainId"`
	URL         string     `json:"url"`
	Priority    int        `json:"priority"`
	IsActive    bool       `json:"isActive"`
	LastErrorAt *time.Time `json:"lastErrorAt,omitempty"`
	ErrorCount  int        `json:"errorCount"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`

	// Joined fields
	Chain *Chain `json:"chain,omitempty"`
}

// GetCAIP2ID returns the CAIP-2 formatted chain ID
func (c *Chain) GetCAIP2ID() string {
	return fmt.Sprintf("%s:%s", c.Namespace, c.NetworkID)
}

// TokenType represents token type
type TokenType string

const (
	TokenTypeERC20  TokenType = "ERC20"
	TokenTypeNative TokenType = "NATIVE"
	TokenTypeStable TokenType = "STABLE"
)

// Token represents a token
type Token struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ChainID         uuid.UUID  `json:"chainId" gorm:"type:uuid;not null"` // Updated to UUID
	Chain           *Chain     `json:"chain,omitempty" gorm:"foreignKey:ChainID"`
	Symbol          string     `json:"symbol" gorm:"not null"`
	Name            string     `json:"name" gorm:"not null"`
	Decimals        int        `json:"decimals" gorm:"not null;default:18"`
	ContractAddress string     `json:"contractAddress"`
	Type            TokenType  `json:"type" gorm:"type:varchar(20);not null;default:'ERC20'"`
	LogoURL         string     `json:"logoUrl,omitempty"`
	IsStablecoin    bool       `json:"isStablecoin"`
	IsActive        bool       `json:"isActive" gorm:"default:true"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	DeletedAt       *time.Time `json:"deletedAt,omitempty" gorm:"index"`
}

// SupportedToken represents a token supported on a chain
type SupportedToken struct {
	ID              uuid.UUID   `json:"id"`
	ChainID         uuid.UUID   `json:"chainId"`
	TokenID         uuid.UUID   `json:"tokenId"`
	ContractAddress string      `json:"contractAddress"`
	IsActive        bool        `json:"isActive"`
	MinAmount       string      `json:"minAmount,omitempty"`
	MaxAmount       null.String `json:"maxAmount,omitempty"`
	CreatedAt       time.Time   `json:"createdAt"`

	// Joined fields
	Token *Token `json:"token,omitempty"`
	Chain *Chain `json:"chain,omitempty"`
}

// Wallet represents a user wallet
type Wallet struct {
	ID         uuid.UUID   `json:"id"`
	UserID     null.String `json:"userId,omitempty"`
	MerchantID null.String `json:"merchantId,omitempty"`
	ChainID    uuid.UUID   `json:"chainId"`
	Address    string      `json:"address"`
	IsPrimary  bool        `json:"isPrimary"`
	CreatedAt  time.Time   `json:"createdAt"`
	DeletedAt  null.Time   `json:"-"`
}

// ConnectWalletInput represents input for connecting a wallet
type ConnectWalletInput struct {
	ChainID   string `json:"chainId" binding:"required"` // Can be NetworkID or UUID, handler logic decides
	Address   string `json:"address" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	Message   string `json:"message" binding:"required"`
}
