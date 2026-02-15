package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoutePolicy struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	SourceChainID     uuid.UUID `gorm:"type:uuid;not null;index"`
	DestChainID       uuid.UUID `gorm:"type:uuid;not null;index"`
	DefaultBridgeType int16     `gorm:"type:smallint;not null;default:0"`
	FallbackMode      string    `gorm:"type:varchar(32);not null;default:'strict'"`
	FallbackOrder     string    `gorm:"type:jsonb;not null;default:'[0]'"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

func (RoutePolicy) TableName() string {
	return "route_policies"
}

type LayerZeroConfig struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	SourceChainID uuid.UUID `gorm:"type:uuid;not null;index"`
	DestChainID   uuid.UUID `gorm:"type:uuid;not null;index"`
	DstEID        uint32    `gorm:"type:integer;not null"`
	PeerHex       string    `gorm:"type:varchar(66);not null"`
	OptionsHex    string    `gorm:"type:text;not null;default:'0x'"`
	IsActive      bool      `gorm:"not null;default:true"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (LayerZeroConfig) TableName() string {
	return "layerzero_configs"
}
