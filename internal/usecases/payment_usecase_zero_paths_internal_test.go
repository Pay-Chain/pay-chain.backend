package usecases

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

type scRepoStub struct {
	getActiveFn func(ctx context.Context, chainID uuid.UUID, t entities.SmartContractType) (*entities.SmartContract, error)
}

func (s *scRepoStub) Create(context.Context, *entities.SmartContract) error { return nil }
func (s *scRepoStub) GetByID(context.Context, uuid.UUID) (*entities.SmartContract, error) {
	return nil, errors.New("not found")
}
func (s *scRepoStub) GetByChainAndAddress(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
	return nil, errors.New("not found")
}
func (s *scRepoStub) GetActiveContract(ctx context.Context, chainID uuid.UUID, t entities.SmartContractType) (*entities.SmartContract, error) {
	if s.getActiveFn != nil {
		return s.getActiveFn(ctx, chainID, t)
	}
	return nil, errors.New("not found")
}
func (s *scRepoStub) GetFiltered(context.Context, *uuid.UUID, entities.SmartContractType, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *scRepoStub) GetByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *scRepoStub) GetAll(context.Context, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *scRepoStub) Update(context.Context, *entities.SmartContract) error { return nil }
func (s *scRepoStub) SoftDelete(context.Context, uuid.UUID) error           { return nil }

func TestPaymentUsecase_ResolveVaultAddressForApproval(t *testing.T) {
	chainID := uuid.New()

	u := &PaymentUsecase{
		contractRepo: &scRepoStub{
			getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return &entities.SmartContract{ContractAddress: "0x1111111111111111111111111111111111111111"}, nil
			},
		},
	}
	got := u.resolveVaultAddressForApproval(chainID, "")
	require.Equal(t, "0x1111111111111111111111111111111111111111", got)

	u2 := &PaymentUsecase{
		contractRepo: &scRepoStub{
			getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return nil, errors.New("missing")
			},
		},
	}
	got = u2.resolveVaultAddressForApproval(chainID, "")
	require.Equal(t, "", got)
}

func TestPaymentUsecase_CalculateOnchainApprovalAmount_Guards(t *testing.T) {
	u := &PaymentUsecase{}

	_, err := u.calculateOnchainApprovalAmount(nil, "0xabc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid payment or gateway address")

	_, err = u.calculateOnchainApprovalAmount(&entities.Payment{SourceAmount: "bad"}, "0xabc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid source amount")
}

func TestPaymentUsecase_ResolveBridgeOrder_DefaultFallback(t *testing.T) {
	u := &PaymentUsecase{}
	order := u.resolveBridgeOrder(context.Background(), uuid.New(), uuid.New(), "eip155:8453", "eip155:42161")
	require.NotEmpty(t, order)
}

func TestPaymentUsecase_QuoteBridgeFeeByType_PackError(t *testing.T) {
	u := &PaymentUsecase{}
	require.Panics(t, func() {
		_, _ = u.quoteBridgeFeeByType(context.Background(), nil, "0xrouter", "eip155:42161", 0, "0x1", "0x2", nil, big.NewInt(0))
	})

	require.Panics(t, func() {
		_, _ = u.quoteBridgeFeeByType(context.Background(), nil, "0xrouter", "eip155:42161", 0, "0x1", "0x2", big.NewInt(1), big.NewInt(0))
	})
}
