package usecases

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestResolveRPCURL(t *testing.T) {
	require.Equal(t, "", resolveRPCURL(nil))

	chain := &entities.Chain{RPCURL: "https://main-rpc"}
	require.Equal(t, "https://main-rpc", resolveRPCURL(chain))

	chain = &entities.Chain{RPCs: []entities.ChainRPC{{URL: "https://inactive"}, {URL: "https://active", IsActive: true}}}
	require.Equal(t, "https://active", resolveRPCURL(chain))

	chain = &entities.Chain{RPCs: []entities.ChainRPC{{URL: "https://fallback-1"}, {URL: "https://fallback-2"}}}
	require.Equal(t, "https://fallback-1", resolveRPCURL(chain))
}

func TestParseHexToBytes32(t *testing.T) {
	// 20-byte address input should be left padded to bytes32
	out, err := parseHexToBytes32("0x000000000000000000000000000000000000dEaD")
	require.NoError(t, err)
	require.Len(t, out, 32)

	// Full bytes32 should pass
	out, err = parseHexToBytes32("0x1111111111111111111111111111111111111111111111111111111111111111")
	require.NoError(t, err)
	require.Len(t, out, 32)

	// Invalid length should fail
	_, err = parseHexToBytes32("0x1234")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid bytes32 length")
}

func TestMustParseABI(t *testing.T) {
	require.NotPanics(t, func() {
		_ = mustParseABI(`[{"inputs":[],"name":"ping","outputs":[],"stateMutability":"view","type":"function"}]`)
	})

	require.Panics(t, func() {
		_ = mustParseABI(`[{invalid-json}]`)
	})
}

func TestOnchainAdapterUsecase_RegisterAdapter_InvalidAddress(t *testing.T) {
	u := &OnchainAdapterUsecase{}
	_, err := u.RegisterAdapter(context.Background(), "eip155:8453", "eip155:42161", 0, "not-hex")
	require.Error(t, err)
	require.Equal(t, "invalid input", err.Error())
}

func TestOnchainAdapterUsecase_SendTx_OwnerKeyMissing(t *testing.T) {
	u := &OnchainAdapterUsecase{ownerPrivateKey: ""}
	_, err := u.sendTx(context.Background(), uuid.New(), "0x0000000000000000000000000000000000000001", abi.ABI{}, "set", "arg")
	require.Error(t, err)
	require.Equal(t, "invalid input", err.Error())
}
