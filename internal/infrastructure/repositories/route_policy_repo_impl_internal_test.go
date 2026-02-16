package repositories

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoutePolicyRepo_FallbackOrderHelpers(t *testing.T) {
	t.Run("marshal empty defaults to bridge type 0", func(t *testing.T) {
		raw, err := marshalFallbackOrder(nil)
		require.NoError(t, err)
		require.Equal(t, "\"AA==\"", raw)
	})

	t.Run("marshal non-empty order", func(t *testing.T) {
		raw, err := marshalFallbackOrder([]uint8{2, 1, 0})
		require.NoError(t, err)
		require.Equal(t, "\"AgEA\"", raw)
	})

	t.Run("parse empty or invalid defaults to 0", func(t *testing.T) {
		require.Equal(t, []uint8{0}, parseFallbackOrder(""))
		require.Equal(t, []uint8{0}, parseFallbackOrder("not-json"))
		require.Equal(t, []uint8{0}, parseFallbackOrder("[]"))
	})

	t.Run("parse valid order", func(t *testing.T) {
		require.Equal(t, []uint8{2, 1, 0}, parseFallbackOrder("\"AgEA\""))
	})
}
