package usecases

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeSafeQuoteResult(t *testing.T) {
	boolType, _ := abi.NewType("bool", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	stringType, _ := abi.NewType("string", "", nil)
	args := abi.Arguments{
		{Type: boolType},
		{Type: uint256Type},
		{Type: stringType},
	}

	t.Run("Success case", func(t *testing.T) {
		encoded, err := args.Pack(true, big.NewInt(12345), "")
		require.NoError(t, err)

		ok, fee, reason, err := decodeSafeQuoteResult(encoded)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, big.NewInt(12345), fee)
		assert.Equal(t, "", reason)
	})

	t.Run("Failure case with reason", func(t *testing.T) {
		encoded, err := args.Pack(false, big.NewInt(0), "liquidity error")
		require.NoError(t, err)

		ok, fee, reason, err := decodeSafeQuoteResult(encoded)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "0", fee.String())
		assert.Equal(t, "liquidity error", reason)
	})

	t.Run("Invalid data", func(t *testing.T) {
		_, _, _, err := decodeSafeQuoteResult([]byte{0x00, 0x01})
		require.Error(t, err)
	})
}

// Helper to generate mock return data for tests
func encodeSafeQuoteResult(t *testing.T, ok bool, fee *big.Int, reason string) string {
	boolType, _ := abi.NewType("bool", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	stringType, _ := abi.NewType("string", "", nil)
	args := abi.Arguments{
		{Type: boolType},
		{Type: uint256Type},
		{Type: stringType},
	}
	packed, err := args.Pack(ok, fee, reason)
	require.NoError(t, err)
	return "0x" + hex.EncodeToString(packed)
}
