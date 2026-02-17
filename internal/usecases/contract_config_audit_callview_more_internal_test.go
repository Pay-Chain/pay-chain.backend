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

func TestCallViewHelpers_PackAndClientErrorBranches(t *testing.T) {
	const withArgABI = `[{"inputs":[{"name":"x","type":"uint256"}],"name":"needArg","outputs":[{"type":"uint64"}],"stateMutability":"view","type":"function"}]`
	client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return nil, errors.New("rpc failed")
	})

	// pack error: no args provided for method that requires one arg.
	_, err := callUint64View(context.Background(), client, common.Address{}.Hex(), withArgABI, "needArg")
	require.Error(t, err)

	// client call error.
	_, err = callUint64View(context.Background(), client, common.Address{}.Hex(), withArgABI, "needArg", big.NewInt(1))
	require.Error(t, err)
	require.Contains(t, err.Error(), "rpc failed")
}

func TestCallViewHelpers_TypeAssertionBranches(t *testing.T) {
	const uintAbi = `[{"inputs":[],"name":"v","outputs":[{"type":"uint256"}],"stateMutability":"view","type":"function"}]`
	const bytes32Abi = `[{"inputs":[],"name":"v","outputs":[{"type":"bytes32"}],"stateMutability":"view","type":"function"}]`
	const addressAbi = `[{"inputs":[],"name":"v","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}]`

	uintParsed, err := parseABI(uintAbi)
	require.NoError(t, err)
	bytes32Parsed, err := parseABI(bytes32Abi)
	require.NoError(t, err)
	addressParsed, err := parseABI(addressAbi)
	require.NoError(t, err)

	clientForUint := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return uintParsed.Methods["v"].Outputs.Pack(big.NewInt(42))
	})
	_, err = callUint64View(context.Background(), clientForUint, common.Address{}.Hex(), uintAbi, "v")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected uint64 result type")

	clientForAddr := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		var v [32]byte
		copy(v[:], []byte("x"))
		return bytes32Parsed.Methods["v"].Outputs.Pack(v)
	})
	_, err = callAddressView(context.Background(), clientForAddr, common.Address{}.Hex(), bytes32Abi, "v")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected address result type")

	clientForBytes := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return addressParsed.Methods["v"].Outputs.Pack(common.HexToAddress("0x0000000000000000000000000000000000000001"))
	})
	_, err = callBytesView(context.Background(), clientForBytes, common.Address{}.Hex(), addressAbi, "v")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected bytes result type")
}

func TestCallUint64View_DecodeErrorBranch(t *testing.T) {
	const rawABI = `[{"inputs":[],"name":"v","outputs":[{"type":"uint64"}],"stateMutability":"view","type":"function"}]`
	client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return []byte{}, nil
	})

	_, err := callUint64View(context.Background(), client, common.Address{}.Hex(), rawABI, "v")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode uint64 result")
}

func TestCallViewHelpers_AddressAndBytesSuccess(t *testing.T) {
	const addressABI = `[{"inputs":[],"name":"addr","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}]`
	const bytesABI = `[{"inputs":[],"name":"payload","outputs":[{"type":"bytes"}],"stateMutability":"view","type":"function"}]`

	addressParsed, err := parseABI(addressABI)
	require.NoError(t, err)
	bytesParsed, err := parseABI(bytesABI)
	require.NoError(t, err)

	expectedAddress := common.HexToAddress("0x0000000000000000000000000000000000000009")
	clientAddress := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return addressParsed.Methods["addr"].Outputs.Pack(expectedAddress)
	})
	gotAddress, err := callAddressView(context.Background(), clientAddress, common.Address{}.Hex(), addressABI, "addr")
	require.NoError(t, err)
	require.Equal(t, expectedAddress, gotAddress)

	expectedBytes := []byte{0x01, 0x02, 0x03}
	clientBytes := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return bytesParsed.Methods["payload"].Outputs.Pack(expectedBytes)
	})
	gotBytes, err := callBytesView(context.Background(), clientBytes, common.Address{}.Hex(), bytesABI, "payload")
	require.NoError(t, err)
	require.Equal(t, expectedBytes, gotBytes)
}

func TestCallViewHelpers_AddressAndBytes_ErrorBranches(t *testing.T) {
	const badABI = `not-json`
	client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return []byte{}, nil
	})

	_, err := callAddressView(context.Background(), client, common.Address{}.Hex(), badABI, "x")
	require.Error(t, err)

	_, err = callBytesView(context.Background(), client, common.Address{}.Hex(), badABI, "x")
	require.Error(t, err)

	const needArgAddressABI = `[{"inputs":[{"name":"x","type":"uint256"}],"name":"addr","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}]`
	const needArgBytesABI = `[{"inputs":[{"name":"x","type":"uint256"}],"name":"payload","outputs":[{"type":"bytes"}],"stateMutability":"view","type":"function"}]`

	// pack error branches
	_, err = callAddressView(context.Background(), client, common.Address{}.Hex(), needArgAddressABI, "addr")
	require.Error(t, err)
	_, err = callBytesView(context.Background(), client, common.Address{}.Hex(), needArgBytesABI, "payload")
	require.Error(t, err)

	// decode error branches
	_, err = callAddressView(context.Background(), client, common.Address{}.Hex(), `[{"inputs":[],"name":"addr","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}]`, "addr")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode address result")
	_, err = callBytesView(context.Background(), client, common.Address{}.Hex(), `[{"inputs":[],"name":"payload","outputs":[{"type":"bytes"}],"stateMutability":"view","type":"function"}]`, "payload")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode bytes result")
}

var _ = abi.Arguments{}
