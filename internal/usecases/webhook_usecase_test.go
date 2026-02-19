package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
)

func TestWebhookUsecase_ProcessIndexerWebhook_InvalidJSON(t *testing.T) {
	uc := usecases.NewWebhookUsecase(
		new(MockPaymentRepository),
		new(MockPaymentEventRepository),
		new(MockPaymentRequestRepository),
		new(MockUnitOfWork),
	)

	err := uc.ProcessIndexerWebhook(context.Background(), "PAYMENT_COMPLETED", json.RawMessage("{"))
	assert.Error(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_PaymentCompletedSuccess(t *testing.T) {
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockUOW := new(MockUnitOfWork)

	uc := usecases.NewWebhookUsecase(mockPaymentRepo, mockEventRepo, mockRequestRepo, mockUOW)

	paymentID := uuid.New()
	payload := map[string]any{
		"paymentId":    paymentID.String(),
		"status":       "completed",
		"sourceTxHash": "0xabc",
	}
	raw, _ := json.Marshal(payload)

	ctx := context.Background()
	mockUOW.On("Do", ctx, mock.Anything).Return(nil).Once()
	mockUOW.On("WithLock", ctx).Return(ctx).Once()
	mockPaymentRepo.On("GetByID", ctx, paymentID).Return(&entities.Payment{ID: paymentID}, nil).Once()
	mockPaymentRepo.On("UpdateStatus", ctx, paymentID, entities.PaymentStatusCompleted).Return(nil).Once()
	mockEventRepo.On("Create", ctx, mock.AnythingOfType("*entities.PaymentEvent")).Return(nil).Once()

	err := uc.ProcessIndexerWebhook(ctx, "PAYMENT_COMPLETED", raw)
	assert.NoError(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_PaymentUpdateFails(t *testing.T) {
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockUOW := new(MockUnitOfWork)

	uc := usecases.NewWebhookUsecase(mockPaymentRepo, mockEventRepo, mockRequestRepo, mockUOW)

	paymentID := uuid.New()
	payload := map[string]any{
		"paymentId": paymentID.String(),
		"status":    "processing",
	}
	raw, _ := json.Marshal(payload)

	ctx := context.Background()
	mockUOW.On("Do", ctx, mock.Anything).Return(nil).Once()
	mockUOW.On("WithLock", ctx).Return(ctx).Once()
	mockPaymentRepo.On("GetByID", ctx, paymentID).Return(nil, errors.New("db error")).Once()

	err := uc.ProcessIndexerWebhook(ctx, "PAYMENT_EXECUTED", raw)
	assert.Error(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_RequestPaymentReceived(t *testing.T) {
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockUOW := new(MockUnitOfWork)

	uc := usecases.NewWebhookUsecase(mockPaymentRepo, mockEventRepo, mockRequestRepo, mockUOW)

	requestID := uuid.New()
	payload := map[string]any{
		"id":     requestID.String(),
		"payer":  "0xabc",
		"txHash": "0xhash",
	}
	raw, _ := json.Marshal(payload)

	mockRequestRepo.On("MarkCompleted", context.Background(), requestID, "0xhash").Return(nil).Once()

	err := uc.ProcessIndexerWebhook(context.Background(), "REQUEST_PAYMENT_RECEIVED", raw)
	assert.NoError(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_RequestPaymentReceived_InvalidJSON(t *testing.T) {
	uc := usecases.NewWebhookUsecase(
		new(MockPaymentRepository),
		new(MockPaymentEventRepository),
		new(MockPaymentRequestRepository),
		new(MockUnitOfWork),
	)

	err := uc.ProcessIndexerWebhook(context.Background(), "REQUEST_PAYMENT_RECEIVED", json.RawMessage("{"))
	assert.Error(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_RequestPaymentReceived_MarkCompletedErrorStillNil(t *testing.T) {
	mockRequestRepo := new(MockPaymentRequestRepository)
	uc := usecases.NewWebhookUsecase(
		new(MockPaymentRepository),
		new(MockPaymentEventRepository),
		mockRequestRepo,
		new(MockUnitOfWork),
	)

	requestID := uuid.New()
	payload := map[string]any{
		"id":     requestID.String(),
		"payer":  "0xabc",
		"txHash": "0xhash",
	}
	raw, _ := json.Marshal(payload)

	mockRequestRepo.On("MarkCompleted", context.Background(), requestID, "0xhash").Return(errors.New("update failed")).Once()

	err := uc.ProcessIndexerWebhook(context.Background(), "REQUEST_PAYMENT_RECEIVED", raw)
	assert.NoError(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_PaymentCompleted_UOWDoError(t *testing.T) {
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockUOW := new(MockUnitOfWork)
	uc := usecases.NewWebhookUsecase(mockPaymentRepo, mockEventRepo, mockRequestRepo, mockUOW)

	paymentID := uuid.New()
	payload := map[string]any{
		"paymentId": paymentID.String(),
		"status":    "completed",
	}
	raw, _ := json.Marshal(payload)
	ctx := context.Background()

	mockUOW.On("Do", ctx, mock.Anything).Return(errors.New("tx begin failed")).Once()
	err := uc.ProcessIndexerWebhook(ctx, "PAYMENT_COMPLETED", raw)
	assert.Error(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_UnhandledEvent(t *testing.T) {
	uc := usecases.NewWebhookUsecase(
		new(MockPaymentRepository),
		new(MockPaymentEventRepository),
		new(MockPaymentRequestRepository),
		new(MockUnitOfWork),
	)

	err := uc.ProcessIndexerWebhook(context.Background(), "SOME_UNKNOWN_EVENT", json.RawMessage(`{"x":1}`))
	assert.NoError(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_PaymentUpdateStatusError(t *testing.T) {
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockUOW := new(MockUnitOfWork)

	uc := usecases.NewWebhookUsecase(mockPaymentRepo, mockEventRepo, mockRequestRepo, mockUOW)

	paymentID := uuid.New()
	payload := map[string]any{
		"paymentId": paymentID.String(),
		"status":    "processing",
	}
	raw, _ := json.Marshal(payload)
	ctx := context.Background()

	mockUOW.On("Do", ctx, mock.Anything).Return(nil).Once()
	mockUOW.On("WithLock", ctx).Return(ctx).Once()
	mockPaymentRepo.On("GetByID", ctx, paymentID).Return(&entities.Payment{ID: paymentID}, nil).Once()
	mockPaymentRepo.On("UpdateStatus", ctx, paymentID, entities.PaymentStatusProcessing).Return(errors.New("update status failed")).Once()

	err := uc.ProcessIndexerWebhook(ctx, "PAYMENT_EXECUTED", raw)
	assert.EqualError(t, err, "update status failed")
}

func TestWebhookUsecase_ProcessIndexerWebhook_PaymentRequestCreatedBranch(t *testing.T) {
	uc := usecases.NewWebhookUsecase(
		new(MockPaymentRepository),
		new(MockPaymentEventRepository),
		new(MockPaymentRequestRepository),
		new(MockUnitOfWork),
	)

	err := uc.ProcessIndexerWebhook(context.Background(), "PAYMENT_REQUEST_CREATED", json.RawMessage(`{"requestId":"abc"}`))
	assert.NoError(t, err)
}

func TestWebhookUsecase_ProcessIndexerWebhook_PaymentFailed(t *testing.T) {
	mockPaymentRepo := new(MockPaymentRepository)
	mockEventRepo := new(MockPaymentEventRepository)
	mockRequestRepo := new(MockPaymentRequestRepository)
	mockUOW := new(MockUnitOfWork)

	uc := usecases.NewWebhookUsecase(mockPaymentRepo, mockEventRepo, mockRequestRepo, mockUOW)

	paymentID := uuid.New()
	failurePayload := map[string]any{
		"paymentId":    paymentID.String(),
		"status":       "FAILED",
		"reason":       "execution reverted",
		"revertData":   "0x08c379a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001bc5d2460186f7233303a3a416363657373436f6e74726f6c3a206163636f756e74203078313233206973206d697373696e6720726f6c652030783435360000000000", // "Error(string)" selector + "Some Error" (simulated)
		"sourceTxHash": "0xfail",
	}
	// Note: The revertData above is just a filler hex for now,
	// specific decoding logic verification is covered by unit tests for decodeRouteErrorData.
	// We just want to ensure it passes the Update call with correct fields.

	raw, _ := json.Marshal(failurePayload)
	ctx := context.Background()

	mockUOW.On("Do", ctx, mock.Anything).Return(nil).Once()
	mockUOW.On("WithLock", ctx).Return(ctx).Once()

	existingPayment := &entities.Payment{ID: paymentID, Status: entities.PaymentStatusPending}
	mockPaymentRepo.On("GetByID", ctx, paymentID).Return(existingPayment, nil).Once()

	// Expect Update to be called with modified payment
	mockPaymentRepo.On("Update", ctx, mock.MatchedBy(func(p *entities.Payment) bool {
		return p.ID == paymentID &&
			p.Status == entities.PaymentStatusFailed &&
			p.FailureReason.Valid &&
			p.RevertData.Valid &&
			p.RevertData.String == failurePayload["revertData"]
	})).Return(nil).Once()

	mockEventRepo.On("Create", ctx, mock.MatchedBy(func(e *entities.PaymentEvent) bool {
		return e.PaymentID == paymentID && e.EventType == "PAYMENT_FAILED" && e.TxHash == "0xfail"
	})).Return(nil).Once()

	err := uc.ProcessIndexerWebhook(ctx, "PAYMENT_FAILED", raw)
	assert.NoError(t, err)
}
