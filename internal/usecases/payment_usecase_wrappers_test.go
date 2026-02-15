package usecases_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
)

func TestPaymentUsecase_ReadWrappers(t *testing.T) {
	paymentRepo := new(MockPaymentRepository)
	eventRepo := new(MockPaymentEventRepository)

	uc := usecases.NewPaymentUsecase(
		paymentRepo,
		eventRepo,
		new(MockWalletRepository),
		new(MockMerchantRepository),
		new(MockSmartContractRepository),
		new(MockChainRepository),
		new(MockTokenRepository),
		nil,
		nil,
		nil,
		new(MockUnitOfWork),
		nil,
	)

	paymentID := uuid.New()
	userID := uuid.New()
	p := &entities.Payment{ID: paymentID}
	evs := []*entities.PaymentEvent{{ID: uuid.New(), PaymentID: paymentID}}

	paymentRepo.On("GetByID", context.Background(), paymentID).Return(p, nil).Once()
	got, err := uc.GetPayment(context.Background(), paymentID)
	assert.NoError(t, err)
	assert.Equal(t, paymentID, got.ID)

	paymentRepo.On("GetByUserID", context.Background(), userID, 5, 5).Return([]*entities.Payment{p}, 1, nil).Once()
	items, total, err := uc.GetPaymentsByUser(context.Background(), userID, 2, 5)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)

	eventRepo.On("GetByPaymentID", context.Background(), paymentID).Return(evs, nil).Once()
	gotEvents, err := uc.GetPaymentEvents(context.Background(), paymentID)
	assert.NoError(t, err)
	assert.Len(t, gotEvents, 1)
}
