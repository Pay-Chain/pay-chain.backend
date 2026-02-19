package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestOnchainAdapterUsecase_CallHelpers_CallViewErrorBranches(t *testing.T) {
	u := &OnchainAdapterUsecase{}
	ctx := context.Background()
	dest := "eip155:42161"
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111").Hex()
	client := blockchain.NewEVMClientWithCallView(nil, func(context.Context, string, []byte) ([]byte, error) {
		return nil, errors.New("call view failed")
	})

	_, err := u.callDefaultBridgeType(ctx, client, addr, FallbackPayChainGatewayABI, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callHasAdapter(ctx, client, addr, FallbackPayChainRouterAdminABI, dest, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callGetAdapter(ctx, client, addr, FallbackPayChainRouterAdminABI, dest, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callHyperbridgeConfigured(ctx, client, addr, FallbackHyperbridgeSenderAdminABI, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callHyperbridgeBytes(ctx, client, addr, FallbackHyperbridgeSenderAdminABI, "stateMachineIds", dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callCCIPSelector(ctx, client, addr, FallbackCCIPSenderAdminABI, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callCCIPDestinationAdapter(ctx, client, addr, FallbackCCIPSenderAdminABI, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callLayerZeroConfigured(ctx, client, addr, FallbackLayerZeroSenderAdminABI, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callLayerZeroDstEid(ctx, client, addr, FallbackLayerZeroSenderAdminABI, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callLayerZeroPeer(ctx, client, addr, FallbackLayerZeroSenderAdminABI, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")

	_, err = u.callLayerZeroOptions(ctx, client, addr, FallbackLayerZeroSenderAdminABI, dest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "call view failed")
}
