package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

type crosschainAdapterStub struct {
	statusFn            func(context.Context, string, string) (*OnchainAdapterStatus, error)
	registerAdapterFn   func(context.Context, string, string, uint8, string) (string, error)
	setDefaultBridgeFn  func(context.Context, string, string, uint8) (string, error)
	setHyperbridgeCfgFn func(context.Context, string, string, string, string) (string, []string, error)
}

func (s *crosschainAdapterStub) GetStatus(ctx context.Context, sourceChainInput, destChainInput string) (*OnchainAdapterStatus, error) {
	if s.statusFn != nil {
		return s.statusFn(ctx, sourceChainInput, destChainInput)
	}
	return nil, errors.New("status not configured")
}

func (s *crosschainAdapterStub) RegisterAdapter(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8, adapterAddress string) (string, error) {
	if s.registerAdapterFn != nil {
		return s.registerAdapterFn(ctx, sourceChainInput, destChainInput, bridgeType, adapterAddress)
	}
	return "", nil
}

func (s *crosschainAdapterStub) SetDefaultBridgeType(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8) (string, error) {
	if s.setDefaultBridgeFn != nil {
		return s.setDefaultBridgeFn(ctx, sourceChainInput, destChainInput, bridgeType)
	}
	return "", nil
}

func (s *crosschainAdapterStub) SetHyperbridgeConfig(ctx context.Context, sourceChainInput, destChainInput string, stateMachineIDHex, destinationContractHex string) (string, []string, error) {
	if s.setHyperbridgeCfgFn != nil {
		return s.setHyperbridgeCfgFn(ctx, sourceChainInput, destChainInput, stateMachineIDHex, destinationContractHex)
	}
	return "", nil, nil
}

func TestCrosschainConfigUsecase_Preflight_WithAdapterStub(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()

	chainRepo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{
			sourceID: {ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true},
			destID:   {ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true},
		},
		byChain: map[string]*entities.Chain{},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  {ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true},
			"eip155:42161": {ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true},
		},
		allChain: []*entities.Chain{
			{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true},
			{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true},
		},
	}
	tokenRepo := &ccTokenRepoStub{byChain: map[uuid.UUID][]*entities.Token{}}
	contractRepo := &ccContractRepoStub{active: map[string]*entities.SmartContract{}}
	adapter := &crosschainAdapterStub{
		statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
			return &OnchainAdapterStatus{
				DefaultBridgeType:      1,
				HasAdapterType0:        true,
				AdapterType0:           "0x1111111111111111111111111111111111111111",
				HasAdapterType1:        true,
				AdapterType1:           "0x2222222222222222222222222222222222222222",
				HasAdapterType2:        false,
				AdapterType2:           "",
				HasAdapterDefault:      true,
				AdapterDefaultType:     "0x2222222222222222222222222222222222222222",
				HyperbridgeConfigured:  true,
				CCIPChainSelector:      0,
				CCIPDestinationAdapter: "0x",
				LayerZeroConfigured:    false,
			}, nil
		},
	}

	u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, adapter)
	res, err := u.Preflight(context.Background(), "eip155:8453", "eip155:42161")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint8(1), res.DefaultBridgeType)
	require.False(t, res.PolicyExecutable)
	require.Len(t, res.Bridges, 3)
	require.Equal(t, uint8(1), res.Bridges[1].BridgeType)
	require.Equal(t, "CCIP_NOT_CONFIGURED", res.Bridges[1].ErrorCode)
}

func TestCrosschainConfigUsecase_AutoFix_WithAdapterStub(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	srcChain := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true}
	dstChain := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}

	chainRepo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{sourceID: srcChain, destID: dstChain},
		byChain: map[string]*entities.Chain{
			"8453":  srcChain,
			"42161": dstChain,
		},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  srcChain,
			"eip155:42161": dstChain,
		},
		allChain: []*entities.Chain{srcChain, dstChain},
	}
	tokenRepo := &ccTokenRepoStub{byChain: map[uuid.UUID][]*entities.Token{}}
	contractRepo := &ccContractRepoStub{
		active: map[string]*entities.SmartContract{
			contractKey(sourceID, entities.ContractTypeAdapterHyperbridge): &entities.SmartContract{
				ID:              uuid.New(),
				ChainUUID:       sourceID,
				Type:            entities.ContractTypeAdapterHyperbridge,
				ContractAddress: "0x1111111111111111111111111111111111111111",
				IsActive:        true,
			},
			contractKey(destID, entities.ContractTypeAdapterHyperbridge): &entities.SmartContract{
				ID:              uuid.New(),
				ChainUUID:       destID,
				Type:            entities.ContractTypeAdapterHyperbridge,
				ContractAddress: "0x2222222222222222222222222222222222222222",
				IsActive:        true,
			},
		},
	}

	adapter := &crosschainAdapterStub{
		statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
			return &OnchainAdapterStatus{
				DefaultBridgeType: 2,
				AdapterType0:      "",
				AdapterType1:      "",
				AdapterType2:      "",
			}, nil
		},
		registerAdapterFn: func(context.Context, string, string, uint8, string) (string, error) {
			return "0xreg", nil
		},
		setDefaultBridgeFn: func(context.Context, string, string, uint8) (string, error) {
			return "0xdefault", nil
		},
		setHyperbridgeCfgFn: func(context.Context, string, string, string, string) (string, []string, error) {
			return "", []string{"0xsm", "0xdst"}, nil
		},
	}

	u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, adapter)
	bridgeType := uint8(0)
	res, err := u.AutoFix(context.Background(), &AutoFixRequest{
		SourceChainID: "eip155:8453",
		DestChainID:   "eip155:42161",
		BridgeType:    &bridgeType,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint8(0), res.BridgeType)
	require.Len(t, res.Steps, 3)
	require.Equal(t, "SUCCESS", res.Steps[0].Status)
	require.Equal(t, "SUCCESS", res.Steps[1].Status)
	require.Equal(t, "SUCCESS", res.Steps[2].Status)
}

