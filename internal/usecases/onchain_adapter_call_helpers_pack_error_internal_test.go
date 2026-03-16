package usecases

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"payment-kita.backend/internal/infrastructure/blockchain"
)

func TestOnchainAdapterUsecase_CallHelpers_PackErrorByEmptyABI(t *testing.T) {
	u := &OnchainAdapterUsecase{}
	ctx := context.Background()
	client := blockchain.NewEVMClientWithCallView(nil, func(context.Context, string, []byte) ([]byte, error) {
		return []byte{}, nil
	})
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111").Hex()
	dest := "eip155:42161"

	origGateway := FallbackPaymentKitaGatewayABI
	origRouter := FallbackPaymentKitaRouterAdminABI
	origHyper := FallbackHyperbridgeSenderAdminABI
	origCCIP := FallbackCCIPSenderAdminABI
	origLZ := FallbackStargateSenderAdminABI
	t.Cleanup(func() {
		FallbackPaymentKitaGatewayABI = origGateway
		FallbackPaymentKitaRouterAdminABI = origRouter
		FallbackHyperbridgeSenderAdminABI = origHyper
		FallbackCCIPSenderAdminABI = origCCIP
		FallbackStargateSenderAdminABI = origLZ
	})

	// Empty ABIs force Pack(...) to fail deterministically.
	FallbackPaymentKitaGatewayABI = abi.ABI{}
	FallbackPaymentKitaRouterAdminABI = abi.ABI{}
	FallbackHyperbridgeSenderAdminABI = abi.ABI{}
	FallbackCCIPSenderAdminABI = abi.ABI{}
	FallbackStargateSenderAdminABI = abi.ABI{}

	_, err := u.callDefaultBridgeType(ctx, client, addr, FallbackPaymentKitaGatewayABI, dest)
	require.Error(t, err)

	_, err = u.callHasAdapter(ctx, client, addr, FallbackPaymentKitaRouterAdminABI, dest, 0)
	require.Error(t, err)

	_, err = u.callGetAdapter(ctx, client, addr, FallbackPaymentKitaRouterAdminABI, dest, 0)
	require.Error(t, err)

	_, err = u.callHyperbridgeConfigured(ctx, client, addr, FallbackHyperbridgeSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callHyperbridgeBytes(ctx, client, addr, FallbackHyperbridgeSenderAdminABI, "stateMachineIds", dest)
	require.Error(t, err)

	_, err = u.callCCIPSelector(ctx, client, addr, FallbackCCIPSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callCCIPDestinationAdapter(ctx, client, addr, FallbackCCIPSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callStargateConfigured(ctx, client, addr, FallbackStargateSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callStargateDstEid(ctx, client, addr, FallbackStargateSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callStargatePeer(ctx, client, addr, FallbackStargateSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callStargateOptions(ctx, client, addr, FallbackStargateSenderAdminABI, dest)
	require.Error(t, err)
}
