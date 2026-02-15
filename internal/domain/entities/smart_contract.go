package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
)

// SmartContractType represents the type of smart contract
type SmartContractType string

const (
	ContractTypeGateway            SmartContractType = "GATEWAY"
	ContractTypeVault              SmartContractType = "VAULT"
	ContractTypeRouter             SmartContractType = "ROUTER"
	ContractTypeTokenRegistry      SmartContractType = "TOKEN_REGISTRY"
	ContractTypeTokenSwapper       SmartContractType = "TOKEN_SWAPPER"
	ContractTypeAdapterCCIP        SmartContractType = "ADAPTER_CCIP"
	ContractTypeAdapterHyperbridge SmartContractType = "ADAPTER_HYPERBRIDGE"
	ContractTypeAdapterLayerZero   SmartContractType = "ADAPTER_LAYERZERO"
	ContractTypeReceiverLayerZero  SmartContractType = "RECEIVER_LAYERZERO"
	ContractTypePool               SmartContractType = "POOL" // DEX Pool
	ContractTypeMock               SmartContractType = "MOCK" // For testing
)

// SmartContract represents a deployed smart contract
type SmartContract struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Type            SmartContractType `json:"type"`
	Version         string            `json:"version"`   // Semver e.g. "1.0.0"
	ChainUUID       uuid.UUID         `json:"chainUuid"` // Internal UUID
	BlockchainID    string            `json:"chainId"`   // Frontend-compatible Blockchain ID (e.g. "1")
	ContractAddress string            `json:"contractAddress"`
	DeployerAddress null.String       `json:"deployerAddress,omitempty"`
	Token0Address   null.String       `json:"token0Address,omitempty"`
	Token1Address   null.String       `json:"token1Address,omitempty"`
	FeeTier         null.Int          `json:"feeTier,omitempty"`
	HookAddress     null.String       `json:"hookAddress,omitempty"`
	StartBlock      uint64            `json:"startBlock"` // Block number deployment/indexing start
	ABI             interface{}       `json:"abi"`
	Metadata        null.JSON         `json:"metadata,omitempty"` // Store extra config like gas limits, timeouts
	IsActive        bool              `json:"isActive"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	DeletedAt       null.Time         `json:"-"`
}

// CreateSmartContractInput represents input for creating a smart contract record
type CreateSmartContractInput struct {
	Name            string                 `json:"name" binding:"required,min=1,max=100"`
	Type            SmartContractType      `json:"type" binding:"required"`
	Version         string                 `json:"version" binding:"required"`
	ChainID         string                 `json:"chainId" binding:"required"`
	ContractAddress string                 `json:"contractAddress" binding:"required"`
	DeployerAddress string                 `json:"deployerAddress,omitempty"`
	Token0Address   string                 `json:"token0Address,omitempty"`
	Token1Address   string                 `json:"token1Address,omitempty"`
	FeeTier         int                    `json:"feeTier,omitempty"`
	HookAddress     string                 `json:"hookAddress,omitempty"`
	StartBlock      uint64                 `json:"startBlock" binding:"required"`
	ABI             interface{}            `json:"abi" binding:"required"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	IsActive        *bool                  `json:"isActive,omitempty"`
}

// UpdateSmartContractInput represents input for updating a smart contract
type UpdateSmartContractInput struct {
	Name            string                 `json:"name,omitempty"`
	Type            SmartContractType      `json:"type,omitempty"`
	Version         string                 `json:"version,omitempty"`
	ChainID         string                 `json:"chainId,omitempty"`
	ContractAddress string                 `json:"contractAddress,omitempty"`
	DeployerAddress string                 `json:"deployerAddress,omitempty"`
	Token0Address   string                 `json:"token0Address,omitempty"`
	Token1Address   string                 `json:"token1Address,omitempty"`
	FeeTier         *int                   `json:"feeTier,omitempty"`
	HookAddress     string                 `json:"hookAddress,omitempty"`
	StartBlock      *uint64                `json:"startBlock,omitempty"`
	ABI             interface{}            `json:"abi,omitempty"`
	IsActive        *bool                  `json:"isActive,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// FilterSmartContractInput represents filter options for listing contracts
type FilterSmartContractInput struct {
	ChainID  string            `form:"chainId"`
	Type     SmartContractType `form:"type"`
	IsActive *bool             `form:"isActive"`
}
