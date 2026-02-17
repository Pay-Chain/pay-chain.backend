package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestCrosschainConfigUsecase_AutoFix_MoreBranches(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	srcChain := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true}
	dstChain := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}

	baseChainRepo := &ccChainRepoStub{
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

	t.Run("status error", func(t *testing.T) {
		u := NewCrosschainConfigUsecase(baseChainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return nil, errors.New("status failed")
			},
		})
		_, err := u.AutoFix(context.Background(), &AutoFixRequest{
			SourceChainID: "eip155:8453",
			DestChainID:   "eip155:42161",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "status failed")
	})

	t.Run("missing adapter and invalid source chain input", func(t *testing.T) {
		u := NewCrosschainConfigUsecase(baseChainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType: 0,
					AdapterType0:      "",
				}, nil
			},
		})
		_, err := u.AutoFix(context.Background(), &AutoFixRequest{
			SourceChainID: "invalid-source",
			DestChainID:   "eip155:42161",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid input")
	})

	t.Run("ccip bridge with existing adapter only two skipped steps", func(t *testing.T) {
		u := NewCrosschainConfigUsecase(baseChainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType: 1,
					AdapterType1:      "0x1111111111111111111111111111111111111111",
				}, nil
			},
		})
		res, err := u.AutoFix(context.Background(), &AutoFixRequest{
			SourceChainID: "eip155:8453",
			DestChainID:   "eip155:42161",
		})
		require.NoError(t, err)
		require.Len(t, res.Steps, 2)
		require.Equal(t, "registerAdapter", res.Steps[0].Step)
		require.Equal(t, "SKIPPED", res.Steps[0].Status)
		require.Equal(t, "setDefaultBridge", res.Steps[1].Step)
		require.Equal(t, "SKIPPED", res.Steps[1].Status)
	})

		t.Run("hyperbridge destination contract derive failure", func(t *testing.T) {
		contractRepo := &ccContractRepoStub{
			active: map[string]*entities.SmartContract{},
		}
		bridgeType := uint8(0)
		u := NewCrosschainConfigUsecase(baseChainRepo, &ccTokenRepoStub{}, contractRepo, nil, &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType: 2,
					AdapterType0:      "0x1111111111111111111111111111111111111111",
				}, nil
			},
			setDefaultBridgeFn: func(context.Context, string, string, uint8) (string, error) {
				return "0xset", nil
			},
		})
		res, err := u.AutoFix(context.Background(), &AutoFixRequest{
			SourceChainID: "eip155:8453",
			DestChainID:   "eip155:42161",
			BridgeType:    &bridgeType,
		})
		require.NoError(t, err)
		require.Len(t, res.Steps, 3)
		require.Equal(t, "FAILED", res.Steps[2].Status)
			require.Equal(t, "setHyperbridgeDestination", res.Steps[2].Step)
		})

		t.Run("ccip missing adapter contract in db", func(t *testing.T) {
			bridgeType := uint8(1)
			u := NewCrosschainConfigUsecase(baseChainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{active: map[string]*entities.SmartContract{}}, nil, &crosschainAdapterStub{
				statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
					return &OnchainAdapterStatus{
						DefaultBridgeType: 0,
						AdapterType1:      "",
					}, nil
				},
			})
			res, err := u.AutoFix(context.Background(), &AutoFixRequest{
				SourceChainID: "eip155:8453",
				DestChainID:   "eip155:42161",
				BridgeType:    &bridgeType,
			})
			require.NoError(t, err)
			require.Len(t, res.Steps, 1)
			require.Equal(t, "registerAdapter", res.Steps[0].Step)
			require.Equal(t, "FAILED", res.Steps[0].Status)
		})

		t.Run("layerzero missing adapter contract in db", func(t *testing.T) {
			bridgeType := uint8(2)
			u := NewCrosschainConfigUsecase(baseChainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{active: map[string]*entities.SmartContract{}}, nil, &crosschainAdapterStub{
				statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
					return &OnchainAdapterStatus{
						DefaultBridgeType: 0,
						AdapterType2:      "",
					}, nil
				},
			})
			res, err := u.AutoFix(context.Background(), &AutoFixRequest{
				SourceChainID: "eip155:8453",
				DestChainID:   "eip155:42161",
				BridgeType:    &bridgeType,
			})
			require.NoError(t, err)
			require.Len(t, res.Steps, 1)
			require.Equal(t, "registerAdapter", res.Steps[0].Step)
			require.Equal(t, "FAILED", res.Steps[0].Status)
		})

		t.Run("hyperbridge set config error", func(t *testing.T) {
			bridgeType := uint8(0)
			u := NewCrosschainConfigUsecase(baseChainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{
				active: map[string]*entities.SmartContract{
					contractKey(destID, entities.ContractTypeAdapterHyperbridge): {
						ID:              uuid.New(),
						ChainUUID:       destID,
						Type:            entities.ContractTypeAdapterHyperbridge,
						ContractAddress: "0x2222222222222222222222222222222222222222",
						IsActive:        true,
					},
				},
			}, nil, &crosschainAdapterStub{
				statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
					return &OnchainAdapterStatus{
						DefaultBridgeType: 0,
						AdapterType0:      "0x1111111111111111111111111111111111111111",
					}, nil
				},
				setHyperbridgeCfgFn: func(context.Context, string, string, string, string) (string, []string, error) {
					return "", nil, errors.New("set hyperbridge failed")
				},
			})
			res, err := u.AutoFix(context.Background(), &AutoFixRequest{
				SourceChainID: "eip155:8453",
				DestChainID:   "eip155:42161",
				BridgeType:    &bridgeType,
			})
			require.NoError(t, err)
			require.Len(t, res.Steps, 3)
			require.Equal(t, "setHyperbridgeConfig", res.Steps[2].Step)
			require.Equal(t, "FAILED", res.Steps[2].Status)
		})
	}
