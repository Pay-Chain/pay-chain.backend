package usecases

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/domain/repositories"
)

// WebhookUsecase handles incoming notifications from the indexer
type WebhookUsecase struct {
	paymentRepo        repositories.PaymentRepository
	paymentEventRepo   repositories.PaymentEventRepository
	paymentRequestRepo repositories.PaymentRequestRepository
	uow                repositories.UnitOfWork
}

// NewWebhookUsecase creates a new webhook usecase
func NewWebhookUsecase(
	paymentRepo repositories.PaymentRepository,
	paymentEventRepo repositories.PaymentEventRepository,
	paymentRequestRepo repositories.PaymentRequestRepository,
	uow repositories.UnitOfWork,
) *WebhookUsecase {
	return &WebhookUsecase{
		paymentRepo:        paymentRepo,
		paymentEventRepo:   paymentEventRepo,
		paymentRequestRepo: paymentRequestRepo,
		uow:                uow,
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
		newStatus := mapStatus(paymentData.Status)

		// Update payment status with locking to prevent race conditions
		err := u.uow.Do(ctx, func(txCtx context.Context) error {
			lockCtx := u.uow.WithLock(txCtx)

			// 1. Get current Payment with Lock
			// Note: We need GetByID on repository. Assuming it's available.
			_, err := u.paymentRepo.GetByID(lockCtx, paymentUUID)
			if err != nil {
				return err // Or ignore if not found?
			}

			// 2. Validate Transition (Optional state machine check can be added here)
			// For now, we trust the indexer but having the lock ensures we serialize updates.

			// 3. Update status
			if err := u.paymentRepo.UpdateStatus(lockCtx, paymentUUID, newStatus); err != nil {
				return err
			}

			// 4. Create event
			return u.paymentEventRepo.Create(lockCtx, &entities.PaymentEvent{
				PaymentID: paymentUUID,
				EventType: entities.PaymentEventType(eventType),
				TxHash:    paymentData.SourceTxHash,
			})
		})

		if err != nil {
			log.Printf("Error processing payment update: %v", err)
			return err
		}

	case "PAYMENT_FAILED":
		var failureData struct {
			PaymentId    string `json:"paymentId"`
			Status       string `json:"status"`
			Reason       string `json:"reason"`
			RevertData   string `json:"revertData"`
			SourceTxHash string `json:"sourceTxHash"`
		}
		if err := json.Unmarshal(data, &failureData); err != nil {
			return err
		}

		paymentUUID, _ := uuid.Parse(failureData.PaymentId)
		newStatus := mapStatus(failureData.Status)
		if newStatus == entities.PaymentStatusPending { // Fallback if status not explicitly failed in payload
			newStatus = entities.PaymentStatusFailed
		}

		// Try to decode revert data if present and reason is generic or empty
		decodedReason := failureData.Reason
		if failureData.RevertData != "" {
			// decodeRouteErrorData is available in the same package
			if revertBytes, err := hex.DecodeString(strings.TrimPrefix(failureData.RevertData, "0x")); err == nil {
				decoded := decodeRouteErrorData(revertBytes)
				if decoded.Message != "" {
					if decodedReason != "" {
						decodedReason = fmt.Sprintf("%s (Revert: %s)", decodedReason, decoded.Message)
					} else {
						decodedReason = decoded.Message
					}
				}
			}
		}

		err := u.uow.Do(ctx, func(txCtx context.Context) error {
			lockCtx := u.uow.WithLock(txCtx)

			// 1. Get current Payment
			payment, err := u.paymentRepo.GetByID(lockCtx, paymentUUID)
			if err != nil {
				return err
			}

			// 2. Update status and failure fields
			payment.Status = newStatus
			payment.FailureReason.String = decodedReason
			payment.FailureReason.Valid = decodedReason != ""
			payment.RevertData.String = failureData.RevertData
			payment.RevertData.Valid = failureData.RevertData != ""

			if err := u.paymentRepo.Update(lockCtx, payment); err != nil {
				return err
			}

			// 3. Create event
			return u.paymentEventRepo.Create(lockCtx, &entities.PaymentEvent{
				PaymentID: paymentUUID,
				EventType: entities.PaymentEventType(eventType),
				TxHash:    failureData.SourceTxHash,
				Metadata:  string(data), // Store full failure payload in metadata
			})
		})

		if err != nil {
			log.Printf("Error processing payment failure: %v", err)
			return err
		}

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
