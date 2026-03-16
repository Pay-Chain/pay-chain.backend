package usecases_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/usecases"
)

func TestWebhookUsecase_ProcessIndexerWebhook_InvalidJSON(t *testing.T) {
	uc := usecases.NewWebhookUsecase(
		new(MockPaymentRepository),
		new(MockPaymentEventRepository),
		new(MockPaymentRequestRepository),
		new(MockMerchantRepository),
		new(MockWebhookLogRepository),
		nil, // WebhookDispatcher
		new(MockUnitOfWork),
	)

	err := uc.ProcessIndexerWebhook(context.Background(), "PAYMENT_COMPLETED", json.RawMessage("{"))
	assert.Error(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_PaymentCompletedSuccess(t *testing.T) {
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockMerchantRepo := new(MockMerchantRepository)
	mockWebhookRepo := new(MockWebhookLogRepository)
	mockUOW := new(MockUnitOfWork)

	uc := usecases.NewWebhookUsecase(
		mockPaymentRepo,
		mockEventRepo,
		mockRequestRepo,
		mockMerchantRepo,
		mockWebhookRepo,
		nil, // WebhookDispatcher
		mockUOW,
	)

	paymentID := uuid.New()
	merchantID := uuid.New()
	payload := map[string]interface{}{
		"paymentId":    paymentID.String(),
		"status":       "completed",
		"sourceTxHash": "0x123",
		"destTxHash":   "0x456",
	}
	data, _ := json.Marshal(payload)
	ctx := context.Background()

	mockUOW.On("Do", ctx, mock.Anything).Return(nil).Once()
	mockUOW.On("WithLock", ctx).Return(ctx).Once()
	
	mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(&entities.Payment{ID: paymentID, MerchantID: &merchantID}, nil)
	mockPaymentRepo.On("UpdateStatus", mock.Anything, paymentID, entities.PaymentStatusCompleted).Return(nil)
	mockEventRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockWebhookRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := uc.ProcessIndexerWebhook(ctx, "PAYMENT_COMPLETED", data)
	assert.NoError(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_PaymentFailed(t *testing.T) {
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockMerchantRepo := new(MockMerchantRepository)
	mockWebhookRepo := new(MockWebhookLogRepository)
	mockUOW := new(MockUnitOfWork)

	uc := usecases.NewWebhookUsecase(
		mockPaymentRepo,
		mockEventRepo,
		mockRequestRepo,
		mockMerchantRepo,
		mockWebhookRepo,
		nil, // WebhookDispatcher
		mockUOW,
	)

	paymentID := uuid.New()
	merchantID := uuid.New()
	failurePayload := map[string]any{
		"paymentId":    paymentID.String(),
		"status":       "FAILED",
		"reason":       "execution reverted",
		"revertData":   "0x08c379a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001bc5d2460186f7233303a3a416363657373436f6e74726f6c3a206163636f756e74203078313233206973206d697373696e6720726f6c652030783435360000000000",
		"sourceTxHash": "0xfail",
	}
	raw, _ := json.Marshal(failurePayload)
	ctx := context.Background()

	mockUOW.On("Do", ctx, mock.Anything).Return(nil).Once()
	mockUOW.On("WithLock", ctx).Return(ctx).Once()

	mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(&entities.Payment{ID: paymentID, MerchantID: &merchantID}, nil)
	mockPaymentRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockEventRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockWebhookRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := uc.ProcessIndexerWebhook(ctx, "PAYMENT_FAILED", raw)
	assert.NoError(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_RequestPaymentReceived(t *testing.T) {
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockMerchantRepo := new(MockMerchantRepository)
	mockWebhookRepo := new(MockWebhookLogRepository)
	mockUOW := new(MockUnitOfWork)

	uc := usecases.NewWebhookUsecase(
		mockPaymentRepo,
		mockEventRepo,
		mockRequestRepo,
		mockMerchantRepo,
		mockWebhookRepo,
		nil, // WebhookDispatcher
		mockUOW,
	)

	requestID := uuid.New()
	payload := map[string]any{
		"id":     requestID.String(),
		"payer":  "0xPayer",
		"txHash": "0xTx",
	}
	raw, _ := json.Marshal(payload)

	mockRequestRepo.On("MarkCompleted", mock.Anything, requestID, "0xTx").Return(nil)

	err := uc.ProcessIndexerWebhook(context.Background(), "REQUEST_PAYMENT_RECEIVED", raw)
	assert.NoError(t, err)
}
