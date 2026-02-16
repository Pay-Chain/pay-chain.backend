package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

type paymentAppServiceStub struct {
	createFn func(ctx context.Context, input *entities.CreatePaymentAppInput) (*entities.CreatePaymentResponse, error)
}

func (s paymentAppServiceStub) CreatePaymentApp(ctx context.Context, input *entities.CreatePaymentAppInput) (*entities.CreatePaymentResponse, error) {
	return s.createFn(ctx, input)
}

func TestPaymentAppHandler_CreatePaymentApp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("bad request", func(t *testing.T) {
		r := gin.New()
		h := NewPaymentAppHandler(paymentAppServiceStub{
			createFn: func(context.Context, *entities.CreatePaymentAppInput) (*entities.CreatePaymentResponse, error) {
				t.Fatal("should not be called")
				return nil, nil
			},
		})
		r.POST("/payment-app", h.CreatePaymentApp)

		req := httptest.NewRequest(http.MethodPost, "/payment-app", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("usecase error", func(t *testing.T) {
		r := gin.New()
		h := NewPaymentAppHandler(paymentAppServiceStub{
			createFn: func(context.Context, *entities.CreatePaymentAppInput) (*entities.CreatePaymentResponse, error) {
				return nil, errors.New("boom")
			},
		})
		r.POST("/payment-app", h.CreatePaymentApp)

		body := `{"sourceChainId":"eip155:8453","destChainId":"eip155:42161","sourceTokenAddress":"0x1","destTokenAddress":"0x2","amount":"1","decimals":6,"senderWalletAddress":"0xabc","receiverAddress":"0xdef"}`
		req := httptest.NewRequest(http.MethodPost, "/payment-app", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		r := gin.New()
		expectedID := uuid.New()
		h := NewPaymentAppHandler(paymentAppServiceStub{
			createFn: func(_ context.Context, input *entities.CreatePaymentAppInput) (*entities.CreatePaymentResponse, error) {
				if input.SourceChainID != "eip155:8453" {
					t.Fatalf("unexpected source chain id: %s", input.SourceChainID)
				}
				return &entities.CreatePaymentResponse{
					PaymentID: expectedID,
					Status:    entities.PaymentStatusPending,
				}, nil
			},
		})
		r.POST("/payment-app", h.CreatePaymentApp)

		body := `{"sourceChainId":"eip155:8453","destChainId":"eip155:42161","sourceTokenAddress":"0x1","destTokenAddress":"0x2","amount":"1","decimals":6,"senderWalletAddress":"0xabc","receiverAddress":"0xdef"}`
		req := httptest.NewRequest(http.MethodPost, "/payment-app", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(expectedID.String())) {
			t.Fatalf("expected response to contain payment id %s, body=%s", expectedID.String(), w.Body.String())
		}
	})
}
