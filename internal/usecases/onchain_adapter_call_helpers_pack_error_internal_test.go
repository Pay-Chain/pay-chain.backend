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

	origGateway := payChainGatewayAdminABI
	origRouter := payChainRouterAdminABI
	origHyper := hyperbridgeSenderAdminABI
	origCCIP := ccipSenderAdminABI
	origLZ := layerZeroSenderAdminABI
	t.Cleanup(func() {
		payChainGatewayAdminABI = origGateway
		payChainRouterAdminABI = origRouter
		hyperbridgeSenderAdminABI = origHyper
		ccipSenderAdminABI = origCCIP
		layerZeroSenderAdminABI = origLZ
	})

	// Empty ABIs force Pack(...) to fail deterministically.
	payChainGatewayAdminABI = abi.ABI{}
	payChainRouterAdminABI = abi.ABI{}
	hyperbridgeSenderAdminABI = abi.ABI{}
	ccipSenderAdminABI = abi.ABI{}
	layerZeroSenderAdminABI = abi.ABI{}

	_, err := u.callDefaultBridgeType(ctx, client, addr, dest)
	require.Error(t, err)

	_, err = u.callHasAdapter(ctx, client, addr, dest, 0)
	require.Error(t, err)

	_, err = u.callGetAdapter(ctx, client, addr, dest, 0)
	require.Error(t, err)

	_, err = u.callHyperbridgeConfigured(ctx, client, addr, dest)
	require.Error(t, err)

	_, err = u.callHyperbridgeBytes(ctx, client, addr, "stateMachineIds", dest)
	require.Error(t, err)

	_, err = u.callCCIPSelector(ctx, client, addr, dest)
	require.Error(t, err)

	_, err = u.callCCIPDestinationAdapter(ctx, client, addr, dest)
	require.Error(t, err)

	_, err = u.callLayerZeroConfigured(ctx, client, addr, dest)
	require.Error(t, err)

	_, err = u.callLayerZeroDstEid(ctx, client, addr, dest)
	require.Error(t, err)

	_, err = u.callLayerZeroPeer(ctx, client, addr, dest)
	require.Error(t, err)

	_, err = u.callLayerZeroOptions(ctx, client, addr, dest)
	require.Error(t, err)
}
