package entities

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"gorm.io/gorm"
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
	ID             uuid.UUID  `json:"uuid" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ChainID        string     `json:"id" gorm:"uniqueIndex;not null"` // Map blockchain ID to "id" for FE
	Name           string     `json:"name"`
	Type           ChainType  `json:"chainType" gorm:"type:varchar(50);not null"` // Map Type to "chainType"
	ImageURL       string     `json:"imageUrl,omitempty"`
	IsActive       bool       `json:"isActive"`
	IsTestnet      bool       `json:"isTestnet"`
	CurrencySymbol string     `json:"currencySymbol"`
	ExplorerURL    string     `json:"explorerUrl,omitempty"`
	RPCURL         string     `json:"rpcUrl"` // Main RPC
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty" gorm:"index"`

	// Relationships
	RPCs []ChainRPC `json:"rpcs,omitempty" gorm:"foreignKey:ChainID"`
}

// ChainRPC represents a blockchain RPC endpoint
type ChainRPC struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ChainID     uuid.UUID      `json:"chainId"`
	URL         string         `json:"url"`
	Priority    int            `json:"priority"`
	IsActive    bool           `json:"isActive"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	LastErrorAt *time.Time     `json:"lastErrorAt,omitempty"`
	ErrorCount  int            `json:"errorCount"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Joined fields
	Chain *Chain `json:"chain,omitempty"`
}

// GetCAIP2ID returns the CAIP-2 formatted chain ID
// Deprecated: Logic moved to specific adapters.
// However, useful helper: if ChainType is EVM, generic logic applies.
func (c *Chain) GetCAIP2ID() string {
	raw := strings.TrimSpace(c.ChainID)
	if strings.Contains(raw, ":") {
		return raw
	}

	// Simple heuristic. ideally implementation details should handle this.
	// For EVM: eip155:ChainID
	if c.Type == ChainTypeEVM {
		return fmt.Sprintf("eip155:%s", raw)
	}
	// For SVM: solana:ChainID?
	// This might need refinement based on exact storage of ChainID for Solana.
	if c.Type == ChainTypeSVM {
		return fmt.Sprintf("solana:%s", raw)
	}
	return raw
}

// TokenType represents token type
type TokenType string

const (
	TokenTypeERC20  TokenType = "ERC20"
	TokenTypeNative TokenType = "NATIVE"
	TokenTypeSPL    TokenType = "SPL"
)

// Token represents a token (now Chain-Specific)
type Token struct {
	ID              uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
	ChainUUID       uuid.UUID   `json:"chainId" gorm:"type:uuid;not null;column:chain_id"` // Keep internal UUID mapping
	BlockchainID    string      `json:"blockchainId" gorm:"-"`                             // Virtual field for FE
	Chain           *Chain      `json:"chain,omitempty" gorm:"foreignKey:ChainUUID"`
	Name            string      `json:"name" gorm:"not null"`
	Symbol          string      `json:"symbol" gorm:"not null"`
	Decimals        int         `json:"decimals" gorm:"not null;default:18"`
	Type            TokenType   `json:"type" gorm:"type:varchar(20);not null;default:'ERC20'"`
	ContractAddress string      `json:"contractAddress"` // Renamed from Address
	LogoURL         string      `json:"logoUrl,omitempty"`
	IsActive        bool        `json:"isActive" gorm:"default:true"`
	IsNative        bool        `json:"isNative" gorm:"default:false"`
	IsStablecoin    bool        `json:"isStablecoin" gorm:"default:false"`
	MinAmount       string      `json:"minAmount" gorm:"type:decimal(36,18);default:0"`
	MaxAmount       null.String `json:"maxAmount,omitempty" gorm:"type:decimal(36,18)"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
	DeletedAt       *time.Time  `json:"deletedAt,omitempty" gorm:"index"`
}
