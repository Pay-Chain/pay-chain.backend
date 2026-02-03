package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type SmartContract struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name            string         `gorm:"type:varchar(100);not null"`
	ChainID         string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_chain_contract"` // CAIP-2 format
	ContractAddress string         `gorm:"type:varchar(66);not null;uniqueIndex:idx_chain_contract"`
	ABI             string         `gorm:"type:jsonb;not null"` // Storing JSON as string/bytes for GORM or using specific JSON type
	Type            string         `gorm:"type:varchar(50);not null;default:'GATEWAY'"`
	Version         string         `gorm:"type:varchar(20);not null;default:'1.0.0'"`
	DeployerAddress string         `gorm:"type:varchar(66)"`
	StartBlock      int64          `gorm:"default:0"`
	Metadata        string         `gorm:"type:jsonb;default:'{}'"`
	IsActive        bool           `gorm:"default:true"`
	DestinationMap  pq.StringArray `gorm:"type:text[];default:'{}'"` // For destination contract mapping
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}
