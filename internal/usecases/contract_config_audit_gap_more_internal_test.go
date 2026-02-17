package usecases

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestRunEVMOnchainChecks_GatewayOrRouterMissingBranches(t *testing.T) {
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "mock://missing-branches"}
	u := newAuditUsecaseWithMockEVM(source.RPCURL, nil)

	t.Run("gateway missing only", func(t *testing.T) {
		checks := u.runEVMOnchainChecks(context.Background(), source, []*entities.SmartContract{
			{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true},
		}, "eip155:42161")
		require.NotEmpty(t, checks)
		require.Equal(t, "GATEWAY_MISSING", checks[0].Code)
	})

	t.Run("router missing only", func(t *testing.T) {
		checks := u.runEVMOnchainChecks(context.Background(), source, []*entities.SmartContract{
			{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true},
		}, "eip155:42161")
		require.NotEmpty(t, checks)
		require.Equal(t, "ROUTER_MISSING", checks[len(checks)-1].Code)
	})
}

func TestRunEVMOnchainChecks_AdapterAndBridgeBranches(t *testing.T) {
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "mock://adapter-branches"}
	contracts := auditContracts(sourceID)

	defaultBridgeABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}]`
	hasAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`
	getAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
	hyperConfiguredABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"}]`
	hyperBytesABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`
	hyperDestABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`
	ccipSelectorABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"}]`
	ccipDestABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`
	adapter := common.HexToAddress("0x1111111111111111111111111111111111111111")

	t.Run("adapter not registered", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(0)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", false))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", common.Address{}))},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, contracts, "eip155:42161")
		require.Equal(t, "ADAPTER_NOT_REGISTERED", checks[1].Code)
	})

	t.Run("hyperbridge fallback not configured due empty bytes", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(0)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", adapter))},
			{err: errors.New("isChainConfigured not present")},
			{out: common.FromHex(encodeMethodOutput(t, hyperBytesABI, "stateMachineIds", []byte{}))},
			{out: common.FromHex(encodeMethodOutput(t, hyperDestABI, "destinationContracts", []byte{}))},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, contracts, "eip155:42161")
		require.Equal(t, "HYPERBRIDGE_CHAIN_NOT_CONFIGURED", checks[len(checks)-1].Code)
	})

	t.Run("hyperbridge fallback read failed", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(0)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", adapter))},
			{err: errors.New("isChainConfigured not present")},
			{err: errors.New("sm read failed")},
			{err: errors.New("dst read failed")},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, contracts, "eip155:42161")
		require.Equal(t, "HYPERBRIDGE_CONFIG_READ_FAILED", checks[len(checks)-1].Code)
	})

	t.Run("hyperbridge configured", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(0)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", adapter))},
			{out: common.FromHex(encodeMethodOutput(t, hyperConfiguredABI, "isChainConfigured", true))},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, contracts, "eip155:42161")
		require.Equal(t, "HYPERBRIDGE_CHAIN_CONFIGURED", checks[len(checks)-1].Code)
	})

	t.Run("ccip selector+destination configured", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(1)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", adapter))},
			{out: common.FromHex(encodeMethodOutput(t, ccipSelectorABI, "chainSelectors", uint64(111)))},
			{out: common.FromHex(encodeMethodOutput(t, ccipDestABI, "destinationAdapters", []byte{0x01}))},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, contracts, "eip155:42161")
		require.Equal(t, "CCIP_SELECTOR_CONFIGURED", checks[len(checks)-2].Code)
		require.Equal(t, "CCIP_DEST_ADAPTER_CONFIGURED", checks[len(checks)-1].Code)
	})
}

func TestContractConfigAudit_HelperGapBranches(t *testing.T) {
	t.Run("extract function names non-map and empty name", func(t *testing.T) {
		names := extractFunctionNames([]interface{}{
			123,
			map[string]interface{}{"type": "function", "name": "   "},
			map[string]interface{}{"type": "event", "name": "Ignored"},
		})
		require.Empty(t, names)
	})

	t.Run("callBoolView pack error", func(t *testing.T) {
		const withArgABI = `[{"inputs":[{"name":"x","type":"uint256"}],"name":"needArg","outputs":[{"type":"bool"}],"stateMutability":"view","type":"function"}]`
		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
			return []byte{}, nil
		})
		_, err := callBoolView(context.Background(), client, common.Address{}.Hex(), withArgABI, "needArg")
		require.Error(t, err)
	})

	t.Run("callBoolView type assertion error", func(t *testing.T) {
		const rawABI = `[{"inputs":[],"name":"v","outputs":[{"type":"uint8"}],"stateMutability":"view","type":"function"}]`
		parsed, err := parseABI(rawABI)
		require.NoError(t, err)
		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
			return parsed.Methods["v"].Outputs.Pack(uint8(1))
		})
		_, err = callBoolView(context.Background(), client, common.Address{}.Hex(), rawABI, "v")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unexpected bool result type")
	})

	t.Run("callAddressView client error", func(t *testing.T) {
		const rawABI = `[{"inputs":[],"name":"v","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}]`
		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
			return nil, errors.New("rpc down")
		})
		_, err := callAddressView(context.Background(), client, common.Address{}.Hex(), rawABI, "v")
		require.Error(t, err)
	})

	t.Run("callBytesView client error", func(t *testing.T) {
		const rawABI = `[{"inputs":[],"name":"v","outputs":[{"type":"bytes"}],"stateMutability":"view","type":"function"}]`
		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
			return nil, errors.New("rpc down")
		})
		_, err := callBytesView(context.Background(), client, common.Address{}.Hex(), rawABI, "v")
		require.Error(t, err)
	})
}
