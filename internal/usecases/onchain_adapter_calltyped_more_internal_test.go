package usecases

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCallTypedView_MoreSuccessTypes(t *testing.T) {
	t.Run("uint8 success", func(t *testing.T) {
		parsed := parseABIForOnchainGapTest(t, `[
			{"inputs":[],"name":"bridgeType","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
		]`)
		client := newTestEVMClient(t, []string{
			encodeABIReturnForOnchainGapTest(t, parsed, "bridgeType", uint8(2)),
		})
		v, err := callTypedView[uint8](context.Background(), client, common.Address{}.Hex(), parsed, "bridgeType")
		require.NoError(t, err)
		require.Equal(t, uint8(2), v)
	})

	t.Run("address success", func(t *testing.T) {
		parsed := parseABIForOnchainGapTest(t, `[
			{"inputs":[],"name":"adapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
		]`)
		addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
		client := newTestEVMClient(t, []string{
			encodeABIReturnForOnchainGapTest(t, parsed, "adapter", addr),
		})
		v, err := callTypedView[common.Address](context.Background(), client, common.Address{}.Hex(), parsed, "adapter")
		require.NoError(t, err)
		require.Equal(t, addr, v)
	})

	t.Run("bytes success", func(t *testing.T) {
		parsed := parseABIForOnchainGapTest(t, `[
			{"inputs":[],"name":"payload","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
		]`)
		client := newTestEVMClient(t, []string{
			encodeABIReturnForOnchainGapTest(t, parsed, "payload", []byte{0xaa, 0xbb, 0xcc}),
		})
		v, err := callTypedView[[]byte](context.Background(), client, common.Address{}.Hex(), parsed, "payload")
		require.NoError(t, err)
		require.Equal(t, []byte{0xaa, 0xbb, 0xcc}, v)
	})
}

func TestCallTypedView_TypeMismatchError(t *testing.T) {
	parsed := parseABIForOnchainGapTest(t, `[
		{"inputs":[],"name":"flag","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`)
	client := newTestEVMClient(t, []string{
		encodeABIReturnForOnchainGapTest(t, parsed, "flag", true),
	})

	_, err := callTypedView[uint8](context.Background(), client, common.Address{}.Hex(), parsed, "flag")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid flag return type")
}
