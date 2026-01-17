package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
)

// SmartContract represents a deployed smart contract
type SmartContract struct {
	ID              uuid.UUID   `json:"id"`
	Name            string      `json:"name"`
	ChainID         string      `json:"chainId"`         // CAIP-2 format: namespace:chainId
	ContractAddress string      `json:"contractAddress"`
	ABI             interface{} `json:"abi"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
	DeletedAt       null.Time   `json:"-"`
}

// CreateSmartContractInput represents input for creating a smart contract record
type CreateSmartContractInput struct {
	Name            string      `json:"name" binding:"required,min=1,max=100"`
	ChainID         string      `json:"chainId" binding:"required"`
	ContractAddress string      `json:"contractAddress" binding:"required"`
	ABI             interface{} `json:"abi" binding:"required"`
}

// UpdateSmartContractInput represents input for updating a smart contract
type UpdateSmartContractInput struct {
	Name string      `json:"name,omitempty"`
	ABI  interface{} `json:"abi,omitempty"`
}
