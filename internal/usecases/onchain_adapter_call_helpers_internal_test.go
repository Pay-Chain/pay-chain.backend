package usecases

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestOnchainAdapterUsecase_CallHelpers_PackError(t *testing.T) {
	origGatewayABI := FallbackPayChainGatewayABI
	origRouterABI := FallbackPayChainRouterAdminABI
	origHyperABI := FallbackHyperbridgeSenderAdminABI
	origCCIPABI := FallbackCCIPSenderAdminABI
	origLZABI := FallbackLayerZeroSenderAdminABI
	t.Cleanup(func() {
		FallbackPayChainGatewayABI = origGatewayABI
		FallbackPayChainRouterAdminABI = origRouterABI
		FallbackHyperbridgeSenderAdminABI = origHyperABI
		FallbackCCIPSenderAdminABI = origCCIPABI
		FallbackLayerZeroSenderAdminABI = origLZABI
	})

	// Force Pack(...) to fail before any RPC call.
	FallbackPayChainGatewayABI = abi.ABI{}
	FallbackPayChainRouterAdminABI = abi.ABI{}
	FallbackHyperbridgeSenderAdminABI = abi.ABI{}
	FallbackCCIPSenderAdminABI = abi.ABI{}
	FallbackLayerZeroSenderAdminABI = abi.ABI{}

	u := &OnchainAdapterUsecase{}
	ctx := context.Background()
	var client *blockchain.EVMClient

	_, err := u.callDefaultBridgeType(ctx, client, "0x0", FallbackPayChainGatewayABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callHasAdapter(ctx, client, "0x0", FallbackPayChainRouterAdminABI, "eip155:42161", 0)
	require.Error(t, err)
	_, err = u.callGetAdapter(ctx, client, "0x0", FallbackPayChainRouterAdminABI, "eip155:42161", 0)
	require.Error(t, err)
	_, err = u.callHyperbridgeConfigured(ctx, client, "0x0", FallbackHyperbridgeSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callHyperbridgeBytes(ctx, client, "0x0", FallbackHyperbridgeSenderAdminABI, "stateMachineIds", "eip155:42161")
	require.Error(t, err)
	_, err = u.callCCIPSelector(ctx, client, "0x0", FallbackCCIPSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callCCIPDestinationAdapter(ctx, client, "0x0", FallbackCCIPSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callLayerZeroConfigured(ctx, client, "0x0", FallbackLayerZeroSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callLayerZeroDstEid(ctx, client, "0x0", FallbackLayerZeroSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callLayerZeroPeer(ctx, client, "0x0", FallbackLayerZeroSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callLayerZeroOptions(ctx, client, "0x0", FallbackLayerZeroSenderAdminABI, "eip155:42161")
	require.Error(t, err)
}