func TestCrosschainConfigUsecase_AutoFix_BranchMatrix(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	srcChain := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true}
	dstChain := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}

	chainRepo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{sourceID: srcChain, destID: dstChain},
		byChain: map[string]*entities.Chain{
			"8453":  srcChain,
			"42161": dstChain,
		},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  srcChain,
			"eip155:42161": dstChain,
		},
		allChain: []*entities.Chain{srcChain, dstChain},
	}
	tokenRepo := &ccTokenRepoStub{byChain: map[uuid.UUID][]*entities.Token{}}

	t.Run("register adapter fail", func(t *testing.T) {
		contractRepo := &ccContractRepoStub{
			active: map[string]*entities.SmartContract{
				contractKey(sourceID, entities.ContractTypeAdapterHyperbridge): {
					ID:              uuid.New(),
					ChainUUID:       sourceID,
					Type:            entities.ContractTypeAdapterHyperbridge,
					ContractAddress: "0x1111111111111111111111111111111111111111",
					IsActive:        true,
				},
			},
		}
		adapter := &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{DefaultBridgeType: 0}, nil
			},
			registerAdapterFn: func(context.Context, string, string, uint8, string) (string, error) {
				return "", errors.New("register failed")
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, adapter)
		res, err := u.AutoFix(context.Background(), &AutoFixRequest{SourceChainID: "eip155:8453", DestChainID: "eip155:42161"})
		require.NoError(t, err)
		require.Equal(t, "FAILED", res.Steps[0].Status)
		require.Equal(t, "registerAdapter", res.Steps[0].Step)
	})

	t.Run("set default fail", func(t *testing.T) {
		contractRepo := &ccContractRepoStub{
			active: map[string]*entities.SmartContract{
				contractKey(destID, entities.ContractTypeAdapterHyperbridge): {
					ID:              uuid.New(),
					ChainUUID:       destID,
					Type:            entities.ContractTypeAdapterHyperbridge,
					ContractAddress: "0x2222222222222222222222222222222222222222",
					IsActive:        true,
				},
			},
		}
		adapter := &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType: 2,
					AdapterType0:      "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				}, nil
			},
			setDefaultBridgeFn: func(context.Context, string, string, uint8) (string, error) {
				return "", errors.New("set default failed")
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, adapter)
		bridgeType := uint8(0)
		res, err := u.AutoFix(context.Background(), &AutoFixRequest{
			SourceChainID: "eip155:8453",
			DestChainID:   "eip155:42161",
			BridgeType:    &bridgeType,
		})
		require.NoError(t, err)
		require.Len(t, res.Steps, 2)
		require.Equal(t, "SKIPPED", res.Steps[0].Status)
		require.Equal(t, "FAILED", res.Steps[1].Status)
		require.Equal(t, "setDefaultBridge", res.Steps[1].Step)
	})

	t.Run("bridge type two skip route config", func(t *testing.T) {
		adapter := &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType: 2,
					AdapterType2:      "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				}, nil
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, &ccContractRepoStub{active: map[string]*entities.SmartContract{}}, nil, adapter)
		res, err := u.AutoFix(context.Background(), &AutoFixRequest{
			SourceChainID: "eip155:8453",
			DestChainID:   "eip155:42161",
		})
		require.NoError(t, err)
		require.Len(t, res.Steps, 3)
		require.Equal(t, "SKIPPED", res.Steps[0].Status)
		require.Equal(t, "SKIPPED", res.Steps[1].Status)
		require.Equal(t, "SKIPPED", res.Steps[2].Status)
		require.Equal(t, "setLayerZeroConfig", res.Steps[2].Step)
	})
}
