package entities

import (
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
	ID                  int       `json:"id"`
	Namespace           string    `json:"namespace"`
	Name                string    `json:"name"`
	ChainType           ChainType `json:"chainType"`
	RPCURL              string    `json:"rpcUrl"`
	ExplorerURL         string    `json:"explorerUrl,omitempty"`
	ContractAddress     string    `json:"contractAddress,omitempty"`
	CCIPRouterAddress   string    `json:"ccipRouterAddress,omitempty"`
	HyperbridgeGateway  string    `json:"hyperbridgeGateway,omitempty"`
	IsActive            bool      `json:"isActive"`
	CreatedAt           time.Time `json:"createdAt"`
}

// GetCAIP2ID returns the CAIP-2 formatted chain ID
func (c *Chain) GetCAIP2ID() string {
	return c.Namespace + ":" + string(rune(c.ID))
}

// Token represents a token
type Token struct {
	ID          uuid.UUID `json:"id"`
	Symbol      string    `json:"symbol"`
	Name        string    `json:"name"`
	Decimals    int       `json:"decimals"`
	LogoURL     string    `json:"logoUrl,omitempty"`
	IsStablecoin bool     `json:"isStablecoin"`
	CreatedAt   time.Time `json:"createdAt"`
}

// SupportedToken represents a token supported on a chain
type SupportedToken struct {
	ID              uuid.UUID   `json:"id"`
	ChainID         int         `json:"chainId"`
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
	ChainID    int         `json:"chainId"`
	Address    string      `json:"address"`
	IsPrimary  bool        `json:"isPrimary"`
	CreatedAt  time.Time   `json:"createdAt"`
	DeletedAt  null.Time   `json:"-"`
}

// ConnectWalletInput represents input for connecting a wallet
type ConnectWalletInput struct {
	ChainID   string `json:"chainId" binding:"required"`
	Address   string `json:"address" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	Message   string `json:"message" binding:"required"`
}
