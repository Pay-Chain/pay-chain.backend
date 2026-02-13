package usecases_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/internal/usecases"
)

func TestCreatePayment_Success(t *testing.T) {
	// Setup Mocks
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockMerchantRepo := new(MockMerchantRepository)
	mockContractRepo := new(MockSmartContractRepository)
	mockChainRepo := new(MockChainRepository)
	mockTokenRepo := new(MockTokenRepository)
	mockUOW := new(MockUnitOfWork)

	// Client factory mock is complex, let's skip deep chain interaction logic if possible or Mock it?
	// The usecase uses clientFactory.GetEVMClient in getBridgeFeeQuote (only if cross-chain).
	// We will test SAME-CHAIN payment to avoid blockchain calls for now.
	clientFactory := blockchain.NewClientFactory() // Real factory but unused if we don't trigger cross-chain fee quote

	uc := usecases.NewPaymentUsecase(
		mockPaymentRepo,
		mockEventRepo,
		mockWalletRepo,
		mockMerchantRepo,
		mockContractRepo,
		mockChainRepo,
		mockTokenRepo,
		nil,
		nil,
		mockUOW,
		clientFactory,
	)

	srcChainID := uuid.New()
	token := &entities.Token{ID: uuid.New(), Symbol: "USDC", Decimals: 6}
	srcChain := &entities.Chain{
		ID:      srcChainID,
		ChainID: "1",
		RPCs: []entities.ChainRPC{
			{URL: "https://eth.llama.rpc.com"},
		},
	}
	destChain := &entities.Chain{
		ID:      uuid.New(), // destination chain ID
		ChainID: "137",
		RPCs: []entities.ChainRPC{
			{URL: "https://polygon-rpc.com"},
		},
	}

	req := &entities.CreatePaymentInput{
		SourceChainID:      "eip155:1",
		DestChainID:        "eip155:137",
		SourceTokenAddress: "0x123",
		DestTokenAddress:   "0x456",
		Amount:             "1000000000000000000",
		Decimals:           18,
		ReceiverAddress:    "0xReceiver",
	}

	// Mocks setup
	mockUOW.On("Do", mock.Anything, mock.Anything).Return(nil)
	mockPaymentRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Payment")).Return(nil)
	mockEventRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.PaymentEvent")).Return(nil)
	mockChainRepo.On("GetByChainID", mock.Anything, "eip155:1").Return(srcChain, nil)
	mockChainRepo.On("GetByChainID", mock.Anything, "eip155:137").Return(destChain, nil)
	mockChainRepo.On("GetByChainID", mock.Anything, "1").Return(srcChain, nil)
	mockChainRepo.On("GetByChainID", mock.Anything, "137").Return(destChain, nil).Maybe()
	mockChainRepo.On("GetByID", mock.Anything, srcChain.ID).Return(srcChain, nil)
	mockChainRepo.On("GetByID", mock.Anything, destChain.ID).Return(destChain, nil)
	mockTokenRepo.On("GetByAddress", mock.Anything, "0x123", srcChain.ID).Return(token, nil)
	mockTokenRepo.On("GetByAddress", mock.Anything, "0x456", destChain.ID).Return(token, nil)

	// Mock Gateway for source chain
	mockContractRepo.On("GetActiveContract", mock.Anything, srcChain.ID, entities.ContractTypeGateway).Return(&entities.SmartContract{
		ID:              uuid.New(),
		ContractAddress: "0xGatewayAddress",
		Type:            entities.ContractTypeGateway,
	}, nil)

	// Mock Router for source chain (fee calculation)
	mockContractRepo.On("GetActiveContract", mock.Anything, srcChain.ID, entities.ContractTypeRouter).Return(&entities.SmartContract{
		ID:              uuid.New(),
		ContractAddress: "0xSrcRouterAddress",
		Type:            entities.ContractTypeRouter,
	}, nil)

	// Execute
	payment, err := uc.CreatePayment(context.Background(), uuid.New(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, req.Amount, payment.SourceAmount)
	assert.Equal(t, entities.PaymentStatusPending, payment.Status)

	mockPaymentRepo.AssertExpectations(t)
	mockEventRepo.AssertExpectations(t)
	mockUOW.AssertExpectations(t)
	mockChainRepo.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
	mockContractRepo.AssertExpectations(t)
}

func TestProcessIndexerWebhook_Locking(t *testing.T) {
	// Setup Mocks
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockUOW := new(MockUnitOfWork)

	// Context with Lock Key to verify propagation
	ctx := context.Background()
	lockedKey := "locked"
	lockedCtx := context.WithValue(ctx, lockedKey, true)

	uc := usecases.NewWebhookUsecase(
		mockPaymentRepo,
		mockEventRepo,
		mockRequestRepo,
		mockUOW,
	)

	// Payload data matching the anonymous struct in ProcessIndexerWebhook
	paymentData := map[string]interface{}{
		"paymentId":    "00000000-0000-0000-0000-000000000001",
		"status":       "completed",
		"sourceTxHash": "0xSourceTxHash",
		"destTxHash":   "0xDestTxHash",
	}
	dataBytes, _ := json.Marshal(paymentData)

	paymentID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// UOW Expectation
	mockUOW.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		fn(ctx) // Execute with original context, internal logic calls WithLock
	})

	// WithLock Expectation
	mockUOW.On("WithLock", ctx).Return(lockedCtx)

	// Payment Repo Expectation - MUST be called with lockedCtx
	mockPaymentRepo.On("GetByID", lockedCtx, paymentID).Return(&entities.Payment{
		ID:           paymentID,
		Status:       entities.PaymentStatusPending,
		SourceAmount: "1000000000000000000",
	}, nil)

	mockPaymentRepo.On("UpdateStatus", lockedCtx, paymentID, entities.PaymentStatusCompleted).Return(nil)

	// Event Repo Expectation
	mockEventRepo.On("Create", lockedCtx, mock.AnythingOfType("*entities.PaymentEvent")).Return(nil)

	// Execute
	err := uc.ProcessIndexerWebhook(ctx, "PAYMENT_COMPLETED", json.RawMessage(dataBytes))

	// Assert
	assert.NoError(t, err)
	mockUOW.AssertExpectations(t)
	mockPaymentRepo.AssertExpectations(t)
	mockEventRepo.AssertExpectations(t)
}
