package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/domain/services"
	"payment-kita.backend/internal/infrastructure/metrics"
)

type WebhookDispatcher struct {
	webhookLogRepo repositories.WebhookLogRepository
	merchantRepo   repositories.MerchantRepository
	hmacService    services.HMACService
	httpClient     *http.Client
}

func NewWebhookDispatcher(
	webhookLogRepo repositories.WebhookLogRepository,
	merchantRepo repositories.MerchantRepository,
	hmacService services.HMACService,
) *WebhookDispatcher {
	return &WebhookDispatcher{
		webhookLogRepo: webhookLogRepo,
		merchantRepo:   merchantRepo,
		hmacService:    hmacService,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *WebhookDispatcher) Dispatch(ctx context.Context, delivery *entities.WebhookDelivery) error {
	start := time.Now()
	// 1. Get Merchant for Secret
	merchant, err := d.merchantRepo.GetByID(ctx, delivery.MerchantID)
	if err != nil {
		return fmt.Errorf("failed to get merchant: %w", err)
	}

	if !merchant.WebhookIsActive || merchant.CallbackURL == "" {
		delivery.DeliveryStatus = entities.WebhookDeliveryStatusDropped
		delivery.ResponseBody = "Webhook inactive or callback URL missing"
		return d.webhookLogRepo.Update(ctx, delivery)
	}

	// 2. Prepare Payload
	payloadBytes, err := json.Marshal(delivery.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 3. Generate HMAC Signature
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	// Format: timestamp + payload
	signature := d.hmacService.Generate(timestamp+string(payloadBytes), merchant.WebhookSecret)

	// 4. Send Request
	req, err := http.NewRequestWithContext(ctx, "POST", merchant.CallbackURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Timestamp", timestamp)
	req.Header.Set("User-Agent", "PaymentKita-Webhook-Dispatcher/1.0")

	now := time.Now()
	delivery.LastAttemptAt = &now
	delivery.DeliveryStatus = entities.WebhookDeliveryStatusDelivering
	_ = d.webhookLogRepo.Update(ctx, delivery)

	resp, err := d.httpClient.Do(req)
	duration := time.Since(start).Seconds()
	if err != nil {
		delivery.DeliveryStatus = entities.WebhookDeliveryStatusRetrying
		delivery.RetryCount++
		delivery.ResponseBody = err.Error()
		metrics.RecordWebhookDelivery(delivery.MerchantID.String(), delivery.EventType, "error", duration)
		return d.webhookLogRepo.Update(ctx, delivery)
	}
	defer resp.Body.Close()

	// 5. Update Status
	delivery.HttpStatus = resp.StatusCode
	body, _ := io.ReadAll(resp.Body)
	delivery.ResponseBody = string(body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		delivery.DeliveryStatus = entities.WebhookDeliveryStatusDelivered
		metrics.RecordWebhookDelivery(delivery.MerchantID.String(), delivery.EventType, "success", duration)
	} else {
		delivery.DeliveryStatus = entities.WebhookDeliveryStatusRetrying
		delivery.RetryCount++
		metrics.RecordWebhookDelivery(delivery.MerchantID.String(), delivery.EventType, fmt.Sprintf("status_%d", resp.StatusCode), duration)
	}

	return d.webhookLogRepo.Update(ctx, delivery)
}

