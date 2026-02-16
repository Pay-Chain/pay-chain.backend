package usecases

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestContractConfigAudit_CallViewHelpers_WithInjectedClient(t *testing.T) {
	const rawABI = `[
		{"inputs":[],"name":"getBool","outputs":[{"type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"getU8","outputs":[{"type":"uint8"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"getU64","outputs":[{"type":"uint64"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"getAddr","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"getBytes","outputs":[{"type":"bytes"}],"stateMutability":"view","type":"function"}
	]`

	parsed, err := parseABI(rawABI)
	require.NoError(t, err)

	expectedAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	expectedBytes := []byte{0xde, 0xad, 0xbe, 0xef}

	client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, data []byte) ([]byte, error) {
		method := ""
		for name, m := range parsed.Methods {
			if len(data) >= 4 && string(m.ID) == string(data[:4]) {
				method = name
				break
			}
		}
		switch method {
		case "getBool":
			out, _ := parsed.Methods["getBool"].Outputs.Pack(true)
			return out, nil
		case "getU8":
			out, _ := parsed.Methods["getU8"].Outputs.Pack(uint8(7))
			return out, nil
		case "getU64":
			out, _ := parsed.Methods["getU64"].Outputs.Pack(uint64(42))
			return out, nil
		case "getAddr":
			out, _ := parsed.Methods["getAddr"].Outputs.Pack(expectedAddr)
			return out, nil
		case "getBytes":
			out, _ := parsed.Methods["getBytes"].Outputs.Pack(expectedBytes)
			return out, nil
		default:
			return []byte{}, nil
		}
	})

	vBool, err := callBoolView(context.Background(), client, expectedAddr.Hex(), rawABI, "getBool")
	require.NoError(t, err)
	require.True(t, vBool)

	vU8, err := callUint8View(context.Background(), client, expectedAddr.Hex(), rawABI, "getU8")
	require.NoError(t, err)
	require.Equal(t, uint8(7), vU8)

	vU64, err := callUint64View(context.Background(), client, expectedAddr.Hex(), rawABI, "getU64")
	require.NoError(t, err)
	require.Equal(t, uint64(42), vU64)

	vAddr, err := callAddressView(context.Background(), client, expectedAddr.Hex(), rawABI, "getAddr")
	require.NoError(t, err)
	require.Equal(t, expectedAddr, vAddr)

	vBytes, err := callBytesView(context.Background(), client, expectedAddr.Hex(), rawABI, "getBytes")
	require.NoError(t, err)
	require.Equal(t, expectedBytes, vBytes)
}

func TestContractConfigAudit_CallViewHelpers_ErrorBranches_WithInjectedClient(t *testing.T) {
	const rawABI = `[{"inputs":[],"name":"getBool","outputs":[{"type":"bool"}],"stateMutability":"view","type":"function"}]`
	emptyOutClient := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return []byte{}, nil
	})

	_, err := callBoolView(context.Background(), emptyOutClient, common.Address{}.Hex(), "not-json", "getBool")
	require.Error(t, err)

	_, err = callBoolView(context.Background(), emptyOutClient, common.Address{}.Hex(), rawABI, "getBool")
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "decode"))
}
