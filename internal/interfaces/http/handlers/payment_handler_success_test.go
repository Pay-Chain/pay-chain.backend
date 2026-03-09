package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/interfaces/http/middleware"
)

type paymentServiceStub struct {
	createFn        func(ctx context.Context, userID uuid.UUID, input *entities.CreatePaymentInput) (*entities.CreatePaymentResponse, error)
	getFn           func(ctx context.Context, id uuid.UUID) (*entities.Payment, error)
	listFn          func(ctx context.Context, userID uuid.UUID, page, limit int) ([]*entities.Payment, int, error)
	eventsFn        func(ctx context.Context, paymentID uuid.UUID) ([]*entities.PaymentEvent, error)
	privacyFn       func(ctx context.Context, paymentID uuid.UUID) (*entities.PaymentPrivacyStatus, error)
	retryPrivacyFn  func(ctx context.Context, paymentID uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error)
	claimPrivacyFn  func(ctx context.Context, paymentID uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error)
	refundPrivacyFn func(ctx context.Context, paymentID uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error)
}

func (s paymentServiceStub) CreatePayment(ctx context.Context, userID uuid.UUID, input *entities.CreatePaymentInput) (*entities.CreatePaymentResponse, error) {
	return s.createFn(ctx, userID, input)
}
func (s paymentServiceStub) GetPayment(ctx context.Context, id uuid.UUID) (*entities.Payment, error) {
	return s.getFn(ctx, id)
}
func (s paymentServiceStub) GetPaymentsByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*entities.Payment, int, error) {
	return s.listFn(ctx, userID, page, limit)
}
func (s paymentServiceStub) GetPaymentEvents(ctx context.Context, paymentID uuid.UUID) ([]*entities.PaymentEvent, error) {
	return s.eventsFn(ctx, paymentID)
}
func (s paymentServiceStub) GetPaymentPrivacyStatus(ctx context.Context, paymentID uuid.UUID) (*entities.PaymentPrivacyStatus, error) {
	if s.privacyFn == nil {
		return &entities.PaymentPrivacyStatus{PaymentID: paymentID, Stage: entities.PrivacyLifecycleUnknown}, nil
	}
	return s.privacyFn(ctx, paymentID)
}
func (s paymentServiceStub) BuildRetryPrivacyRecoveryTx(ctx context.Context, paymentID uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error) {
	if s.retryPrivacyFn == nil {
		return nil, errors.New("retry not implemented")
	}
	return s.retryPrivacyFn(ctx, paymentID, onchainPaymentID)
}
func (s paymentServiceStub) BuildClaimPrivacyRecoveryTx(ctx context.Context, paymentID uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error) {
	if s.claimPrivacyFn == nil {
		return nil, errors.New("claim not implemented")
	}
	return s.claimPrivacyFn(ctx, paymentID, onchainPaymentID)
}
func (s paymentServiceStub) BuildRefundPrivacyRecoveryTx(ctx context.Context, paymentID uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error) {
	if s.refundPrivacyFn == nil {
		return nil, errors.New("refund not implemented")
	}
	return s.refundPrivacyFn(ctx, paymentID, onchainPaymentID)
}

