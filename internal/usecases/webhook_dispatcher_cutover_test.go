package usecases

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"payment-kita.backend/internal/domain/entities"
	domainrepos "payment-kita.backend/internal/domain/repositories"
	servicesimpl "payment-kita.backend/internal/infrastructure/services"
)

type fakeMerchantRepo struct {
	merchant *entities.Merchant
}

func (f *fakeMerchantRepo) Create(ctx context.Context, merchant *entities.Merchant) error { return nil }
func (f *fakeMerchantRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Merchant, error) {
	return f.merchant, nil
}
func (f *fakeMerchantRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.Merchant, error) {
	return nil, nil
}
func (f *fakeMerchantRepo) Update(ctx context.Context, merchant *entities.Merchant) error { return nil }
func (f *fakeMerchantRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.MerchantStatus) error {
	return nil
}
func (f *fakeMerchantRepo) SoftDelete(ctx context.Context, id uuid.UUID) error     { return nil }
func (f *fakeMerchantRepo) List(ctx context.Context) ([]*entities.Merchant, error) { return nil, nil }

type fakeWebhookLogRepo struct {
	updated []*entities.WebhookDelivery
}

func (f *fakeWebhookLogRepo) Create(ctx context.Context, log *entities.WebhookDelivery) error {
	return nil
}
func (f *fakeWebhookLogRepo) Update(ctx context.Context, log *entities.WebhookDelivery) error {
	cp := *log
	f.updated = append(f.updated, &cp)
	return nil
}
func (f *fakeWebhookLogRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.WebhookDelivery, error) {
	return nil, nil
}
func (f *fakeWebhookLogRepo) GetPendingAttempts(ctx context.Context, limit int) ([]entities.WebhookDelivery, error) {
	return nil, nil
}
func (f *fakeWebhookLogRepo) GetMerchantHistory(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]entities.WebhookDelivery, int64, error) {
	return nil, 0, nil
}
func (f *fakeWebhookLogRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, httpCode int, body string) error {
	return nil
}

type captureRoundTripper struct {
	lastRequest *http.Request
	lastBody    string
	statusCode  int
}

func (c *captureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	c.lastRequest = req
	body, _ := io.ReadAll(req.Body)
	c.lastBody = string(body)
	return &http.Response{
		StatusCode: c.statusCode,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
	}, nil
}

var (
	_ domainrepos.MerchantRepository   = (*fakeMerchantRepo)(nil)
	_ domainrepos.WebhookLogRepository = (*fakeWebhookLogRepo)(nil)
)

func TestWebhookDispatcher_CutoverHeadersAndLegacyCompatibility(t *testing.T) {
	merchantID := uuid.New()
	merchant := &entities.Merchant{
		ID:              merchantID,
		CallbackURL:     "https://merchant.example/webhook",
		WebhookSecret:   "super-secret",
		WebhookIsActive: true,
	}
	webhookRepo := &fakeWebhookLogRepo{}
	dispatcher := NewWebhookDispatcher(webhookRepo, &fakeMerchantRepo{merchant: merchant}, servicesimpl.NewHMACService())
	transport := &captureRoundTripper{statusCode: http.StatusOK}
	dispatcher.httpClient = &http.Client{Transport: transport}

	delivery := &entities.WebhookDelivery{
		ID:             uuid.New(),
		MerchantID:     merchantID,
		PaymentID:      uuid.New(),
		EventType:      "COMPLETED",
		Payload:        null.JSONFrom([]byte(`{"status":"COMPLETED"}`)),
		DeliveryStatus: entities.WebhookDeliveryStatusPending,
		CreatedAt:      time.Now(),
	}

	if err := dispatcher.Dispatch(context.Background(), delivery); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}

	if transport.lastRequest == nil {
		t.Fatalf("expected outbound request")
	}
	if got := transport.lastRequest.Header.Get("X-Webhook-Event"); got != "COMPLETED" {
		t.Fatalf("unexpected event header: %s", got)
	}
	if got := transport.lastRequest.Header.Get("X-Webhook-Delivery-Id"); got != delivery.ID.String() {
		t.Fatalf("unexpected delivery id header: %s", got)
	}
	if transport.lastRequest.Header.Get("X-Webhook-Signature") == "" {
		t.Fatalf("expected canonical signature header")
	}
	if transport.lastRequest.Header.Get("X-Webhook-Signature-Legacy") == "" {
		t.Fatalf("expected legacy signature header")
	}
	if len(webhookRepo.updated) == 0 || webhookRepo.updated[len(webhookRepo.updated)-1].DeliveryStatus != entities.WebhookDeliveryStatusDelivered {
		t.Fatalf("expected delivered status update")
	}
}

func TestWebhookDispatcher_SetsNextRetryAtOnFailure(t *testing.T) {
	merchantID := uuid.New()
	merchant := &entities.Merchant{
		ID:              merchantID,
		CallbackURL:     "https://merchant.example/webhook",
		WebhookSecret:   "super-secret",
		WebhookIsActive: true,
	}
	webhookRepo := &fakeWebhookLogRepo{}
	dispatcher := NewWebhookDispatcher(webhookRepo, &fakeMerchantRepo{merchant: merchant}, servicesimpl.NewHMACService())
	transport := &captureRoundTripper{statusCode: http.StatusInternalServerError}
	dispatcher.httpClient = &http.Client{Transport: transport}

	delivery := &entities.WebhookDelivery{
		ID:             uuid.New(),
		MerchantID:     merchantID,
		PaymentID:      uuid.New(),
		EventType:      "COMPLETED",
		Payload:        null.JSONFrom([]byte(`{"status":"COMPLETED"}`)),
		DeliveryStatus: entities.WebhookDeliveryStatusPending,
		CreatedAt:      time.Now(),
	}

	if err := dispatcher.Dispatch(context.Background(), delivery); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}

	last := webhookRepo.updated[len(webhookRepo.updated)-1]
	if last.DeliveryStatus != entities.WebhookDeliveryStatusRetrying {
		t.Fatalf("expected retrying status, got %s", last.DeliveryStatus)
	}
	if last.NextRetryAt == nil {
		t.Fatalf("expected next retry timestamp")
	}
	if !last.NextRetryAt.After(time.Now()) {
		t.Fatalf("expected future retry timestamp")
	}
}
