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

	t.Run("quote lower than totalCharged uses totalCharged", func(t *testing.T) {
		u := &PaymentUsecase{
			chainRepo: chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256", "uint256"}, big.NewInt(900), big.NewInt(10)))},
			}),
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
			chainRepo: chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: []byte{}, err: nil},                      // quote decode fail -> fallback
				{out: nil, err: errors.New("fixed fee failed")}, // FIXED_BASE_FEE call
			}),
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
			chainRepo: chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: []byte{}, err: nil}, // quote decode fail
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256"}, big.NewInt(50))), err: nil},
				{out: nil, err: errors.New("bps failed")},
			}),
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
			chainRepo: chainRepo,
			clientFactory: mockApprovalFactory(rpcURL, []approvalCallStep{
				{out: []byte{}, err: nil}, // quote decode fail
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256"}, big.NewInt(10))), err: nil},
				{out: commonFromHex(t, mustPackOutputs(t, []string{"uint256"}, big.NewInt(1000))), err: nil}, // 10%
			}),
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
}

func commonFromHex(t *testing.T, hexString string) []byte {
	t.Helper()
	return common.FromHex(hexString)
}
