package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
)

func newPaymentRequestUC(
	pr *MockPaymentRequestRepository,
	mr *MockMerchantRepository,
	wr *MockWalletRepository,
	cr *MockChainRepository,
	sr *MockSmartContractRepository,
	tr *MockTokenRepository,
) *usecases.PaymentRequestUsecase {
	return usecases.NewPaymentRequestUsecase(pr, mr, wr, cr, sr, tr)
}

func TestPaymentRequestUsecase_CreatePaymentRequest_MerchantNotFound(t *testing.T) {
	pr := new(MockPaymentRequestRepository)
	mr := new(MockMerchantRepository)
	wr := new(MockWalletRepository)
	cr := new(MockChainRepository)
	sr := new(MockSmartContractRepository)
	tr := new(MockTokenRepository)
	uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

	userID := uuid.New()
	mr.On("GetByUserID", context.Background(), userID).Return(nil, assert.AnError).Once()

	_, err := uc.CreatePaymentRequest(context.Background(), usecases.CreatePaymentRequestInput{
		UserID:       userID,
		ChainID:      "eip155:8453",
		TokenAddress: "0x1",
		Amount:       "1",
		Decimals:     6,
	})
	assert.Error(t, err)
}

func TestPaymentRequestUsecase_CreatePaymentRequest_MerchantInactive(t *testing.T) {
	pr := new(MockPaymentRequestRepository)
	mr := new(MockMerchantRepository)
	wr := new(MockWalletRepository)
	cr := new(MockChainRepository)
	sr := new(MockSmartContractRepository)
	tr := new(MockTokenRepository)
	uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

	userID := uuid.New()
	mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
		ID:     uuid.New(),
		UserID: userID,
		Status: entities.MerchantStatusPending,
	}, nil).Once()

	_, err := uc.CreatePaymentRequest(context.Background(), usecases.CreatePaymentRequestInput{
		UserID:       userID,
		ChainID:      "eip155:8453",
		TokenAddress: "0x1",
		Amount:       "1",
		Decimals:     6,
	})
	assert.Error(t, err)
	assert.Equal(t, "invalid input", err.Error())
}

func TestPaymentRequestUsecase_CreatePaymentRequest_SuccessEVM(t *testing.T) {
	pr := new(MockPaymentRequestRepository)
	mr := new(MockMerchantRepository)
	wr := new(MockWalletRepository)
	cr := new(MockChainRepository)
	sr := new(MockSmartContractRepository)
	tr := new(MockTokenRepository)
	uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

	userID := uuid.New()
	merchantID := uuid.New()
	chainID := uuid.New()
	tokenID := uuid.New()
	input := usecases.CreatePaymentRequestInput{
		UserID:       userID,
		ChainID:      "eip155:8453",
		TokenAddress: "0xToken",
		Amount:       "1.5",
		Decimals:     6,
		Description:  "invoice",
	}

	mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
		ID:     merchantID,
		UserID: userID,
		Status: entities.MerchantStatusActive,
	}, nil).Once()
	wr.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{
		{ID: uuid.New(), Address: "0xMerchant", IsPrimary: true},
	}, nil).Once()
	cr.On("GetByCAIP2", context.Background(), input.ChainID).Return(&entities.Chain{
		ID:      chainID,
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}, nil).Once()
	tr.On("GetByAddress", context.Background(), input.TokenAddress, chainID).Return(&entities.Token{
		ID:       tokenID,
		Decimals: 6,
	}, nil).Once()
	sr.On("GetActiveContract", context.Background(), chainID, entities.ContractTypeGateway).Return(&entities.SmartContract{
		ID:              uuid.New(),
		ContractAddress: "0xGateway",
	}, nil).Once()
	pr.On("Create", context.Background(), mock.AnythingOfType("*entities.PaymentRequest")).Return(nil).Once()

	out, err := uc.CreatePaymentRequest(context.Background(), input)
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.NotEmpty(t, out.RequestID)
	assert.Equal(t, usecases.PaymentRequestExpiryMinutes*60, out.ExpiresInSecs)
	assert.NotNil(t, out.TxData)
	assert.Equal(t, "eip155:8453", out.TxData.ChainID)
	assert.Contains(t, out.TxData.Hex, usecases.PayRequestSelector)
}

