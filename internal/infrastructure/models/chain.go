package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Chain struct {
	ID             int    `gorm:"primaryKey;autoIncrement:false"` // Chain ID is manually set (e.g., 11155111)
	Namespace      string `gorm:"type:varchar(50);not null"`
	Name           string `gorm:"type:varchar(100);not null"`
	ChainType      string `gorm:"type:varchar(50);not null;default:'EVM'"`
	RPCURL         string `gorm:"type:text;column:rpc_url"` // Legacy column, kept for backward compatibility
	ExplorerURL    string `gorm:"type:text"`
	IsActive       bool   `gorm:"default:true"`
	StateMachineID string `gorm:"type:varchar(100)"` // e.g., 'isis-local', 'suave-local'
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	// Relations
	RPCs           []ChainRPC      `gorm:"foreignKey:ChainID"`
	SmartContracts []SmartContract `gorm:"foreignKey:ChainID"`
}

type ChainRPC struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ChainID   int       `gorm:"not null;index"`
	URL       string    `gorm:"type:text;not null"`
	Priority  int       `gorm:"default:0"`
	IsActive  bool      `gorm:"default:true;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
