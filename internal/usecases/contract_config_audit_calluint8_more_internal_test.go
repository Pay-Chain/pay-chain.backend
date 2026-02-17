package usecases

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestCallUint8View_ErrorAndSuccessBranches(t *testing.T) {
	const needArgABI = `[{"inputs":[{"name":"x","type":"uint256"}],"name":"needArg","outputs":[{"type":"uint8"}],"stateMutability":"view","type":"function"}]`
	clientCallErr := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return nil, errors.New("rpc failed")
	})

	// pack error
	_, err := callUint8View(context.Background(), clientCallErr, common.Address{}.Hex(), needArgABI, "needArg")
	require.Error(t, err)

	// call error
	_, err = callUint8View(context.Background(), clientCallErr, common.Address{}.Hex(), needArgABI, "needArg", big.NewInt(1))
	require.Error(t, err)
	require.Contains(t, err.Error(), "rpc failed")

	// decode error
	const simpleU8ABI = `[{"inputs":[],"name":"u8","outputs":[{"type":"uint8"}],"stateMutability":"view","type":"function"}]`
	clientDecodeErr := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return []byte{}, nil
	})
	_, err = callUint8View(context.Background(), clientDecodeErr, common.Address{}.Hex(), simpleU8ABI, "u8")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode uint8 result")

	// unexpected type
	const uint256ABI = `[{"inputs":[],"name":"u","outputs":[{"type":"uint256"}],"stateMutability":"view","type":"function"}]`
	parsed, err := parseABI(uint256ABI)
	require.NoError(t, err)
	clientTypeErr := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return parsed.Methods["u"].Outputs.Pack(big.NewInt(10))
	})
	_, err = callUint8View(context.Background(), clientTypeErr, common.Address{}.Hex(), uint256ABI, "u")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected uint8 result type")

	// success
	parsedU8, err := parseABI(simpleU8ABI)
	require.NoError(t, err)
	clientSuccess := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return parsedU8.Methods["u8"].Outputs.Pack(uint8(7))
	})
	got, err := callUint8View(context.Background(), clientSuccess, common.Address{}.Hex(), simpleU8ABI, "u8")
	require.NoError(t, err)
	require.Equal(t, uint8(7), got)
}

var _ = abi.Arguments{}
