package entities

import (
	"time"

	"github.com/google/uuid"
)

type BridgeFallbackMode string

const (
	BridgeFallbackModeStrict       BridgeFallbackMode = "strict"
	BridgeFallbackModeAutoFallback BridgeFallbackMode = "auto_fallback"
)

type RoutePolicy struct {
	ID                uuid.UUID          `json:"id"`
	SourceChainID     uuid.UUID          `json:"sourceChainId"`
	DestChainID       uuid.UUID          `json:"destChainId"`
	DefaultBridgeType uint8              `json:"defaultBridgeType"`
	FallbackMode      BridgeFallbackMode `json:"fallbackMode"`
	FallbackOrder     []uint8            `json:"fallbackOrder"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
	DeletedAt         *time.Time         `json:"-"`
}

type LayerZeroConfig struct {
	ID            uuid.UUID  `json:"id"`
	SourceChainID uuid.UUID  `json:"sourceChainId"`
	DestChainID   uuid.UUID  `json:"destChainId"`
	DstEID        uint32     `json:"dstEid"`
	PeerHex       string     `json:"peerHex"`
	OptionsHex    string     `json:"optionsHex"`
	IsActive      bool       `json:"isActive"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"-"`
}
