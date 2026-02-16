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
	_, err := u.callHasAdapter(ctx, client, addr, dest, 0)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callGetAdapter(ctx, client, addr, dest, 0)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callHyperbridgeConfigured(ctx, client, addr, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callHyperbridgeBytes(ctx, client, addr, "stateMachineIds", dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callCCIPSelector(ctx, client, addr, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callCCIPDestinationAdapter(ctx, client, addr, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callLayerZeroConfigured(ctx, client, addr, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callLayerZeroDstEid(ctx, client, addr, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callLayerZeroPeer(ctx, client, addr, dest)
	require.Error(t, err)

	client = newTestEVMClient(t, []string{"0x"})
	_, err = u.callLayerZeroOptions(ctx, client, addr, dest)
	require.Error(t, err)
}
