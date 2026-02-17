package usecases

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func mustEncodeOutMismatch(t *testing.T, parsed abi.ABI, method string, values ...interface{}) string {
	t.Helper()
	out, err := parsed.Methods[method].Outputs.Pack(values...)
	require.NoError(t, err)
	return "0x" + common.Bytes2Hex(out)
}

func TestOnchainAdapterUsecase_CallHelpers_InvalidReturnTypeBranches(t *testing.T) {
	u := &OnchainAdapterUsecase{}
	ctx := context.Background()
	dest := "eip155:42161"
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111").Hex()

	origGatewayABI := payChainGatewayAdminABI
	origRouterABI := payChainRouterAdminABI
	origHyperABI := hyperbridgeSenderAdminABI
	origCCIPABI := ccipSenderAdminABI
	origLZABI := layerZeroSenderAdminABI
	t.Cleanup(func() {
		payChainGatewayAdminABI = origGatewayABI
		payChainRouterAdminABI = origRouterABI
		hyperbridgeSenderAdminABI = origHyperABI
		ccipSenderAdminABI = origCCIPABI
		layerZeroSenderAdminABI = origLZABI
	})

	gatewayMismatch := mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`)
	payChainGatewayAdminABI = gatewayMismatch
	client := newTestEVMClient(t, []string{
		mustEncodeOutMismatch(t, gatewayMismatch, "defaultBridgeTypes", true),
	})
	_, err := u.callDefaultBridgeType(ctx, client, addr, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid defaultBridgeTypes return type")

	routerMismatch := mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
	]`)
	payChainRouterAdminABI = routerMismatch
	client = newTestEVMClient(t, []string{
		mustEncodeOutMismatch(t, routerMismatch, "hasAdapter", big.NewInt(1)),
		mustEncodeOutMismatch(t, routerMismatch, "getAdapter", big.NewInt(1)),
	})
	_, err = u.callHasAdapter(ctx, client, addr, dest, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid hasAdapter return type")
	_, err = u.callGetAdapter(ctx, client, addr, dest, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid getAdapter return type")

	hyperMismatch := mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
	]`)
	hyperbridgeSenderAdminABI = hyperMismatch
	client = newTestEVMClient(t, []string{
		mustEncodeOutMismatch(t, hyperMismatch, "isChainConfigured", big.NewInt(1)),
		mustEncodeOutMismatch(t, hyperMismatch, "stateMachineIds", big.NewInt(1)),
	})
	_, err = u.callHyperbridgeConfigured(ctx, client, addr, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid isChainConfigured return type")
	_, err = u.callHyperbridgeBytes(ctx, client, addr, "stateMachineIds", dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid stateMachineIds return type")

	ccipMismatch := mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
	]`)
	ccipSenderAdminABI = ccipMismatch
	client = newTestEVMClient(t, []string{
		mustEncodeOutMismatch(t, ccipMismatch, "chainSelectors", true),
		mustEncodeOutMismatch(t, ccipMismatch, "destinationAdapters", big.NewInt(1)),
	})
	_, err = u.callCCIPSelector(ctx, client, addr, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid chainSelectors return type")
	_, err = u.callCCIPDestinationAdapter(ctx, client, addr, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid destinationAdapters return type")

	lzMismatch := mustParseABI(`[
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"isRouteConfigured","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"dstEids","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"peers","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"enforcedOptions","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
	]`)
	layerZeroSenderAdminABI = lzMismatch
	client = newTestEVMClient(t, []string{
		mustEncodeOutMismatch(t, lzMismatch, "isRouteConfigured", big.NewInt(1)),
		mustEncodeOutMismatch(t, lzMismatch, "dstEids", true),
		mustEncodeOutMismatch(t, lzMismatch, "peers", big.NewInt(1)),
		mustEncodeOutMismatch(t, lzMismatch, "enforcedOptions", big.NewInt(1)),
	})
	_, err = u.callLayerZeroConfigured(ctx, client, addr, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid isRouteConfigured return type")
	_, err = u.callLayerZeroDstEid(ctx, client, addr, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid dstEids return type")
	_, err = u.callLayerZeroPeer(ctx, client, addr, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid peers return type")
	_, err = u.callLayerZeroOptions(ctx, client, addr, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid enforcedOptions return type")
}
