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
