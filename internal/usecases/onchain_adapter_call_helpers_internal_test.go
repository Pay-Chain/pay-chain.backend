package usecases

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
	"payment-kita.backend/internal/infrastructure/blockchain"
)

func TestOnchainAdapterUsecase_CallHelpers_PackError(t *testing.T) {
	origGatewayABI := FallbackPaymentKitaGatewayABI
	origRouterABI := FallbackPaymentKitaRouterAdminABI
	origHyperABI := FallbackHyperbridgeSenderAdminABI
	origCCIPABI := FallbackCCIPSenderAdminABI
	origLZABI := FallbackStargateSenderAdminABI
	t.Cleanup(func() {
		FallbackPaymentKitaGatewayABI = origGatewayABI
		FallbackPaymentKitaRouterAdminABI = origRouterABI
		FallbackHyperbridgeSenderAdminABI = origHyperABI
		FallbackCCIPSenderAdminABI = origCCIPABI
		FallbackStargateSenderAdminABI = origLZABI
	})

	// Force Pack(...) to fail before any RPC call.
	FallbackPaymentKitaGatewayABI = abi.ABI{}
	FallbackPaymentKitaRouterAdminABI = abi.ABI{}
	FallbackHyperbridgeSenderAdminABI = abi.ABI{}
	FallbackCCIPSenderAdminABI = abi.ABI{}
	FallbackStargateSenderAdminABI = abi.ABI{}

	u := &OnchainAdapterUsecase{}
	ctx := context.Background()
	var client *blockchain.EVMClient

	_, err := u.callDefaultBridgeType(ctx, client, "0x0", FallbackPaymentKitaGatewayABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callHasAdapter(ctx, client, "0x0", FallbackPaymentKitaRouterAdminABI, "eip155:42161", 0)
	require.Error(t, err)
	_, err = u.callGetAdapter(ctx, client, "0x0", FallbackPaymentKitaRouterAdminABI, "eip155:42161", 0)
	require.Error(t, err)
	_, err = u.callHyperbridgeConfigured(ctx, client, "0x0", FallbackHyperbridgeSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callHyperbridgeBytes(ctx, client, "0x0", FallbackHyperbridgeSenderAdminABI, "stateMachineIds", "eip155:42161")
	require.Error(t, err)
	_, err = u.callCCIPSelector(ctx, client, "0x0", FallbackCCIPSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callCCIPDestinationAdapter(ctx, client, "0x0", FallbackCCIPSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callStargateConfigured(ctx, client, "0x0", FallbackStargateSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callStargateDstEid(ctx, client, "0x0", FallbackStargateSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callStargatePeer(ctx, client, "0x0", FallbackStargateSenderAdminABI, "eip155:42161")
	require.Error(t, err)
	_, err = u.callStargateOptions(ctx, client, "0x0", FallbackStargateSenderAdminABI, "eip155:42161")
	require.Error(t, err)
}
