package usecases

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestOnchainAdapterUsecase_CallHelpers_PackErrorByEmptyABI(t *testing.T) {
	u := &OnchainAdapterUsecase{}
	ctx := context.Background()
	client := blockchain.NewEVMClientWithCallView(nil, func(context.Context, string, []byte) ([]byte, error) {
		return []byte{}, nil
	})
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111").Hex()
	dest := "eip155:42161"

	origGateway := FallbackPayChainGatewayABI
	origRouter := FallbackPayChainRouterAdminABI
	origHyper := FallbackHyperbridgeSenderAdminABI
	origCCIP := FallbackCCIPSenderAdminABI
	origLZ := FallbackLayerZeroSenderAdminABI
	t.Cleanup(func() {
		FallbackPayChainGatewayABI = origGateway
		FallbackPayChainRouterAdminABI = origRouter
		FallbackHyperbridgeSenderAdminABI = origHyper
		FallbackCCIPSenderAdminABI = origCCIP
		FallbackLayerZeroSenderAdminABI = origLZ
	})

	// Empty ABIs force Pack(...) to fail deterministically.
	FallbackPayChainGatewayABI = abi.ABI{}
	FallbackPayChainRouterAdminABI = abi.ABI{}
	FallbackHyperbridgeSenderAdminABI = abi.ABI{}
	FallbackCCIPSenderAdminABI = abi.ABI{}
	FallbackLayerZeroSenderAdminABI = abi.ABI{}

	_, err := u.callDefaultBridgeType(ctx, client, addr, FallbackPayChainGatewayABI, dest)
	require.Error(t, err)

	_, err = u.callHasAdapter(ctx, client, addr, FallbackPayChainRouterAdminABI, dest, 0)
	require.Error(t, err)

	_, err = u.callGetAdapter(ctx, client, addr, FallbackPayChainRouterAdminABI, dest, 0)
	require.Error(t, err)

	_, err = u.callHyperbridgeConfigured(ctx, client, addr, FallbackHyperbridgeSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callHyperbridgeBytes(ctx, client, addr, FallbackHyperbridgeSenderAdminABI, "stateMachineIds", dest)
	require.Error(t, err)

	_, err = u.callCCIPSelector(ctx, client, addr, FallbackCCIPSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callCCIPDestinationAdapter(ctx, client, addr, FallbackCCIPSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callLayerZeroConfigured(ctx, client, addr, FallbackLayerZeroSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callLayerZeroDstEid(ctx, client, addr, FallbackLayerZeroSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callLayerZeroPeer(ctx, client, addr, FallbackLayerZeroSenderAdminABI, dest)
	require.Error(t, err)

	_, err = u.callLayerZeroOptions(ctx, client, addr, FallbackLayerZeroSenderAdminABI, dest)
	require.Error(t, err)
}
