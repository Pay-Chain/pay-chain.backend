package usecases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func newMatrixChains() (*entities.Chain, *entities.Chain, *ccChainRepoStub) {
	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}
	repo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{
			sourceID: source,
			destID:   dest,
		},
		byChain: map[string]*entities.Chain{
			"8453":  source,
			"42161": dest,
		},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  source,
			"eip155:42161": dest,
		},
		allChain: []*entities.Chain{source, dest},
	}
	return source, dest, repo
}

func hasIssueCode(items []ContractConfigCheckItem, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func TestCrosschainConfigUsecase_RecheckRoute_IssueMatrix(t *testing.T) {
	source, dest, chainRepo := newMatrixChains()
	tokenRepo := &ccTokenRepoStub{}
	contractRepo := &ccContractRepoStub{}

	t.Run("default hyperbridge but not configured", func(t *testing.T) {
		adapter := &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType:      0,
					HasAdapterDefault:      true,
					AdapterDefaultType:     "0x1111111111111111111111111111111111111111",
					HyperbridgeConfigured:  false,
					CCIPChainSelector:      0,
					CCIPDestinationAdapter: "0x",
					LayerZeroConfigured:    false,
				}, nil
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, adapter)
		res, err := u.RecheckRoute(context.Background(), source.GetCAIP2ID(), dest.GetCAIP2ID())
		require.NoError(t, err)
		require.Equal(t, "ERROR", res.OverallStatus)
		require.True(t, hasIssueCode(res.Issues, "HYPERBRIDGE_NOT_CONFIGURED"))
		require.True(t, hasIssueCode(res.Issues, "FEE_QUOTE_FAILED"))
	})

	t.Run("default ccip but not configured", func(t *testing.T) {
		adapter := &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType:      1,
					HasAdapterDefault:      true,
					AdapterDefaultType:     "0x1111111111111111111111111111111111111111",
					HyperbridgeConfigured:  true,
					CCIPChainSelector:      0,
					CCIPDestinationAdapter: "0x",
					LayerZeroConfigured:    false,
				}, nil
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, adapter)
		res, err := u.RecheckRoute(context.Background(), source.GetCAIP2ID(), dest.GetCAIP2ID())
		require.NoError(t, err)
		require.Equal(t, "ERROR", res.OverallStatus)
		require.True(t, hasIssueCode(res.Issues, "CCIP_NOT_CONFIGURED"))
		require.True(t, hasIssueCode(res.Issues, "FEE_QUOTE_FAILED"))
	})

	t.Run("default layerzero but not configured", func(t *testing.T) {
		adapter := &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType:      2,
					HasAdapterDefault:      true,
					AdapterDefaultType:     "0x1111111111111111111111111111111111111111",
					HyperbridgeConfigured:  true,
					CCIPChainSelector:      4949039107694359620,
					CCIPDestinationAdapter: "0x11",
					LayerZeroConfigured:    false,
				}, nil
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, adapter)
		res, err := u.RecheckRoute(context.Background(), source.GetCAIP2ID(), dest.GetCAIP2ID())
		require.NoError(t, err)
		require.Equal(t, "ERROR", res.OverallStatus)
		require.True(t, hasIssueCode(res.Issues, "LAYERZERO_NOT_CONFIGURED"))
		require.True(t, hasIssueCode(res.Issues, "FEE_QUOTE_FAILED"))
	})

	t.Run("adapter not registered", func(t *testing.T) {
		adapter := &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType:      0,
					HasAdapterDefault:      false,
					AdapterDefaultType:     "",
					HyperbridgeConfigured:  false,
					CCIPChainSelector:      0,
					CCIPDestinationAdapter: "0x",
					LayerZeroConfigured:    false,
				}, nil
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, adapter)
		res, err := u.RecheckRoute(context.Background(), source.GetCAIP2ID(), dest.GetCAIP2ID())
		require.NoError(t, err)
		require.Equal(t, "ERROR", res.OverallStatus)
		require.True(t, hasIssueCode(res.Issues, "ADAPTER_NOT_REGISTERED"))
	})
}

func TestCrosschainConfigUsecase_Preflight_FeeQuoteFailedBranch(t *testing.T) {
	source, dest, chainRepo := newMatrixChains()
	tokenRepo := &ccTokenRepoStub{}
	contractRepo := &ccContractRepoStub{}

	adapter := &crosschainAdapterStub{
		statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
			return &OnchainAdapterStatus{
				DefaultBridgeType:      0,
				HasAdapterDefault:      true,
				AdapterDefaultType:     "0x1111111111111111111111111111111111111111",
				HasAdapterType0:        true,
				AdapterType0:           "0x1111111111111111111111111111111111111111",
				HasAdapterType1:        false,
				AdapterType1:           "",
				HasAdapterType2:        false,
				AdapterType2:           "",
				HyperbridgeConfigured:  true,
				CCIPChainSelector:      0,
				CCIPDestinationAdapter: "0x",
				LayerZeroConfigured:    false,
			}, nil
		},
	}

	u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, nil, adapter)
	res, err := u.Preflight(context.Background(), source.GetCAIP2ID(), dest.GetCAIP2ID())
	require.NoError(t, err)
	require.False(t, res.PolicyExecutable)
	require.Len(t, res.Bridges, 3)

	var row0 CrosschainBridgePreflight
	for _, row := range res.Bridges {
		if row.BridgeType == 0 {
			row0 = row
			break
		}
	}
	require.False(t, row0.Ready)
	require.Equal(t, "FEE_QUOTE_FAILED", row0.ErrorCode)
}