func TestPaymentHandler_SuccessAndErrorMappings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	paymentID := uuid.New()

	service := paymentServiceStub{
		createFn: func(_ context.Context, gotUserID uuid.UUID, input *entities.CreatePaymentInput) (*entities.CreatePaymentResponse, error) {
			if input.Amount == "bad" {
				return nil, domainerrors.ErrBadRequest
			}
			if input.Amount == "boom" {
				return nil, errors.New("create boom")
			}
			return &entities.CreatePaymentResponse{PaymentID: paymentID, Status: entities.PaymentStatusPending}, nil
		},
		getFn: func(_ context.Context, id uuid.UUID) (*entities.Payment, error) {
			if id == paymentID {
				return &entities.Payment{ID: id, Status: entities.PaymentStatusPending}, nil
			}
			return nil, domainerrors.ErrNotFound
		},
		listFn: func(_ context.Context, gotUserID uuid.UUID, page, limit int) ([]*entities.Payment, int, error) {
			if page == 9 {
				return nil, 0, errors.New("list boom")
			}
			return []*entities.Payment{{ID: paymentID, Status: entities.PaymentStatusPending}}, 1, nil
		},
		eventsFn: func(_ context.Context, id uuid.UUID) ([]*entities.PaymentEvent, error) {
			if id == paymentID {
				return []*entities.PaymentEvent{{PaymentID: id, EventType: entities.PaymentEventTypeCreated, CreatedAt: time.Now()}}, nil
			}
			return nil, errors.New("events boom")
		},
		privacyFn: func(_ context.Context, id uuid.UUID) (*entities.PaymentPrivacyStatus, error) {
			if id == paymentID {
				return &entities.PaymentPrivacyStatus{PaymentID: id, Stage: entities.PrivacyLifecyclePendingOnSource, IsPrivacyCandidate: true}, nil
			}
			return nil, domainerrors.ErrNotFound
		},
		retryPrivacyFn: func(_ context.Context, id uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error) {
			if id != paymentID {
				return nil, domainerrors.ErrNotFound
			}
			if onchainPaymentID == "" {
				return nil, domainerrors.BadRequest("onchainPaymentId required")
			}
			return &entities.PaymentPrivacyRecoveryTx{
				Action:           entities.PrivacyRecoveryActionRetry,
				PaymentID:        id,
				OnchainPaymentID: onchainPaymentID,
				Calldata:         "0x1234",
			}, nil
		},
		claimPrivacyFn: func(_ context.Context, id uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error) {
			if id != paymentID {
				return nil, domainerrors.ErrNotFound
			}
			return &entities.PaymentPrivacyRecoveryTx{
				Action:           entities.PrivacyRecoveryActionClaim,
				PaymentID:        id,
				OnchainPaymentID: onchainPaymentID,
				Calldata:         "0x5678",
			}, nil
		},
		refundPrivacyFn: func(_ context.Context, id uuid.UUID, onchainPaymentID string) (*entities.PaymentPrivacyRecoveryTx, error) {
			if id != paymentID {
				return nil, domainerrors.ErrNotFound
			}
			return &entities.PaymentPrivacyRecoveryTx{
				Action:           entities.PrivacyRecoveryActionRefund,
				PaymentID:        id,
				OnchainPaymentID: onchainPaymentID,
				Calldata:         "0x9999",
			}, nil
		},
	}

	h := NewPaymentHandler(service)
	r := gin.New()
	withUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
	r.POST("/payments", withUser, h.CreatePayment)
	r.GET("/payments/:id", h.GetPayment)
	r.GET("/payments", withUser, h.ListPayments)
	r.GET("/payments/:id/events", h.GetPaymentEvents)
	r.GET("/payments/:id/privacy-status", h.GetPaymentPrivacyStatus)
	r.POST("/payments/:id/privacy/retry", h.RetryPrivacyForward)
	r.POST("/payments/:id/privacy/claim", h.ClaimPrivacyEscrow)
	r.POST("/payments/:id/privacy/refund", h.RefundPrivacyEscrow)

	// Create success
	createBody := []byte(`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161","sourceTokenAddress":"0xabc","destTokenAddress":"0xdef","amount":"1","decimals":6,"receiverAddress":"0x123"}`)
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	// Create bad request mapping
	createBody = []byte(`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161","sourceTokenAddress":"0xabc","destTokenAddress":"0xdef","amount":"bad","decimals":6,"receiverAddress":"0x123"}`)
	req = httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}

	// Create generic error
	createBody = []byte(`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161","sourceTokenAddress":"0xabc","destTokenAddress":"0xdef","amount":"boom","decimals":6,"receiverAddress":"0x123"}`)
	req = httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	// Get success
	req = httptest.NewRequest(http.MethodGet, "/payments/"+paymentID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Get not found mapping
	req = httptest.NewRequest(http.MethodGet, "/payments/"+uuid.NewString(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}

	// List success
	req = httptest.NewRequest(http.MethodGet, "/payments?page=1&limit=5", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// List generic error
	req = httptest.NewRequest(http.MethodGet, "/payments?page=9&limit=5", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	// Events success
	req = httptest.NewRequest(http.MethodGet, "/payments/"+paymentID.String()+"/events", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Events generic error
	req = httptest.NewRequest(http.MethodGet, "/payments/"+uuid.NewString()+"/events", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	// Privacy status success
	req = httptest.NewRequest(http.MethodGet, "/payments/"+paymentID.String()+"/privacy-status", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Privacy status not found mapping
	req = httptest.NewRequest(http.MethodGet, "/payments/"+uuid.NewString()+"/privacy-status", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}

	// Retry recovery payload success
	recoveryBody := []byte(`{"onchainPaymentId":"0x1111111111111111111111111111111111111111111111111111111111111111"}`)
	req = httptest.NewRequest(http.MethodPost, "/payments/"+paymentID.String()+"/privacy/retry", bytes.NewReader(recoveryBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Retry recovery bad request mapping
	req = httptest.NewRequest(http.MethodPost, "/payments/"+paymentID.String()+"/privacy/retry", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}
