package usecases

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/infrastructure/metrics"
)

// WebhookUsecase handles incoming notifications from the indexer
type WebhookUsecase struct {
	paymentRepo        repositories.PaymentRepository
	paymentEventRepo   repositories.PaymentEventRepository
	paymentRequestRepo repositories.PaymentRequestRepository
	merchantRepo       repositories.MerchantRepository
	webhookLogRepo     repositories.WebhookLogRepository
	dispatcher         *WebhookDispatcher
	uow                repositories.UnitOfWork
}

// NewWebhookUsecase creates a new webhook usecase
func NewWebhookUsecase(
	paymentRepo repositories.PaymentRepository,
	paymentEventRepo repositories.PaymentEventRepository,
	paymentRequestRepo repositories.PaymentRequestRepository,
	merchantRepo repositories.MerchantRepository,
	webhookLogRepo repositories.WebhookLogRepository,
	dispatcher *WebhookDispatcher,
	uow repositories.UnitOfWork,
) *WebhookUsecase {
	return &WebhookUsecase{
		paymentRepo:        paymentRepo,
		paymentEventRepo:   paymentEventRepo,
		paymentRequestRepo: paymentRequestRepo,
		merchantRepo:       merchantRepo,
		webhookLogRepo:     webhookLogRepo,
		dispatcher:         dispatcher,
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
			_, err := u.paymentRepo.GetByID(lockCtx, paymentUUID)
			if err != nil {
				return err
			}

			// 3. Update status
			if err := u.paymentRepo.UpdateStatus(lockCtx, paymentUUID, newStatus); err != nil {
				return err
			}

			// 4. Create event
			return u.paymentEventRepo.Create(lockCtx, &entities.PaymentEvent{
				PaymentID: paymentUUID,
				EventType: entities.PaymentEventType(eventType),
				TxHash:    paymentData.SourceTxHash,
				Metadata:  string(data),
			})
		})

		if err != nil {
			log.Printf("Error processing payment update: %v", err)
			return err
		}

		// Trigger Webhook if terminal state
		if newStatus == entities.PaymentStatusCompleted || newStatus == entities.PaymentStatusRefunded {
			_ = u.enqueueWebhookDelivery(ctx, paymentUUID, string(newStatus), data)

			// Record Settlement Latency
			if newStatus == entities.PaymentStatusCompleted {
				if payment, err := u.paymentRepo.GetByID(ctx, paymentUUID); err == nil {
					duration := time.Since(payment.CreatedAt).Seconds()
					metrics.RecordSettlementLatency(payment.DestChainID.String(), duration)
				}
			}
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
		if newStatus == entities.PaymentStatusPending {
			newStatus = entities.PaymentStatusFailed
		}

		decodedReason := failureData.Reason
		if failureData.RevertData != "" {
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
			payment, err := u.paymentRepo.GetByID(lockCtx, paymentUUID)
			if err != nil {
				return err
			}
			payment.Status = newStatus
			payment.FailureReason.String = decodedReason
			payment.FailureReason.Valid = decodedReason != ""
			payment.RevertData.String = failureData.RevertData
			payment.RevertData.Valid = failureData.RevertData != ""

			if err := u.paymentRepo.Update(lockCtx, payment); err != nil {
				return err
			}

			return u.paymentEventRepo.Create(lockCtx, &entities.PaymentEvent{
				PaymentID: paymentUUID,
				EventType: entities.PaymentEventType(eventType),
				TxHash:    failureData.SourceTxHash,
				Metadata:  string(data),
			})
		})

		if err != nil {
			log.Printf("Error processing payment failure: %v", err)
			return err
		}

		// Trigger Webhook for failure
		_ = u.enqueueWebhookDelivery(ctx, paymentUUID, string(entities.PaymentStatusFailed), data)

	case "PAYMENT_REQUEST_CREATED":
		log.Printf("Payment request created on-chain: %s", data)

	case "REQUEST_PAYMENT_RECEIVED":
		var requestData struct {
			Id     string `json:"id"`
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

func (u *WebhookUsecase) enqueueWebhookDelivery(ctx context.Context, paymentID uuid.UUID, eventType string, data json.RawMessage) error {
	payment, err := u.paymentRepo.GetByID(ctx, paymentID)
	if err != nil || payment.MerchantID == nil {
		return nil
	}

	delivery := &entities.WebhookDelivery{
		ID:             uuid.New(),
		MerchantID:     *payment.MerchantID,
		PaymentID:      paymentID,
		EventType:      eventType,
		Payload:        null.JSONFrom(data),
		DeliveryStatus: entities.WebhookDeliveryStatusPending,
		RetryCount:     0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := u.webhookLogRepo.Create(ctx, delivery); err != nil {
		log.Printf("[WebhookUsecase] Failed to create delivery log: %v", err)
		return err
	}

	log.Printf("[WebhookUsecase] Enqueued webhook delivery %s for merchant %s", delivery.ID, delivery.MerchantID)
	return nil
}

// ManualRetry triggers a manual webhook delivery attempt
func (u *WebhookUsecase) ManualRetry(ctx context.Context, deliveryID uuid.UUID) error {
	delivery, err := u.webhookLogRepo.GetByID(ctx, deliveryID)
	if err != nil {
		return fmt.Errorf("webhook delivery not found: %w", err)
	}

	log.Printf("[WebhookUsecase] Manually retrying webhook delivery %s", deliveryID)
	return u.dispatcher.Dispatch(ctx, delivery)
}
