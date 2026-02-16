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
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
)

type paymentServiceStub struct {
	createFn func(ctx context.Context, userID uuid.UUID, input *entities.CreatePaymentInput) (*entities.CreatePaymentResponse, error)
	getFn    func(ctx context.Context, id uuid.UUID) (*entities.Payment, error)
	listFn   func(ctx context.Context, userID uuid.UUID, page, limit int) ([]*entities.Payment, int, error)
	eventsFn func(ctx context.Context, paymentID uuid.UUID) ([]*entities.PaymentEvent, error)
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
}
