package usecases

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestOnchainAdapterUsecase_CallHelpers_PackError(t *testing.T) {
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

	// Force Pack(...) to fail before any RPC call.
	payChainGatewayAdminABI = abi.ABI{}
	payChainRouterAdminABI = abi.ABI{}
	hyperbridgeSenderAdminABI = abi.ABI{}
	ccipSenderAdminABI = abi.ABI{}
	layerZeroSenderAdminABI = abi.ABI{}

	u := &OnchainAdapterUsecase{}
	ctx := context.Background()
	var client *blockchain.EVMClient

	_, err := u.callDefaultBridgeType(ctx, client, "0x0", "eip155:42161")
	require.Error(t, err)
	_, err = u.callHasAdapter(ctx, client, "0x0", "eip155:42161", 0)
	require.Error(t, err)
	_, err = u.callGetAdapter(ctx, client, "0x0", "eip155:42161", 0)
	require.Error(t, err)
	_, err = u.callHyperbridgeConfigured(ctx, client, "0x0", "eip155:42161")
	require.Error(t, err)
	_, err = u.callHyperbridgeBytes(ctx, client, "0x0", "stateMachineIds", "eip155:42161")
	require.Error(t, err)
	_, err = u.callCCIPSelector(ctx, client, "0x0", "eip155:42161")
	require.Error(t, err)
	_, err = u.callCCIPDestinationAdapter(ctx, client, "0x0", "eip155:42161")
	require.Error(t, err)
	_, err = u.callLayerZeroConfigured(ctx, client, "0x0", "eip155:42161")
	require.Error(t, err)
	_, err = u.callLayerZeroDstEid(ctx, client, "0x0", "eip155:42161")
	require.Error(t, err)
	_, err = u.callLayerZeroPeer(ctx, client, "0x0", "eip155:42161")
	require.Error(t, err)
	_, err = u.callLayerZeroOptions(ctx, client, "0x0", "eip155:42161")
	require.Error(t, err)
}