func TestPaymentRequestUsecase_CreatePaymentRequest_DecimalsMismatch(t *testing.T) {
	pr := new(MockPaymentRequestRepository)
	mr := new(MockMerchantRepository)
	wr := new(MockWalletRepository)
	cr := new(MockChainRepository)
	sr := new(MockSmartContractRepository)
	tr := new(MockTokenRepository)
	uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

	userID := uuid.New()
	merchantID := uuid.New()
	chainID := uuid.New()
	input := usecases.CreatePaymentRequestInput{
		UserID:       userID,
		ChainID:      "eip155:8453",
		TokenAddress: "0xToken",
		Amount:       "1.5",
		Decimals:     18,
	}

	mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
		ID:     merchantID,
		UserID: userID,
		Status: entities.MerchantStatusActive,
	}, nil).Once()
	wr.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{
		{ID: uuid.New(), Address: "0xMerchant", IsPrimary: true},
	}, nil).Once()
	cr.On("GetByCAIP2", context.Background(), input.ChainID).Return(&entities.Chain{
		ID:      chainID,
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}, nil).Once()
	tr.On("GetByAddress", context.Background(), input.TokenAddress, chainID).Return(&entities.Token{
		ID:       uuid.New(),
		Decimals: 6,
	}, nil).Once()
	sr.On("GetActiveContract", context.Background(), chainID, entities.ContractTypeGateway).Return(nil, nil).Once()

	_, err := uc.CreatePaymentRequest(context.Background(), input)
	assert.Error(t, err)
	assert.Equal(t, "invalid input", err.Error())
}

func TestPaymentRequestUsecase_CreatePaymentRequest_NoPrimaryWallet_UsesFirstWallet(t *testing.T) {
	pr := new(MockPaymentRequestRepository)
	mr := new(MockMerchantRepository)
	wr := new(MockWalletRepository)
	cr := new(MockChainRepository)
	sr := new(MockSmartContractRepository)
	tr := new(MockTokenRepository)
	uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

	userID := uuid.New()
	merchantID := uuid.New()
	chainID := uuid.New()
	tokenID := uuid.New()
	input := usecases.CreatePaymentRequestInput{
		UserID:       userID,
		ChainID:      "eip155:8453",
		TokenAddress: "0xToken",
		Amount:       "2.0",
		Decimals:     6,
	}

	firstWallet := &entities.Wallet{ID: uuid.New(), Address: "0xFirstWallet", IsPrimary: false}
	mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
		ID:     merchantID,
		UserID: userID,
		Status: entities.MerchantStatusActive,
	}, nil).Once()
	wr.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{
		firstWallet,
		{ID: uuid.New(), Address: "0xSecondWallet", IsPrimary: false},
	}, nil).Once()
	cr.On("GetByCAIP2", context.Background(), input.ChainID).Return(&entities.Chain{
		ID:      chainID,
		Type:    entities.ChainTypeEVM,
		ChainID: "8453",
	}, nil).Once()
	tr.On("GetByAddress", context.Background(), input.TokenAddress, chainID).Return(&entities.Token{
		ID:       tokenID,
		Decimals: 6,
	}, nil).Once()
	sr.On("GetActiveContract", context.Background(), chainID, entities.ContractTypeGateway).Return(nil, nil).Once()
	pr.On("Create", context.Background(), mock.MatchedBy(func(req *entities.PaymentRequest) bool {
		return req != nil && req.WalletAddress == firstWallet.Address
	})).Return(nil).Once()

	out, err := uc.CreatePaymentRequest(context.Background(), input)
	assert.NoError(t, err)
	assert.NotNil(t, out)
}

func TestPaymentRequestUsecase_GetPaymentRequest_ExpirePending(t *testing.T) {
	pr := new(MockPaymentRequestRepository)
	mr := new(MockMerchantRepository)
	wr := new(MockWalletRepository)
	cr := new(MockChainRepository)
	sr := new(MockSmartContractRepository)
	tr := new(MockTokenRepository)
	uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

	requestID := uuid.New()
	chainID := uuid.New()
	request := &entities.PaymentRequest{
		ID:        requestID,
		ChainID:   chainID,
		NetworkID: "eip155:8453",
		Amount:    "1000",
		Decimals:  6,
		Status:    entities.PaymentRequestStatusPending,
		ExpiresAt: time.Now().Add(-time.Minute),
	}

	pr.On("GetByID", context.Background(), requestID).Return(request, nil).Once()
	pr.On("UpdateStatus", context.Background(), requestID, entities.PaymentRequestStatusExpired).Return(nil).Once()
	sr.On("GetActiveContract", context.Background(), chainID, entities.ContractTypeGateway).Return(&entities.SmartContract{
		ContractAddress: "0xGateway",
	}, nil).Once()

	got, tx, err := uc.GetPaymentRequest(context.Background(), requestID)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.NotNil(t, tx)
	assert.Equal(t, entities.PaymentRequestStatusExpired, got.Status)
	assert.Contains(t, tx.Hex, usecases.PayRequestSelector)
}

