package usecases

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestOnchainAdapterUsecase_CallHelpers_DecodeErrorMatrix(t *testing.T) {
	u := &OnchainAdapterUsecase{}
	ctx := context.Background()
	dest := "eip155:42161"
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111").Hex()

	client := newTestEVMClient(t, []string{"0x"})
	_, err := u.callHasAdapter(ctx, client, addr, FallbackPaymentKitaRouterAdminABI, dest, 0)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callGetAdapter(ctx, client, addr, FallbackPaymentKitaRouterAdminABI, dest, 0)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callHyperbridgeConfigured(ctx, client, addr, FallbackHyperbridgeSenderAdminABI, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callHyperbridgeBytes(ctx, client, addr, FallbackHyperbridgeSenderAdminABI, "stateMachineIds", dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callCCIPSelector(ctx, client, addr, FallbackCCIPSenderAdminABI, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callCCIPDestinationAdapter(ctx, client, addr, FallbackCCIPSenderAdminABI, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callStargateConfigured(ctx, client, addr, FallbackStargateSenderAdminABI, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callStargateDstEid(ctx, client, addr, FallbackStargateSenderAdminABI, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callStargatePeer(ctx, client, addr, FallbackStargateSenderAdminABI, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callStargateOptions(ctx, client, addr, FallbackStargateSenderAdminABI, dest)
	require.Error(t, err)
}
