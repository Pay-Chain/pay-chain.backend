package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	WebhookDeliveryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pk_webhook_delivery_total",
		Help: "Total number of webhook deliveries",
	}, []string{"merchant_id", "event_type", "status"})

	WebhookDeliveryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pk_webhook_delivery_duration_seconds",
		Help:    "Webhook delivery latency in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"merchant_id", "event_type"})

	WebhookRetryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pk_webhook_retry_total",
		Help: "Total number of webhook retries",
	}, []string{"merchant_id", "event_type"})

	CreatePaymentCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pk_partner_create_payment_requests_total",
		Help: "The total number of payment creation attempts",
	}, []string{"merchant_id", "status"})

	SettlementLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pk_settlement_duration_seconds",
		Help:    "Time from session creation to COMPLETED status",
		Buckets: prometheus.DefBuckets,
	}, []string{"chain_id"})

	JWEDecryptionErrorCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pk_jwe_decryption_errors_total",
		Help: "The total number of JWE decryption errors",
	}, []string{"reason"})

	IndexerLagGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pk_indexer_lag_seconds",
		Help: "The total lag of the indexer in seconds",
	}, []string{"chain_id"})

	LegacyEndpointUsageTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pk_legacy_endpoint_usage_total",
		Help: "Total number of legacy endpoint hits",
	}, []string{"endpoint_family", "merchant_id"})
)

func RecordSessionCreated(merchID string, err error) {
	status := "success"
	if err != nil {
		status = "failure"
	}
	CreatePaymentCounter.WithLabelValues(merchID, status).Inc()
}

func RecordJWEDecryptionError(reason string) {
	JWEDecryptionErrorCounter.WithLabelValues(reason).Inc()
}

func RecordIndexerLag(chainID string, lag float64) {
	IndexerLagGauge.WithLabelValues(chainID).Set(lag)
}

func RecordSettlementLatency(chainID string, duration float64) {
	SettlementLatency.WithLabelValues(chainID).Observe(duration)
}

func RecordWebhookDelivery(merchantID string, eventType string, status string, duration float64) {
	WebhookDeliveryTotal.WithLabelValues(merchantID, eventType, status).Inc()
	WebhookDeliveryDuration.WithLabelValues(merchantID, eventType).Observe(duration)
}

func RecordWebhookRetry(merchantID string, eventType string) {
	WebhookRetryTotal.WithLabelValues(merchantID, eventType).Inc()
}

func RecordLegacyEndpointUsage(endpointFamily string, merchantID string) {
	if merchantID == "" {
		merchantID = "unknown"
	}
	LegacyEndpointUsageTotal.WithLabelValues(endpointFamily, merchantID).Inc()
}
