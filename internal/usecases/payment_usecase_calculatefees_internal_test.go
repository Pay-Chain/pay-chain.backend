package usecases

import (
	"context"
	"math/big"
	"testing"

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
			sourceTokenID,
			"native",
			"native",
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
			sourceTokenID,
			"0x1111111111111111111111111111111111111111",
			"0x2222222222222222222222222222222222222222",
			0,
		)

		// base 0.5 + 0.3%*10 = 0.53, bridge fallback 0.10
		require.Equal(t, "53", fees.PlatformFee)
		require.Equal(t, "10", fees.BridgeFee)
		require.Equal(t, "63", fees.TotalFee)
		require.Equal(t, "936", fees.NetAmount)
	})
}
