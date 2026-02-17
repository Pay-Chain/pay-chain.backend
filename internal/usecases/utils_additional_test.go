package usecases

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertToSmallestUnit_AdditionalBranches(t *testing.T) {
	t.Run("leading dot normalizes whole part", func(t *testing.T) {
		got, err := convertToSmallestUnit(".25", 6)
		require.NoError(t, err)
		require.Equal(t, "250000", got)
	})

	t.Run("multiple dots invalid format", func(t *testing.T) {
		_, err := convertToSmallestUnit("1.2.3", 6)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid amount format")
	})

	t.Run("trim spaces and plus with zero result", func(t *testing.T) {
		got, err := convertToSmallestUnit("  +0.000  ", 3)
		require.NoError(t, err)
		require.Equal(t, "0", got)
	})
}

func TestAddressToBytes32_HexAndAsciiBranches(t *testing.T) {
	t.Run("hex shorter than 32 bytes is left padded", func(t *testing.T) {
		out := addressToBytes32("0x1234")
		require.Equal(t, "0000000000000000000000000000000000000000000000000000000000001234", hex.EncodeToString(out[:]))
	})

	t.Run("invalid hex prefix falls back to ascii bytes", func(t *testing.T) {
		out := addressToBytes32("0xinvalid")
		require.NotEqual(t, [32]byte{}, out)
	})

	t.Run("non-base58 long ascii is right-trimmed", func(t *testing.T) {
		out := addressToBytes32("________________________________________")
		require.NotEqual(t, [32]byte{}, out)
	})
}

func TestBase58Decode_EmptyStringBranch(t *testing.T) {
	require.Nil(t, base58Decode(""))
}
