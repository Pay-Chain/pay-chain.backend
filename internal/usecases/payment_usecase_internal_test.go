package usecases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
)

func TestBuildBridgeOrderFromPolicy_Strict(t *testing.T) {
	order := buildBridgeOrderFromPolicy(&entities.RoutePolicy{
		DefaultBridgeType: 1,
		FallbackMode:      entities.BridgeFallbackModeStrict,
		FallbackOrder:     []uint8{2, 0},
	})

	assert.Equal(t, []uint8{1}, order)
}

func TestBuildBridgeOrderFromPolicy_AutoFallbackDedupAndFilter(t *testing.T) {
	order := buildBridgeOrderFromPolicy(&entities.RoutePolicy{
		DefaultBridgeType: 0,
		FallbackMode:      entities.BridgeFallbackModeAutoFallback,
		FallbackOrder:     []uint8{0, 2, 1, 2, 9},
	})

	assert.Equal(t, []uint8{0, 2, 1}, order)
}

func TestBuildBridgeOrderFromPolicy_DefaultFallbackCases(t *testing.T) {
	assert.Equal(t, []uint8{0}, buildBridgeOrderFromPolicy(nil))

	order := buildBridgeOrderFromPolicy(&entities.RoutePolicy{
		DefaultBridgeType: 9,
		FallbackMode:      entities.BridgeFallbackModeStrict,
		FallbackOrder:     []uint8{8, 7},
	})
	assert.Equal(t, []uint8{0}, order)
}