func TestCrosschainConfigUsecase_Preflight_DefaultLayerZeroPolicyExecutable(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	rpcServer := newCrosschainFeeRPCServer(t)
	defer rpcServer.Close()

	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true, RPCURL: rpcServer.URL}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}
	chainRepo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{
			sourceID: source,
			destID:   dest,
		},
		byChain: map[string]*entities.Chain{
			"8453":  source,
			"42161": dest,
		},
		byCAIP2: map[string]*entities.Chain{
			source.GetCAIP2ID(): source,
			dest.GetCAIP2ID():   dest,
		},
	}
	tokenRepo := &ccTokenRepoStub{
		byChain: map[uuid.UUID][]*entities.Token{
			sourceID: {&entities.Token{ContractAddress: "0x1111111111111111111111111111111111111111"}},
			destID:   {&entities.Token{ContractAddress: "0x2222222222222222222222222222222222222222"}},
		},
	}
	contractRepo := &ccContractRepoStub{
		active: map[string]*entities.SmartContract{
			contractKey(sourceID, entities.ContractTypeRouter): {
				ChainUUID:       sourceID,
				Type:            entities.ContractTypeRouter,
				ContractAddress: "0x9999999999999999999999999999999999999999",
				IsActive:        true,
			},
		},
	}

	adapter := &crosschainAdapterStub{
		statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
			return &OnchainAdapterStatus{
				DefaultBridgeType:   2,
				HasAdapterDefault:   true,
				AdapterDefaultType:  "0x3333333333333333333333333333333333333333",
				HasAdapterType2:     true,
				AdapterType2:        "0x3333333333333333333333333333333333333333",
				LayerZeroConfigured: true,
			}, nil
		},
	}

	u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, blockchain.NewClientFactory(), adapter)
	res, err := u.Preflight(context.Background(), source.GetCAIP2ID(), dest.GetCAIP2ID())
	require.NoError(t, err)
	require.True(t, res.PolicyExecutable)
	require.Equal(t, uint8(2), res.DefaultBridgeType)

	var row2 *CrosschainBridgePreflight
	for i := range res.Bridges {
		if res.Bridges[i].BridgeType == 2 {
			row2 = &res.Bridges[i]
			break
		}
	}
	require.NotNil(t, row2)
	require.True(t, row2.Ready)
}

func TestCrosschainConfigUsecase_Preflight_DefaultCCIPPolicyExecutable(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	rpcServer := newCrosschainFeeRPCServer(t)
	defer rpcServer.Close()

	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true, RPCURL: rpcServer.URL}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}
	chainRepo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{
			sourceID: source,
			destID:   dest,
		},
		byChain: map[string]*entities.Chain{
			"8453":  source,
			"42161": dest,
		},
		byCAIP2: map[string]*entities.Chain{
			source.GetCAIP2ID(): source,
			dest.GetCAIP2ID():   dest,
		},
	}
	tokenRepo := &ccTokenRepoStub{
		byChain: map[uuid.UUID][]*entities.Token{
			sourceID: {&entities.Token{ContractAddress: "0x1111111111111111111111111111111111111111"}},
			destID:   {&entities.Token{ContractAddress: "0x2222222222222222222222222222222222222222"}},
		},
	}
	contractRepo := &ccContractRepoStub{
		active: map[string]*entities.SmartContract{
			contractKey(sourceID, entities.ContractTypeRouter): {
				ChainUUID:       sourceID,
				Type:            entities.ContractTypeRouter,
				ContractAddress: "0x9999999999999999999999999999999999999999",
				IsActive:        true,
			},
		},
	}

	adapter := &crosschainAdapterStub{
		statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
			return &OnchainAdapterStatus{
				DefaultBridgeType:      1,
				HasAdapterDefault:      true,
				AdapterDefaultType:     "0x1111111111111111111111111111111111111111",
				HasAdapterType1:        true,
				AdapterType1:           "0x1111111111111111111111111111111111111111",
				CCIPChainSelector:      4949039107694359620,
				CCIPDestinationAdapter: "0x2222",
			}, nil
		},
	}

	u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, blockchain.NewClientFactory(), adapter)
	res, err := u.Preflight(context.Background(), source.GetCAIP2ID(), dest.GetCAIP2ID())
	require.NoError(t, err)
	require.True(t, res.PolicyExecutable)
	require.Equal(t, uint8(1), res.DefaultBridgeType)

	var row1 *CrosschainBridgePreflight
	for i := range res.Bridges {
		if res.Bridges[i].BridgeType == 1 {
			row1 = &res.Bridges[i]
			break
		}
	}
	require.NotNil(t, row1)
	require.True(t, row1.Ready)
}
