package usecases

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func encodeMethodOut(t *testing.T, methodName string, values ...interface{}) []byte {
	t.Helper()
	var (
		out []byte
		err error
	)

	switch methodName {
	case "defaultBridgeTypes":
		out, err = FallbackPayChainGatewayABI.Methods[methodName].Outputs.Pack(values...)
	case "hasAdapter", "getAdapter":
		out, err = FallbackPayChainRouterAdminABI.Methods[methodName].Outputs.Pack(values...)
	case "isChainConfigured", "stateMachineIds", "destinationContracts":
		out, err = FallbackHyperbridgeSenderAdminABI.Methods[methodName].Outputs.Pack(values...)
	case "chainSelectors", "destinationAdapters":
		out, err = FallbackCCIPSenderAdminABI.Methods[methodName].Outputs.Pack(values...)
	case "isRouteConfigured", "dstEids", "peers", "enforcedOptions":
		out, err = FallbackLayerZeroSenderAdminABI.Methods[methodName].Outputs.Pack(values...)
	default:
		t.Fatalf("unsupported method: %s", methodName)
	}
	require.NoError(t, err)
	return out
}

func TestOnchainAdapterUsecase_GetStatus_WithInjectedClient(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, RPCURL: "mock://rpc"}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo := &quoteChainRepoStub{
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  source,
			"eip155:42161": dest,
		},
		byID: map[uuid.UUID]*entities.Chain{
			sourceID: source,
			destID:   dest,
		},
	}
	contractRepo := &scRepoStub{
		getActiveFn: func(_ context.Context, chainID uuid.UUID, typ entities.SmartContractType) (*entities.SmartContract, error) {
			if chainID != sourceID {
				return nil, fmt.Errorf("unexpected chain")
			}
			if typ == entities.ContractTypeGateway {
				return gateway, nil
			}
			if typ == entities.ContractTypeRouter {
				return router, nil
			}
			return nil, fmt.Errorf("unexpected type")
		},
	}

	adapter0 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	adapter1 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	adapter2 := common.HexToAddress("0x3333333333333333333333333333333333333333")
	destCAIP2 := "eip155:42161"

	mockClient := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, data []byte) ([]byte, error) {
		if len(data) < 4 {
			return nil, fmt.Errorf("invalid calldata")
		}
		methodID := "0x" + hex.EncodeToString(data[:4])
		switch methodID {
		case "0x" + hex.EncodeToString(FallbackPayChainGatewayABI.Methods["defaultBridgeTypes"].ID):
			return encodeMethodOut(t, "defaultBridgeTypes", uint8(1)), nil
		case "0x" + hex.EncodeToString(FallbackPayChainRouterAdminABI.Methods["hasAdapter"].ID):
			vals, err := FallbackPayChainRouterAdminABI.Methods["hasAdapter"].Inputs.Unpack(data[4:])
			require.NoError(t, err)
			require.Equal(t, destCAIP2, vals[0].(string))
			return encodeMethodOut(t, "hasAdapter", true), nil
		case "0x" + hex.EncodeToString(FallbackPayChainRouterAdminABI.Methods["getAdapter"].ID):
			vals, err := FallbackPayChainRouterAdminABI.Methods["getAdapter"].Inputs.Unpack(data[4:])
			require.NoError(t, err)
			require.Equal(t, destCAIP2, vals[0].(string))
			bridgeType := vals[1].(uint8)
			if bridgeType == 0 {
				return encodeMethodOut(t, "getAdapter", adapter0), nil
			}
			if bridgeType == 1 {
				return encodeMethodOut(t, "getAdapter", adapter1), nil
			}
			return encodeMethodOut(t, "getAdapter", adapter2), nil
		case "0x" + hex.EncodeToString(FallbackHyperbridgeSenderAdminABI.Methods["isChainConfigured"].ID):
			return encodeMethodOut(t, "isChainConfigured", true), nil
		case "0x" + hex.EncodeToString(FallbackHyperbridgeSenderAdminABI.Methods["stateMachineIds"].ID):
			return encodeMethodOut(t, "stateMachineIds", []byte("EVM-42161")), nil
		case "0x" + hex.EncodeToString(FallbackHyperbridgeSenderAdminABI.Methods["destinationContracts"].ID):
			return encodeMethodOut(t, "destinationContracts", common.LeftPadBytes(common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Bytes(), 32)), nil
		case "0x" + hex.EncodeToString(FallbackCCIPSenderAdminABI.Methods["chainSelectors"].ID):
			return encodeMethodOut(t, "chainSelectors", uint64(4949039107694359620)), nil
		case "0x" + hex.EncodeToString(FallbackCCIPSenderAdminABI.Methods["destinationAdapters"].ID):
			return encodeMethodOut(t, "destinationAdapters", common.LeftPadBytes(common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Bytes(), 32)), nil
		case "0x" + hex.EncodeToString(FallbackLayerZeroSenderAdminABI.Methods["isRouteConfigured"].ID):
			return encodeMethodOut(t, "isRouteConfigured", true), nil
		case "0x" + hex.EncodeToString(FallbackLayerZeroSenderAdminABI.Methods["dstEids"].ID):
			return encodeMethodOut(t, "dstEids", uint32(30110)), nil
		case "0x" + hex.EncodeToString(FallbackLayerZeroSenderAdminABI.Methods["peers"].ID):
			return encodeMethodOut(t, "peers", [32]byte{1}), nil
		case "0x" + hex.EncodeToString(FallbackLayerZeroSenderAdminABI.Methods["enforcedOptions"].ID):
			return encodeMethodOut(t, "enforcedOptions", []byte{0x01, 0x02}), nil
		default:
			return nil, fmt.Errorf("unexpected method id: %s", methodID)
		}
	})

	factory := blockchain.NewClientFactory()
	factory.RegisterEVMClient(source.RPCURL, mockClient)

	u := NewOnchainAdapterUsecase(chainRepo, contractRepo, factory, "")
	status, err := u.GetStatus(context.Background(), "eip155:8453", "eip155:42161")
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, uint8(1), status.DefaultBridgeType)
	require.True(t, status.HasAdapterType0)
	require.True(t, status.HasAdapterType1)
	require.True(t, status.HasAdapterType2)
	require.Equal(t, adapter0.Hex(), status.AdapterType0)
	require.Equal(t, adapter1.Hex(), status.AdapterType1)
	require.Equal(t, adapter2.Hex(), status.AdapterType2)
	require.True(t, status.HyperbridgeConfigured)
	require.True(t, status.LayerZeroConfigured)
	require.Equal(t, uint32(30110), status.LayerZeroDstEID)
	require.NotEmpty(t, status.CCIPDestinationAdapter)
}

func TestOnchainAdapterUsecase_GetStatus_DefaultBridgeError(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "mock://rpc-fail"}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM}
	gateway := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true}
	router := &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true}

	chainRepo := &quoteChainRepoStub{
		byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
		byID:    map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
	}
	contractRepo := &scRepoStub{
		getActiveFn: func(_ context.Context, _ uuid.UUID, typ entities.SmartContractType) (*entities.SmartContract, error) {
			if typ == entities.ContractTypeGateway {
				return gateway, nil
			}
			return router, nil
		},
	}

	mockClient := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return nil, errors.New("boom")
	})
	factory := blockchain.NewClientFactory()
	factory.RegisterEVMClient(source.RPCURL, mockClient)

	u := NewOnchainAdapterUsecase(chainRepo, contractRepo, factory, "")
	_, err := u.GetStatus(context.Background(), "eip155:8453", "eip155:42161")
	require.Error(t, err)
}
