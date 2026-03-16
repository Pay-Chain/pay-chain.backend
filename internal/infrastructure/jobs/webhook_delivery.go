package jobs

import (
	"context"
	"log"
	"time"

	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/usecases"
)

type WebhookDeliveryJob struct {
	webhookLogRepo repositories.WebhookLogRepository
	dispatcher     *usecases.WebhookDispatcher
}

func NewWebhookDeliveryJob(
	webhookLogRepo repositories.WebhookLogRepository,
	dispatcher *usecases.WebhookDispatcher,
) *WebhookDeliveryJob {
	return &WebhookDeliveryJob{
		webhookLogRepo: webhookLogRepo,
		dispatcher:     dispatcher,
	}
}

func (j *WebhookDeliveryJob) Run(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.Println("[WebhookDeliveryJob] Started")

	for {
		select {
		case <-ctx.Done():
			log.Println("[WebhookDeliveryJob] Stopping")
			return
		case <-ticker.C:
			j.processPendingDeliveries(ctx)
		}
	}
}

func (j *WebhookDeliveryJob) processPendingDeliveries(ctx context.Context) {
	// 1. Fetch pending and retrying deliveries
	deliveries, err := j.webhookLogRepo.GetPendingAttempts(ctx, 20) // Batch of 20
	if err != nil {
		log.Printf("[WebhookDeliveryJob] Error fetching pending deliveries: %v", err)
		return
	}

	if len(deliveries) == 0 {
		return
	}

	log.Printf("[WebhookDeliveryJob] Processing %d deliveries", len(deliveries))

	for i := range deliveries {
		delivery := &deliveries[i]
		// Check for context cancellation during processing
		if ctx.Err() != nil {
			return
		}

		// Check retry limit (e.g., 5 attempts)
		if delivery.RetryCount >= 5 {
			delivery.DeliveryStatus = entities.WebhookDeliveryStatusFailed
			delivery.ResponseBody = "Max retry attempts reached"
			_ = j.webhookLogRepo.Update(ctx, delivery)
			continue
		}

		// Exponential Backoff check (simple version)
		if delivery.DeliveryStatus == entities.WebhookDeliveryStatusRetrying {
			backoff := time.Duration(1<<uint(delivery.RetryCount)) * time.Minute
			if time.Since(*delivery.LastAttemptAt) < backoff {
				continue
			}
		}

		err := j.dispatcher.Dispatch(ctx, delivery)
		if err != nil {
			log.Printf("[WebhookDeliveryJob] Error dispatching delivery %s: %v", delivery.ID, err)
		}
	}
}
