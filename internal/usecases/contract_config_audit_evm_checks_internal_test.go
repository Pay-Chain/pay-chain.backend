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

type evmCallStep struct {
	out []byte
	err error
}

func newAuditUsecaseWithMockEVM(rpcURL string, steps []evmCallStep) *ContractConfigAuditUsecase {
	clientFactory := blockchain.NewClientFactory()
	idx := 0
	clientFactory.RegisterEVMClient(rpcURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
		if idx >= len(steps) {
			return nil, errors.New("unexpected call")
		}
		step := steps[idx]
		idx++
		return step.out, step.err
	}))
	return &ContractConfigAuditUsecase{clientFactory: clientFactory}
}

func auditContracts(sourceID uuid.UUID) []*entities.SmartContract {
	return []*entities.SmartContract{
		{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true},
		{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true},
	}
}

func TestRunEVMOnchainChecks_CoreAndReadFailureBranches(t *testing.T) {
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, Type: entities.ChainTypeEVM}

	u := &ContractConfigAuditUsecase{clientFactory: blockchain.NewClientFactory()}
	checks := u.runEVMOnchainChecks(context.Background(), source, nil, "eip155:42161")
	require.Equal(t, "RPC_MISSING", checks[0].Code)

	source.RPCURL = "mock://rpc-read-fail"
	u = newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
		{err: errors.New("default bridge fail")},
	})
	checks = u.runEVMOnchainChecks(context.Background(), source, auditContracts(sourceID), "eip155:42161")
	require.Equal(t, "DEFAULT_BRIDGE_READ_FAILED", checks[len(checks)-1].Code)

	source.RPCURL = "mock://rpc-has-fail"
	defaultBridgeABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}]`
	u = newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
		{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(0)))},
		{err: errors.New("has adapter fail")},
	})
	checks = u.runEVMOnchainChecks(context.Background(), source, auditContracts(sourceID), "eip155:42161")
	require.Equal(t, "HAS_ADAPTER_READ_FAILED", checks[len(checks)-1].Code)
}

func TestRunEVMOnchainChecks_HyperbridgeBranches(t *testing.T) {
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "mock://hyper"}
	defaultBridgeABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}]`
	hasAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`
	getAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
	hyperConfiguredABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"}]`
	hyperBytesABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`
	hyperDestABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`
	adapter := common.HexToAddress("0x1111111111111111111111111111111111111111")

	t.Run("adapter zero", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(0)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", common.Address{}))},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, auditContracts(sourceID), "eip155:42161")
		require.Equal(t, "HYPERBRIDGE_ADAPTER_ADDRESS_INVALID", checks[len(checks)-1].Code)
	})

	t.Run("configured false", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(0)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", adapter))},
			{out: common.FromHex(encodeMethodOutput(t, hyperConfiguredABI, "isChainConfigured", false))},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, auditContracts(sourceID), "eip155:42161")
		require.Equal(t, "HYPERBRIDGE_CHAIN_NOT_CONFIGURED", checks[len(checks)-1].Code)
	})

	t.Run("fallback configured from bytes", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(0)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", adapter))},
			{err: errors.New("isChainConfigured missing")},
			{out: common.FromHex(encodeMethodOutput(t, hyperBytesABI, "stateMachineIds", []byte{0x01}))},
			{out: common.FromHex(encodeMethodOutput(t, hyperDestABI, "destinationContracts", []byte{0x02}))},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, auditContracts(sourceID), "eip155:42161")
		require.Equal(t, "HYPERBRIDGE_CHAIN_CONFIGURED_FALLBACK", checks[len(checks)-1].Code)
	})
}

func TestRunEVMOnchainChecks_CCIPBranches(t *testing.T) {
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "mock://ccip"}
	defaultBridgeABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}]`
	hasAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`
	getAdapterABI := `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
	ccipSelectorABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"}]`
	ccipDestABI := `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`
	adapter := common.HexToAddress("0x2222222222222222222222222222222222222222")

	t.Run("adapter zero", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(1)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", common.Address{}))},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, auditContracts(sourceID), "eip155:42161")
		require.Equal(t, "CCIP_ADAPTER_ADDRESS_INVALID", checks[len(checks)-1].Code)
	})

	t.Run("selector and destination missing", func(t *testing.T) {
		u := newAuditUsecaseWithMockEVM(source.RPCURL, []evmCallStep{
			{out: common.FromHex(encodeMethodOutput(t, defaultBridgeABI, "defaultBridgeTypes", uint8(1)))},
			{out: common.FromHex(encodeMethodOutput(t, hasAdapterABI, "hasAdapter", true))},
			{out: common.FromHex(encodeMethodOutput(t, getAdapterABI, "getAdapter", adapter))},
			{out: common.FromHex(encodeMethodOutput(t, ccipSelectorABI, "chainSelectors", uint64(0)))},
			{out: common.FromHex(encodeMethodOutput(t, ccipDestABI, "destinationAdapters", []byte{}))},
		})
		checks := u.runEVMOnchainChecks(context.Background(), source, auditContracts(sourceID), "eip155:42161")
		require.Equal(t, "CCIP_SELECTOR_MISSING", checks[len(checks)-2].Code)
		require.Equal(t, "CCIP_DEST_ADAPTER_MISSING", checks[len(checks)-1].Code)
	})
}