func TestPaymentRequestUsecase_ListAndMarkCompleted(t *testing.T) {
	pr := new(MockPaymentRequestRepository)
	mr := new(MockMerchantRepository)
	wr := new(MockWalletRepository)
	cr := new(MockChainRepository)
	sr := new(MockSmartContractRepository)
	tr := new(MockTokenRepository)
	uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

	userID := uuid.New()
	merchantID := uuid.New()
	mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{ID: merchantID}, nil).Once()
	pr.On("GetByMerchantID", context.Background(), merchantID, 10, 0).Return([]*entities.PaymentRequest{}, 0, nil).Once()

	items, total, err := uc.ListPaymentRequests(context.Background(), userID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, items, 0)
	assert.Equal(t, 0, total)

	requestID := uuid.New()
	pr.On("MarkCompleted", context.Background(), requestID, "0xhash").Return(nil).Once()
	err = uc.MarkPaymentCompleted(context.Background(), requestID, "0xhash")
	assert.NoError(t, err)
}

func TestPaymentRequestUsecase_CreatePaymentRequest_ErrorBranches(t *testing.T) {
	t.Run("wallet repository error", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		mr := new(MockMerchantRepository)
		wr := new(MockWalletRepository)
		cr := new(MockChainRepository)
		sr := new(MockSmartContractRepository)
		tr := new(MockTokenRepository)
		uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

		userID := uuid.New()
		mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
			ID:     uuid.New(),
			UserID: userID,
			Status: entities.MerchantStatusActive,
		}, nil).Once()
		wr.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{}, assert.AnError).Once()

		_, err := uc.CreatePaymentRequest(context.Background(), usecases.CreatePaymentRequestInput{
			UserID:       userID,
			ChainID:      "eip155:8453",
			TokenAddress: "0x1",
			Amount:       "1",
			Decimals:     6,
		})
		assert.Error(t, err)
	})

	t.Run("no wallet found", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		mr := new(MockMerchantRepository)
		wr := new(MockWalletRepository)
		cr := new(MockChainRepository)
		sr := new(MockSmartContractRepository)
		tr := new(MockTokenRepository)
		uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

		userID := uuid.New()
		mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
			ID:     uuid.New(),
			UserID: userID,
			Status: entities.MerchantStatusActive,
		}, nil).Once()
		wr.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{}, nil).Once()

		_, err := uc.CreatePaymentRequest(context.Background(), usecases.CreatePaymentRequestInput{
			UserID:       userID,
			ChainID:      "eip155:8453",
			TokenAddress: "0x1",
			Amount:       "1",
			Decimals:     6,
		})
		assert.Error(t, err)
	})

	t.Run("invalid chain id", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		mr := new(MockMerchantRepository)
		wr := new(MockWalletRepository)
		cr := new(MockChainRepository)
		sr := new(MockSmartContractRepository)
		tr := new(MockTokenRepository)
		uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

		userID := uuid.New()
		mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
			ID:     uuid.New(),
			UserID: userID,
			Status: entities.MerchantStatusActive,
		}, nil).Once()
		wr.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{{Address: "0xM", IsPrimary: true}}, nil).Once()
		cr.On("GetByCAIP2", context.Background(), "bad").Return(nil, assert.AnError).Twice()
		cr.On("GetByChainID", context.Background(), "bad").Return(nil, assert.AnError).Twice()

		_, err := uc.CreatePaymentRequest(context.Background(), usecases.CreatePaymentRequestInput{
			UserID:       userID,
			ChainID:      "bad",
			TokenAddress: "0x1",
			Amount:       "1",
			Decimals:     6,
		})
		assert.Error(t, err)
	})

	t.Run("token fallback to native failed", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		mr := new(MockMerchantRepository)
		wr := new(MockWalletRepository)
		cr := new(MockChainRepository)
		sr := new(MockSmartContractRepository)
		tr := new(MockTokenRepository)
		uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

		userID := uuid.New()
		chainID := uuid.New()
		mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
			ID:     uuid.New(),
			UserID: userID,
			Status: entities.MerchantStatusActive,
		}, nil).Once()
		wr.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{{Address: "0xM", IsPrimary: true}}, nil).Once()
		cr.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}, nil).Once()
		tr.On("GetByAddress", context.Background(), "native", chainID).Return(nil, assert.AnError).Once()
		tr.On("GetNative", context.Background(), chainID).Return(nil, assert.AnError).Once()

		_, err := uc.CreatePaymentRequest(context.Background(), usecases.CreatePaymentRequestInput{
			UserID:       userID,
			ChainID:      "eip155:8453",
			TokenAddress: "native",
			Amount:       "1",
			Decimals:     6,
		})
		assert.Error(t, err)
	})

	t.Run("amount conversion invalid", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		mr := new(MockMerchantRepository)
		wr := new(MockWalletRepository)
		cr := new(MockChainRepository)
		sr := new(MockSmartContractRepository)
		tr := new(MockTokenRepository)
		uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

		userID := uuid.New()
		chainID := uuid.New()
		mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
			ID:     uuid.New(),
			UserID: userID,
			Status: entities.MerchantStatusActive,
		}, nil).Once()
		wr.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{{Address: "0xM", IsPrimary: true}}, nil).Once()
		cr.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}, nil).Once()
		tr.On("GetByAddress", context.Background(), "0xToken", chainID).Return(&entities.Token{ID: uuid.New(), Decimals: 6}, nil).Once()
		sr.On("GetActiveContract", context.Background(), chainID, entities.ContractTypeGateway).Return(nil, nil).Once()

		_, err := uc.CreatePaymentRequest(context.Background(), usecases.CreatePaymentRequestInput{
			UserID:       userID,
			ChainID:      "eip155:8453",
			TokenAddress: "0xToken",
			Amount:       "abc",
			Decimals:     6,
		})
		assert.Error(t, err)
	})

	t.Run("create payment request repository error", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		mr := new(MockMerchantRepository)
		wr := new(MockWalletRepository)
		cr := new(MockChainRepository)
		sr := new(MockSmartContractRepository)
		tr := new(MockTokenRepository)
		uc := newPaymentRequestUC(pr, mr, wr, cr, sr, tr)

		userID := uuid.New()
		chainID := uuid.New()
		tokenID := uuid.New()
		mr.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
			ID:     uuid.New(),
			UserID: userID,
			Status: entities.MerchantStatusActive,
		}, nil).Once()
		wr.On("GetByUserID", context.Background(), userID).Return([]*entities.Wallet{{Address: "0xM", IsPrimary: true}}, nil).Once()
		cr.On("GetByCAIP2", context.Background(), "eip155:8453").Return(&entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}, nil).Once()
		tr.On("GetByAddress", context.Background(), "0xToken", chainID).Return(&entities.Token{ID: tokenID, Decimals: 6}, nil).Once()
		sr.On("GetActiveContract", context.Background(), chainID, entities.ContractTypeGateway).Return(nil, nil).Once()
		pr.On("Create", context.Background(), mock.AnythingOfType("*entities.PaymentRequest")).Return(assert.AnError).Once()

		_, err := uc.CreatePaymentRequest(context.Background(), usecases.CreatePaymentRequestInput{
			UserID:       userID,
			ChainID:      "eip155:8453",
			TokenAddress: "0xToken",
			Amount:       "1",
			Decimals:     6,
		})
		assert.Error(t, err)
	})
}

func TestPaymentRequestUsecase_GetAndList_ErrorBranches(t *testing.T) {
	t.Run("get payment request not found", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		uc := newPaymentRequestUC(pr, new(MockMerchantRepository), new(MockWalletRepository), new(MockChainRepository), new(MockSmartContractRepository), new(MockTokenRepository))
		requestID := uuid.New()
		pr.On("GetByID", context.Background(), requestID).Return(nil, assert.AnError).Once()

		_, _, err := uc.GetPaymentRequest(context.Background(), requestID)
		assert.Error(t, err)
	})

	t.Run("list payment requests merchant not found", func(t *testing.T) {
		pr := new(MockPaymentRequestRepository)
		mr := new(MockMerchantRepository)
		uc := newPaymentRequestUC(pr, mr, new(MockWalletRepository), new(MockChainRepository), new(MockSmartContractRepository), new(MockTokenRepository))
		userID := uuid.New()
		mr.On("GetByUserID", context.Background(), userID).Return(nil, assert.AnError).Once()

		_, _, err := uc.ListPaymentRequests(context.Background(), userID, 10, 0)
		assert.Error(t, err)
	})
}
