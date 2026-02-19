package usecases

import (
	"context"
	"math/big"
	"testing"

	"pay-chain.backend/internal/infrastructure/blockchain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

type feeConfigRepoStub struct {
	getByChainAndTokenFn func(ctx context.Context, chainID, tokenID uuid.UUID) (*entities.FeeConfig, error)
}

func (s *feeConfigRepoStub) GetByChainAndToken(ctx context.Context, chainID, tokenID uuid.UUID) (*entities.FeeConfig, error) {
	if s.getByChainAndTokenFn != nil {
		return s.getByChainAndTokenFn(ctx, chainID, tokenID)
	}
	return nil, nil
}
func (s *feeConfigRepoStub) GetByID(context.Context, uuid.UUID) (*entities.FeeConfig, error) {
	return nil, nil
}
func (s *feeConfigRepoStub) List(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.FeeConfig, int64, error) {
	return nil, 0, nil
}
func (s *feeConfigRepoStub) Create(context.Context, *entities.FeeConfig) error { return nil }
func (s *feeConfigRepoStub) Update(context.Context, *entities.FeeConfig) error { return nil }
func (s *feeConfigRepoStub) Delete(context.Context, uuid.UUID) error           { return nil }

func TestPaymentUsecase_CalculateFees_ConfigAndFallback(t *testing.T) {
	ctx := context.Background()
	sourceChainUUID := uuid.New()
	sourceTokenID := uuid.New()

	t.Run("apply min fee after discount", func(t *testing.T) {
		maxFee := "10"
		u := &PaymentUsecase{
			feeConfigRepo: &feeConfigRepoStub{
				getByChainAndTokenFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.FeeConfig, error) {
					return &entities.FeeConfig{
						FixedBaseFee:       "1",
						PlatformFeePercent: "0.1",
						MinFee:             "3",
						MaxFee:             &maxFee,
					}, nil
				},
			},
		}

		// 1000 with decimals=2 => 10.00 token
		fees := u.CalculateFees(
			ctx,
			big.NewInt(1000),
			2,
			"eip155:8453",
			"eip155:8453",
			sourceChainUUID,
			sourceChainUUID, // same chain
			sourceTokenID,
			"native",
			"native",
			2,
			0.5, // discount 50%
		)
		require.Equal(t, "300", fees.PlatformFee)
		require.Equal(t, "0", fees.BridgeFee)
		require.Equal(t, "300", fees.TotalFee)
		require.Equal(t, "700", fees.NetAmount)
	})

	t.Run("cross-chain quote failure uses flat bridge fee fallback", func(t *testing.T) {
		sourceID := uuid.New()
		destID := uuid.New()
		sourceChain := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
		destChain := &entities.Chain{ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM}
		chainRepo := &quoteChainRepoStub{
			byCAIP2: map[string]*entities.Chain{
				"eip155:8453":  sourceChain,
				"eip155:42161": destChain,
			},
			byID: map[uuid.UUID]*entities.Chain{
				sourceID: sourceChain,
				destID:   destChain,
			},
		}
		u := &PaymentUsecase{
			feeConfigRepo: &feeConfigRepoStub{},
			chainRepo:     chainRepo,
			chainResolver: NewChainResolver(chainRepo),
			contractRepo: &quoteContractRepoStub{
				router: &entities.SmartContract{
					ContractAddress: "0x1111111111111111111111111111111111111111",
					Type:            entities.ContractTypeRouter,
				},
			},
		}

		fees := u.CalculateFees(
			ctx,
			big.NewInt(1000), // 10.00 token
			2,
			"eip155:8453",
			"eip155:42161",
			sourceID,
			destID,
			sourceTokenID,
			"native",
			"native",
			2,
			0,
		)

		// platform: min(0.3%*10.00, 0.50) = min(0.03, 0.50) = 0.03 tokens => 3 in 2 decimals
		// bridge fallback: 0.10 tokens => 10 in 2 decimals
		require.Equal(t, "3", fees.PlatformFee)
		require.Equal(t, "10", fees.BridgeFee)
		require.Equal(t, "13", fees.TotalFee)
		require.Equal(t, "987", fees.NetAmount)
	})

	t.Run("max fee clamp is applied", func(t *testing.T) {
		maxFee := "0.4" // Cap below FixedBaseFee (1.0) and Percentage (10.0)
		u := &PaymentUsecase{
			feeConfigRepo: &feeConfigRepoStub{
				getByChainAndTokenFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.FeeConfig, error) {
					return &entities.FeeConfig{
						FixedBaseFee:       "1",
						PlatformFeePercent: "1", // 100%
						MinFee:             "0",
						MaxFee:             &maxFee,
					}, nil
				},
			},
		}

		fees := u.CalculateFees(
			ctx,
			big.NewInt(1000), // 10.00 token
			2,
			"eip155:8453",
			"eip155:8453",
			sourceChainUUID,
			sourceChainUUID,
			sourceTokenID,
			"native",
			"native",
			2,
			0,
		)
		// min(10.0, 1.0) = 1.0 token.
		// Then clamp to maxFee (0.4) = 0.4 token.
		// 0.4 tokens in 2 decimals = 40 units.
		require.Equal(t, "40", fees.PlatformFee)
		require.Equal(t, "40", fees.TotalFee)
		require.Equal(t, "960", fees.NetAmount)
	})

	t.Run("invalid config numbers fallback to defaults", func(t *testing.T) {
		maxFee := "invalid-max"
		u := &PaymentUsecase{
			feeConfigRepo: &feeConfigRepoStub{
				getByChainAndTokenFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.FeeConfig, error) {
					return &entities.FeeConfig{
						FixedBaseFee:       "invalid-base",
						PlatformFeePercent: "invalid-percent",
						MinFee:             "invalid-min",
						MaxFee:             &maxFee,
					}, nil
				},
			},
		}
		fees := u.CalculateFees(
			ctx,
			big.NewInt(1000), // 10.00 token
			2,
			"eip155:8453",
			"eip155:8453",
			sourceChainUUID,
			sourceChainUUID,
			sourceTokenID,
			"native",
			"native",
			2,
			0,
		)
		// defaults => platform fee: min(0.3% * 10.00, 0.50) = 0.03 token => 3 in 2 decimals.
		require.Equal(t, "3", fees.PlatformFee)
		require.Equal(t, "3", fees.TotalFee)
		require.Equal(t, "997", fees.NetAmount)
	})

	t.Run("cross-chain quote success uses quoted bridge fee", func(t *testing.T) {
		sourceID := uuid.New()
		destID := uuid.New()
		sourceChain := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "mock://fee-ok"}
		destChain := &entities.Chain{ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM}
		chainRepo := &quoteChainRepoStub{
			byCAIP2: map[string]*entities.Chain{
				"eip155:8453":  sourceChain,
				"eip155:42161": destChain,
			},
			byID: map[uuid.UUID]*entities.Chain{
				sourceID: sourceChain,
				destID:   destChain,
			},
		}
		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(sourceChain.RPCURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{0x14}, nil // 20 (smallest unit)
		}))
		u := &PaymentUsecase{
			feeConfigRepo: &feeConfigRepoStub{},
			chainRepo:     chainRepo,
			chainResolver: NewChainResolver(chainRepo),
			contractRepo: &quoteContractRepoStub{
				router: &entities.SmartContract{
					ContractAddress: "0x1111111111111111111111111111111111111111",
					Type:            entities.ContractTypeRouter,
				},
			},
			clientFactory: factory,
		}

		fees := u.CalculateFees(
			ctx,
			big.NewInt(1000), // 10.00 token
			2,
			"eip155:8453",
			"eip155:42161",
			sourceID,
			destID,
			sourceTokenID,
			"native",
			"native",
			2,
			0,
		)
		require.Equal(t, "3", fees.PlatformFee)
		require.Equal(t, "20", fees.BridgeFee)
		require.Equal(t, "23", fees.TotalFee)
		require.Equal(t, "977", fees.NetAmount)
	})
}
