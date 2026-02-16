package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
)

func TestPaymentRequestUsecase_GetPaymentRequest_ChainResolutionBranches(t *testing.T) {
	t.Run("resolve chain id from network id when chain uuid is empty", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		mr := new(MockMerchantRepository)
		wr := new(MockWalletRepository)
		cr := new(MockChainRepository)
		sr := new(MockSmartContractRepository)
		tr := new(MockTokenRepository)
		uc := usecases.NewPaymentRequestUsecase(pr, mr, wr, cr, sr, tr)

		requestID := uuid.New()
		resolvedChainID := uuid.New()
		req := &entities.PaymentRequest{
			ID:        requestID,
			ChainID:   uuid.Nil,
			NetworkID: "eip155:8453",
			Amount:    "1000",
			Decimals:  6,
			Status:    entities.PaymentRequestStatusCompleted,
			ExpiresAt: time.Now().Add(time.Hour),
		}

		pr.On("GetByID", context.Background(), requestID).Return(req, nil).Once()
		cr.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{
			ID:      resolvedChainID,
			ChainID: "8453",
			Type:    entities.ChainTypeEVM,
		}, nil).Once()
		sr.On("GetActiveContract", context.Background(), resolvedChainID, entities.ContractTypeGateway).Return(&entities.SmartContract{
			ContractAddress: "0x1111111111111111111111111111111111111111",
		}, nil).Once()

		got, tx, err := uc.GetPaymentRequest(context.Background(), requestID)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.NotNil(t, tx)
		assert.Equal(t, resolvedChainID, got.ChainID)
		assert.Equal(t, "eip155:8453", got.NetworkID)
	})

	t.Run("resolve network id from chain id when network is empty", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		mr := new(MockMerchantRepository)
		wr := new(MockWalletRepository)
		cr := new(MockChainRepository)
		sr := new(MockSmartContractRepository)
		tr := new(MockTokenRepository)
		uc := usecases.NewPaymentRequestUsecase(pr, mr, wr, cr, sr, tr)

		requestID := uuid.New()
		chainID := uuid.New()
		req := &entities.PaymentRequest{
			ID:        requestID,
			ChainID:   chainID,
			NetworkID: "",
			Amount:    "1000",
			Decimals:  6,
			Status:    entities.PaymentRequestStatusCompleted,
			ExpiresAt: time.Now().Add(time.Hour),
		}

		pr.On("GetByID", context.Background(), requestID).Return(req, nil).Once()
		cr.On("GetByID", context.Background(), chainID).Return(&entities.Chain{
			ID:      chainID,
			ChainID: "8453",
			Type:    entities.ChainTypeEVM,
		}, nil).Once()
		sr.On("GetActiveContract", context.Background(), chainID, entities.ContractTypeGateway).Return((*entities.SmartContract)(nil), nil).Once()

		got, tx, err := uc.GetPaymentRequest(context.Background(), requestID)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.NotNil(t, tx)
		assert.Equal(t, "eip155:8453", got.NetworkID)
		assert.Equal(t, "eip155:8453", tx.ChainID)
	})
}
