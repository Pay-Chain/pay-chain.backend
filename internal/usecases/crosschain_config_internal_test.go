package usecases

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBridgeName(t *testing.T) {
	require.Equal(t, "HYPERBRIDGE", bridgeName(0))
	require.Equal(t, "CCIP", bridgeName(1))
	require.Equal(t, "LAYERZERO", bridgeName(2))
	require.Equal(t, "UNKNOWN", bridgeName(99))
}

func TestAddressToPaddedBytesHex(t *testing.T) {
	hexValue, err := addressToPaddedBytesHex("0x000000000000000000000000000000000000dEaD")
	require.NoError(t, err)
	require.Len(t, hexValue, 66)
	require.Equal(t, "0x000000000000000000000000000000000000000000000000000000000000dead", hexValue)

	_, err = addressToPaddedBytesHex("not-an-address")
	require.Error(t, err)
}

func TestDeriveEvmStateMachineHex(t *testing.T) {
	require.Equal(t, "0x45564d2d38343533", deriveEvmStateMachineHex("eip155:8453"))
	require.Equal(t, "0x45564d2d3432313631", deriveEvmStateMachineHex(" EIP155:42161 "))
	require.Equal(t, "", deriveEvmStateMachineHex("solana:mainnet"))
	require.Equal(t, "", deriveEvmStateMachineHex(""))
}
