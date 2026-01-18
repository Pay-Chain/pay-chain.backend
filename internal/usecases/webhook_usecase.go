package usecases

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/domain/repositories"
	infra_repos "pay-chain.backend/internal/infrastructure/repositories"
)

// WebhookUsecase handles incoming notifications from the indexer
type WebhookUsecase struct {
	paymentRepo        repositories.PaymentRepository
	paymentEventRepo   *infra_repos.PaymentEventRepository
	paymentRequestRepo *infra_repos.PaymentRequestRepositoryImpl
}

// NewWebhookUsecase creates a new webhook usecase
func NewWebhookUsecase(
	paymentRepo repositories.PaymentRepository,
	paymentEventRepo *infra_repos.PaymentEventRepository,
	paymentRequestRepo *infra_repos.PaymentRequestRepositoryImpl,
) *WebhookUsecase {
	return &WebhookUsecase{
		paymentRepo:        paymentRepo,
		paymentEventRepo:   paymentEventRepo,
		paymentRequestRepo: paymentRequestRepo,
	}
}

// Map indexer event types to backend status
func mapStatus(indexerStatus string) entities.PaymentStatus {
	switch indexerStatus {
	case "pending":
		return entities.PaymentStatusPending
	case "processing":
		return entities.PaymentStatusProcessing
	case "completed":
		return entities.PaymentStatusCompleted
	case "failed":
		return entities.PaymentStatusFailed
	case "refunded":
		return entities.PaymentStatusRefunded
	default:
		return entities.PaymentStatusPending
	}
}

// ProcessIndexerWebhook processes a webhook payload from the indexer
func (u *WebhookUsecase) ProcessIndexerWebhook(ctx context.Context, eventType string, data json.RawMessage) error {
	log.Printf("Processing indexer event: %s", eventType)

	switch eventType {
	case "PAYMENT_CREATED", "PAYMENT_EXECUTED", "PAYMENT_COMPLETED", "PAYMENT_REFUNDED":
		var paymentData struct {
			PaymentId    string `json:"paymentId"`
			Status       string `json:"status"`
			SourceTxHash string `json:"sourceTxHash"`
			DestTxHash   string `json:"destTxHash"`
		}
		if err := json.Unmarshal(data, &paymentData); err != nil {
			return err
		}

		paymentUUID, _ := uuid.Parse(paymentData.PaymentId)
		status := mapStatus(paymentData.Status)

		// Update payment status
		err := u.paymentRepo.UpdateStatus(ctx, paymentUUID, status)
		if err != nil {
			log.Printf("Error updating payment status: %v", err)
		}

		// Create event
		_ = u.paymentEventRepo.Create(ctx, &entities.PaymentEvent{
			PaymentID: paymentUUID,
			EventType: eventType,
			Chain:     "indexer", // Marker for indexer-triggered update
			TxHash:    paymentData.SourceTxHash,
		})

	case "PAYMENT_REQUEST_CREATED":
		// No action needed if backend already created it, 
		// but we could sync if it originated from elsewhere
		log.Printf("Payment request created on-chain: %s", data)

	case "REQUEST_PAYMENT_RECEIVED":
		var requestData struct {
			Id     string `json:"id"` // requestId
			Payer  string `json:"payer"`
			TxHash string `json:"txHash"`
		}
		if err := json.Unmarshal(data, &requestData); err != nil {
			return err
		}

		requestUUID, _ := uuid.Parse(requestData.Id)
		err := u.paymentRequestRepo.MarkCompleted(ctx, requestUUID, requestData.TxHash)
		if err != nil {
			log.Printf("Error marking payment request as completed: %v", err)
		}

	default:
		log.Printf("Unhandled event type: %s", eventType)
	}

	return nil
}
