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
	"pay-chain.backend/internal/usecases"
)

type paymentRequestServiceStub struct {
	createFn func(ctx context.Context, input usecases.CreatePaymentRequestInput) (*usecases.CreatePaymentRequestOutput, error)
	getFn    func(ctx context.Context, id uuid.UUID) (*entities.PaymentRequest, *entities.PaymentRequestTxData, error)
	listFn   func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error)
}

func (s paymentRequestServiceStub) CreatePaymentRequest(ctx context.Context, input usecases.CreatePaymentRequestInput) (*usecases.CreatePaymentRequestOutput, error) {
	return s.createFn(ctx, input)
}
func (s paymentRequestServiceStub) GetPaymentRequest(ctx context.Context, id uuid.UUID) (*entities.PaymentRequest, *entities.PaymentRequestTxData, error) {
	return s.getFn(ctx, id)
}
func (s paymentRequestServiceStub) ListPaymentRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error) {
	return s.listFn(ctx, userID, limit, offset)
}

func TestPaymentRequestHandler_SuccessAndErrorMappings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	requestID := uuid.New()
	expiresAt := time.Now().Add(10 * time.Minute)

	service := paymentRequestServiceStub{
		createFn: func(_ context.Context, input usecases.CreatePaymentRequestInput) (*usecases.CreatePaymentRequestOutput, error) {
			if input.Amount == "err" {
				return nil, errors.New("create request boom")
			}
			return &usecases.CreatePaymentRequestOutput{
				RequestID:     requestID.String(),
				ExpiresAt:     expiresAt,
				ExpiresInSecs: 600,
				TxData: &entities.PaymentRequestTxData{
					RequestID:       requestID.String(),
					ContractAddress: "0xgateway",
					ChainID:         "eip155:8453",
					Hex:             "0xabc",
				},
			}, nil
		},
		getFn: func(_ context.Context, id uuid.UUID) (*entities.PaymentRequest, *entities.PaymentRequestTxData, error) {
			if id != requestID {
				return nil, nil, domainerrors.ErrNotFound
			}
			return &entities.PaymentRequest{
					ID:            requestID,
					NetworkID:     "eip155:8453",
					Amount:        "1000",
					Decimals:      6,
					WalletAddress: "0xwallet",
					Description:   "Invoice A",
					Status:        entities.PaymentRequestStatusPending,
					ExpiresAt:     expiresAt,
				}, &entities.PaymentRequestTxData{
					RequestID:       requestID.String(),
					ContractAddress: "0xgateway",
					To:              "0xgateway",
					Hex:             "0xabc",
				}, nil
		},
		listFn: func(_ context.Context, userID uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error) {
			if offset > 0 {
				return nil, 0, errors.New("list boom")
			}
			return []*entities.PaymentRequest{{ID: requestID, Amount: "1000", Status: entities.PaymentRequestStatusPending}}, 1, nil
		},
	}

	h := NewPaymentRequestHandler(service)
	r := gin.New()
	withUser := func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	}
	r.POST("/payment-requests", withUser, h.CreatePaymentRequest)
	r.GET("/payment-requests/:id", h.GetPaymentRequest)
	r.GET("/payment-requests", withUser, h.ListPaymentRequests)
	r.GET("/pay/:id", h.GetPublicPaymentRequest)

	// Create success
	body := []byte(`{"chainId":"eip155:8453","tokenAddress":"0xusdc","amount":"1000","decimals":6,"description":"Invoice A"}`)
	req := httptest.NewRequest(http.MethodPost, "/payment-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	// Create generic error
	body = []byte(`{"chainId":"eip155:8453","tokenAddress":"0xusdc","amount":"err","decimals":6,"description":"Invoice A"}`)
	req = httptest.NewRequest(http.MethodPost, "/payment-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	// Get payment request success
	req = httptest.NewRequest(http.MethodGet, "/payment-requests/"+requestID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Get payment request not found mapping
	req = httptest.NewRequest(http.MethodGet, "/payment-requests/"+uuid.NewString(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}

	// List success
	req = httptest.NewRequest(http.MethodGet, "/payment-requests?page=1&limit=10", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// List generic error
	req = httptest.NewRequest(http.MethodGet, "/payment-requests?page=2&limit=10", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	// Public request success
	req = httptest.NewRequest(http.MethodGet, "/pay/"+requestID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Public request not found mapping
	req = httptest.NewRequest(http.MethodGet, "/pay/"+uuid.NewString(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}
