package usecases

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func mustEncodeOutputFromABI(t *testing.T, parsedABI abi.ABI, method string, values ...interface{}) string {
	t.Helper()
	packed, err := parsedABI.Methods[method].Outputs.Pack(values...)
	require.NoError(t, err)
	return "0x" + common.Bytes2Hex(packed)
}

func TestOnchainAdapterUsecase_CallHelpers_Success(t *testing.T) {
	u := &OnchainAdapterUsecase{}
	dest := "eip155:42161"
	routerAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	adapterAddr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	peerHash := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	hyperBytes := []byte{0x65, 0x69, 0x70}
	ccipDest := common.LeftPadBytes(common.HexToAddress("0x3333333333333333333333333333333333333333").Bytes(), 32)
	lzOptions := []byte{0x01, 0x02}

	client := newTestEVMClient(t, []string{
		mustEncodeOutputFromABI(t, FallbackPayChainGatewayABI, "defaultBridgeTypes", uint8(1)),
		mustEncodeOutputFromABI(t, FallbackPayChainRouterAdminABI, "hasAdapter", true),
		mustEncodeOutputFromABI(t, FallbackPayChainRouterAdminABI, "getAdapter", adapterAddr),
		mustEncodeOutputFromABI(t, FallbackHyperbridgeSenderAdminABI, "isChainConfigured", true),
		mustEncodeOutputFromABI(t, FallbackHyperbridgeSenderAdminABI, "stateMachineIds", hyperBytes),
		mustEncodeOutputFromABI(t, FallbackCCIPSenderAdminABI, "chainSelectors", uint64(12345)),
		mustEncodeOutputFromABI(t, FallbackCCIPSenderAdminABI, "destinationAdapters", ccipDest),
		mustEncodeOutputFromABI(t, FallbackLayerZeroSenderAdminABI, "isRouteConfigured", true),
		mustEncodeOutputFromABI(t, FallbackLayerZeroSenderAdminABI, "dstEids", uint32(40161)),
		mustEncodeOutputFromABI(t, FallbackLayerZeroSenderAdminABI, "peers", [32]byte(peerHash)),
		mustEncodeOutputFromABI(t, FallbackLayerZeroSenderAdminABI, "enforcedOptions", lzOptions),
	})

	vBridge, err := u.callDefaultBridgeType(context.Background(), client, routerAddr.Hex(), FallbackPayChainGatewayABI, dest)
	require.NoError(t, err)
	require.Equal(t, uint8(1), vBridge)

	vHas, err := u.callHasAdapter(context.Background(), client, routerAddr.Hex(), FallbackPayChainRouterAdminABI, dest, 1)
	require.NoError(t, err)
	require.True(t, vHas)

	vAdapter, err := u.callGetAdapter(context.Background(), client, routerAddr.Hex(), FallbackPayChainRouterAdminABI, dest, 1)
	require.NoError(t, err)
	require.Equal(t, adapterAddr.Hex(), vAdapter)

	vHB, err := u.callHyperbridgeConfigured(context.Background(), client, adapterAddr.Hex(), FallbackHyperbridgeSenderAdminABI, dest)
	require.NoError(t, err)
	require.True(t, vHB)

	vHBBytes, err := u.callHyperbridgeBytes(context.Background(), client, adapterAddr.Hex(), FallbackHyperbridgeSenderAdminABI, "stateMachineIds", dest)
	require.NoError(t, err)
	require.Equal(t, hyperBytes, vHBBytes)

	vSelector, err := u.callCCIPSelector(context.Background(), client, adapterAddr.Hex(), FallbackCCIPSenderAdminABI, dest)
	require.NoError(t, err)
	require.Equal(t, uint64(12345), vSelector)

	vCCIPDest, err := u.callCCIPDestinationAdapter(context.Background(), client, adapterAddr.Hex(), FallbackCCIPSenderAdminABI, dest)
	require.NoError(t, err)
	require.Equal(t, ccipDest, vCCIPDest)

	vLZCfg, err := u.callLayerZeroConfigured(context.Background(), client, adapterAddr.Hex(), FallbackLayerZeroSenderAdminABI, dest)
	require.NoError(t, err)
	require.True(t, vLZCfg)

	vDstEid, err := u.callLayerZeroDstEid(context.Background(), client, adapterAddr.Hex(), FallbackLayerZeroSenderAdminABI, dest)
	require.NoError(t, err)
	require.Equal(t, uint32(40161), vDstEid)

	vPeer, err := u.callLayerZeroPeer(context.Background(), client, adapterAddr.Hex(), FallbackLayerZeroSenderAdminABI, dest)
	require.NoError(t, err)
	require.Equal(t, peerHash, vPeer)

	vOpts, err := u.callLayerZeroOptions(context.Background(), client, adapterAddr.Hex(), FallbackLayerZeroSenderAdminABI, dest)
	require.NoError(t, err)
	require.Equal(t, lzOptions, vOpts)
}

func TestOnchainAdapterUsecase_CallHelpers_DecodeError(t *testing.T) {
	u := &OnchainAdapterUsecase{}
	client := newTestEVMClient(t, []string{"0x"})
	_, err := u.callDefaultBridgeType(context.Background(), client, common.Address{}.Hex(), FallbackPayChainGatewayABI, "eip155:42161")
	require.Error(t, err)
}
