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
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

type approvalCallStep struct {
	out []byte
	err error
}

func mockApprovalFactory(rpcURL string, steps []approvalCallStep) *blockchain.ClientFactory {
	factory := blockchain.NewClientFactory()
	idx := 0
	factory.RegisterEVMClient(rpcURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
		if idx >= len(steps) {
			return nil, errors.New("unexpected call")
		}
		step := steps[idx]
		idx++
		return step.out, step.err
	}))
	return factory
}

func TestPaymentUsecase_CalculateOnchainApprovalAmount_MockClientBranches(t *testing.T) {
	chainID := uuid.New()
	const rpcURL = "mock://approval"
	chainRepo := &approvalChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: rpcURL}}

	scRepo := &scRepoStub{getActiveFn: func(ctx context.Context, chainID uuid.UUID, t entities.SmartContractType) (*entities.SmartContract, error) {
		return nil, domainerrors.ErrNotFound
	}}
	t.Run("quote lower than totalCharged uses totalCharged", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: scRepo,
			chainRepo:    chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256", "uint256"}, big.NewInt(900), big.NewInt(10)))},
			}),
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		val, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1050",
		}, "0x1111111111111111111111111111111111111111")
		require.NoError(t, err)
		require.Equal(t, "1050", val)
	})

	t.Run("fixed fee call error", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: scRepo,
			chainRepo:    chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: []byte{}, err: nil},                       // quote decode fail -> fallback
				{out: nil, err: errors.New("fixed fee failed")}, // FIXED_BASE_FEE call
			}),
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to call FIXED_BASE_FEE")
	})

	t.Run("fee bps call error", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: scRepo,
			chainRepo:    chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: []byte{}, err: nil}, // quote decode fail
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256"}, big.NewInt(50))), err: nil},
				{out: nil, err: errors.New("bps failed")},
			}),
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to call FEE_RATE_BPS")
	})

	t.Run("fallback success percentage path", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: scRepo,
			chainRepo:    chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: []byte{}, err: nil}, // quote decode fail
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256"}, big.NewInt(10))), err: nil},
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256"}, big.NewInt(1000))), err: nil}, // 10%
			}),
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		val, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1200",
		}, "0x1111111111111111111111111111111111111111")
		require.NoError(t, err)
		// amount + max(fixed=10, percentage=100) => 1100, then max(totalCharged=1200)
		require.Equal(t, "1200", val)
	})

	t.Run("quote call error then fallback succeeds", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: scRepo,
			chainRepo:    chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: nil, err: errors.New("quote call failed")},
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256"}, big.NewInt(20))), err: nil},
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256"}, big.NewInt(100))), err: nil}, // 1%
			}),
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		val, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.NoError(t, err)
		require.Equal(t, "1010", val)
	})

	t.Run("totalCharged parse fallback and RPC list resolution", func(t *testing.T) {
		rpcListURL := "mock://approval-rpcs"
		chainRepoRPCList := &approvalChainRepoStub{chain: &entities.Chain{
			ID:   chainID,
			RPCs: []entities.ChainRPC{{URL: rpcListURL, IsActive: true}},
		}}
		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(rpcListURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return commonFromHex(t, mustPackOutputs(t, []string{"uint256", "uint256"}, big.NewInt(1100), big.NewInt(10))), nil
		}))

		u := &PaymentUsecase{
			chainRepo:        chainRepoRPCList,
			clientFactory:    factory,
			contractRepo:     scRepo,
			routePolicyRepo:  nil,
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		val, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "not-a-number",
		}, "0x1111111111111111111111111111111111111111")
		require.NoError(t, err)
		require.Equal(t, "1100", val)
	})

	t.Run("decode FIXED_BASE_FEE failed", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: scRepo,
			chainRepo:    chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: []byte{}, err: nil}, // quote decode fail -> fallback
				{out: []byte{0x01}, err: nil},
			}),
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode FIXED_BASE_FEE")
	})

	t.Run("decode FEE_RATE_BPS failed", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: scRepo,
			chainRepo:    chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: []byte{}, err: nil}, // quote decode fail
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256"}, big.NewInt(10))), err: nil},
				{out: []byte{0x01}, err: nil},
			}),
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode FEE_RATE_BPS")
	})
}

func commonFromHex(t *testing.T, hexString string) []byte {
	t.Helper()
	return common.FromHex(hexString)
}
