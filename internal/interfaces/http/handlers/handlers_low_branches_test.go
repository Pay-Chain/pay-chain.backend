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
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/usecases"
)

func TestChainHandler_UpdateChain_InvalidBodyBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewChainHandler(&chainRepoErrStub{chainRepoStub: newChainRepoStub()})
	r := gin.New()
	r.PUT("/admin/chains/:id", h.UpdateChain)

	req := httptest.NewRequest(http.MethodPut, "/admin/chains/"+uuid.New().String(), bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestPaymentHandler_GetPayment_GenericErrorBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPaymentHandler(paymentServiceStub{
		createFn: func(context.Context, uuid.UUID, *entities.CreatePaymentInput) (*entities.CreatePaymentResponse, error) {
			return nil, errors.New("unused")
		},
		getFn: func(context.Context, uuid.UUID) (*entities.Payment, error) {
			return nil, errors.New("boom")
		},
		listFn: func(context.Context, uuid.UUID, int, int) ([]*entities.Payment, int, error) {
			return nil, 0, errors.New("unused")
		},
		eventsFn: func(context.Context, uuid.UUID) ([]*entities.PaymentEvent, error) {
			return nil, errors.New("unused")
		},
	})

	r := gin.New()
	r.GET("/payments/:id", h.GetPayment)
	req := httptest.NewRequest(http.MethodGet, "/payments/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestPaymentRequestHandler_GenericErrorAndPaginationBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	h := NewPaymentRequestHandler(paymentRequestServiceStub{
		createFn: func(context.Context, usecases.CreatePaymentRequestInput) (*usecases.CreatePaymentRequestOutput, error) {
			return nil, errors.New("unused")
		},
		getFn: func(context.Context, uuid.UUID) (*entities.PaymentRequest, *entities.PaymentRequestTxData, error) {
			return nil, nil, errors.New("boom")
		},
		listFn: func(_ context.Context, _ uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error) {
			if limit != 10 || offset != 0 {
				return nil, 0, errors.New("unexpected pagination normalization")
			}
			return []*entities.PaymentRequest{}, 0, nil
		},
	})

	r := gin.New()
	withUser := func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	}
	r.GET("/payment-requests/:id", h.GetPaymentRequest)
	r.GET("/pay/:id", h.GetPublicPaymentRequest)
	r.GET("/payment-requests", withUser, h.ListPaymentRequests)

	req := httptest.NewRequest(http.MethodGet, "/payment-requests/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/pay/"+uuid.New().String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	// page <1 and limit invalid should normalize to page=1 limit=10 and still succeed
	req = httptest.NewRequest(http.MethodGet, "/payment-requests?page=0&limit=999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestSmartContractHandler_GetDelete_GenericErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSmartContractHandler(&smartContractRepoStub{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.SmartContract, error) {
			return nil, errors.New("db down")
		},
		softDeleteFn: func(context.Context, uuid.UUID) error {
			return errors.New("delete failed")
		},
	}, &smartContractChainRepoStub{})

	r := gin.New()
	r.GET("/contracts/:id", h.GetSmartContract)
	r.DELETE("/contracts/:id", h.DeleteSmartContract)

	req := httptest.NewRequest(http.MethodGet, "/contracts/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/contracts/"+uuid.New().String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}

	// keep not-found mapping covered in same test with explicit domain error
	h2 := NewSmartContractHandler(&smartContractRepoStub{
		softDeleteFn: func(context.Context, uuid.UUID) error {
			return domainerrors.ErrNotFound
		},
	}, &smartContractChainRepoStub{})
	r2 := gin.New()
	r2.DELETE("/contracts/:id", h2.DeleteSmartContract)
	req = httptest.NewRequest(http.MethodDelete, "/contracts/"+uuid.New().String(), nil)
	w = httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}
