package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
)

type WebhookDeliveryStatus string

const (
	WebhookDeliveryStatusPending    WebhookDeliveryStatus = "pending"
	WebhookDeliveryStatusDelivering WebhookDeliveryStatus = "delivering"
	WebhookDeliveryStatusDelivered  WebhookDeliveryStatus = "delivered"
	WebhookDeliveryStatusRetrying   WebhookDeliveryStatus = "retrying"
	WebhookDeliveryStatusFailed     WebhookDeliveryStatus = "failed"
	WebhookDeliveryStatusDropped    WebhookDeliveryStatus = "dropped"
)

type WebhookDelivery struct {
	ID             uuid.UUID             `json:"id"`
	MerchantID     uuid.UUID             `json:"merchantId"`
	PaymentID      uuid.UUID             `json:"paymentId"`
	EventType      string                `json:"eventType"`
	Payload        null.JSON             `json:"payload"`
	DeliveryStatus WebhookDeliveryStatus `json:"deliveryStatus"`
	HttpStatus     int                   `json:"httpStatus"`
	ResponseBody   string                `json:"responseBody,omitempty"`
	RetryCount     int                   `json:"retryCount"`
	NextRetryAt    *time.Time            `json:"nextRetryAt,omitempty"`
	LastAttemptAt  *time.Time            `json:"lastAttemptAt,omitempty"`
	CreatedAt      time.Time             `json:"createdAt"`
	UpdatedAt      time.Time             `json:"updatedAt"`
}
