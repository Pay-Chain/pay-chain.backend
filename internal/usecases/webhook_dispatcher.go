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

const (
	webhookMaxRetries = 10
)

var webhookRetrySchedule = []time.Duration{
	1 * time.Minute,
	2 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
	1 * time.Hour,
	2 * time.Hour,
	4 * time.Hour,
	8 * time.Hour,
}

func WebhookMaxRetries() int {
	return webhookMaxRetries
}

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
	signaturePayload := timestamp + "." + string(payloadBytes)
	signature := d.hmacService.Generate(signaturePayload, merchant.WebhookSecret)
	legacySignature := d.hmacService.Generate(timestamp+string(payloadBytes), merchant.WebhookSecret)

	// 4. Send Request
	req, err := http.NewRequestWithContext(ctx, "POST", merchant.CallbackURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Signature-Legacy", legacySignature)
	req.Header.Set("X-Webhook-Timestamp", timestamp)
	req.Header.Set("X-Webhook-Event", delivery.EventType)
	req.Header.Set("X-Webhook-Delivery-Id", delivery.ID.String())
	req.Header.Set("User-Agent", "PaymentKita-Webhook-Dispatcher/1.0")

	now := time.Now()
	delivery.LastAttemptAt = &now
	delivery.NextRetryAt = nil
	delivery.DeliveryStatus = entities.WebhookDeliveryStatusDelivering
	_ = d.webhookLogRepo.Update(ctx, delivery)

	resp, err := d.httpClient.Do(req)
	duration := time.Since(start).Seconds()
	if err != nil {
		delivery.DeliveryStatus = entities.WebhookDeliveryStatusRetrying
		delivery.RetryCount++
		setNextRetryAt(delivery)
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
		delivery.NextRetryAt = nil
		metrics.RecordWebhookDelivery(delivery.MerchantID.String(), delivery.EventType, "success", duration)
	} else {
		delivery.DeliveryStatus = entities.WebhookDeliveryStatusRetrying
		delivery.RetryCount++
		setNextRetryAt(delivery)
		metrics.RecordWebhookDelivery(delivery.MerchantID.String(), delivery.EventType, fmt.Sprintf("status_%d", resp.StatusCode), duration)
	}

	return d.webhookLogRepo.Update(ctx, delivery)
}

func setNextRetryAt(delivery *entities.WebhookDelivery) {
	if delivery == nil {
		return
	}
	attempt := delivery.RetryCount
	if attempt <= 0 {
		attempt = 1
	}
	index := attempt - 1
	if index >= len(webhookRetrySchedule) {
		index = len(webhookRetrySchedule) - 1
	}
	next := time.Now().Add(webhookRetrySchedule[index])
	delivery.NextRetryAt = &next
	if delivery.RetryCount >= webhookMaxRetries {
		delivery.DeliveryStatus = entities.WebhookDeliveryStatusFailed
		delivery.ResponseBody = "Max retry attempts reached"
		delivery.NextRetryAt = nil
	}
}
