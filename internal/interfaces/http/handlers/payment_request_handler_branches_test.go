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
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/usecases"
)

func TestPaymentRequestHandler_ExtraBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	requestID := uuid.New()

	var gotListLimit int
	var gotListOffset int
	service := paymentRequestServiceStub{
		createFn: func(_ context.Context, _ usecases.CreatePaymentRequestInput) (*usecases.CreatePaymentRequestOutput, error) {
			return nil, errors.New("create failed")
		},
		getFn: func(_ context.Context, id uuid.UUID) (*entities.PaymentRequest, *entities.PaymentRequestTxData, error) {
			if id == requestID {
				return &entities.PaymentRequest{
						ID:            requestID,
						NetworkID:     "eip155:8453",
						Amount:        "10",
						Decimals:      6,
						WalletAddress: "0xwallet",
						Status:        entities.PaymentRequestStatusPending,
						ExpiresAt:     time.Now().Add(time.Minute),
					}, &entities.PaymentRequestTxData{
						ContractAddress: "0xgateway",
					}, errors.New("get failed")
			}
			return nil, nil, domainerrors.ErrNotFound
		},
		listFn: func(_ context.Context, _ uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error) {
			gotListLimit = limit
			gotListOffset = offset
			return []*entities.PaymentRequest{}, 0, nil
		},
	}

	h := NewPaymentRequestHandler(service)
	r := gin.New()
	r.POST("/payment-requests", h.CreatePaymentRequest)
	r.GET("/payment-requests/:id", h.GetPaymentRequest)
	r.GET("/payment-requests", h.ListPaymentRequests)
	r.GET("/pay/:id", h.GetPublicPaymentRequest)

	// Create unauthorized when userID missing.
	req := httptest.NewRequest(http.MethodPost, "/payment-requests", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Create bad payload bind error.
	rWithUser := gin.New()
	rWithUser.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	rWithUser.POST("/payment-requests", h.CreatePaymentRequest)
	rWithUser.GET("/payment-requests", h.ListPaymentRequests)

	req = httptest.NewRequest(http.MethodPost, "/payment-requests", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	rWithUser.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Create internal service error.
	req = httptest.NewRequest(http.MethodPost, "/payment-requests", bytes.NewBufferString(`{"chainId":"eip155:8453","tokenAddress":"0x1","amount":"1","decimals":6}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	rWithUser.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// Get invalid ID.
	req = httptest.NewRequest(http.MethodGet, "/payment-requests/not-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Get mapped not found.
	req = httptest.NewRequest(http.MethodGet, "/payment-requests/"+uuid.NewString(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	// Get internal error.
	req = httptest.NewRequest(http.MethodGet, "/payment-requests/"+requestID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// List unauthorized when user missing.
	req = httptest.NewRequest(http.MethodGet, "/payment-requests", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// List page/limit sanitization.
	req = httptest.NewRequest(http.MethodGet, "/payment-requests?page=0&limit=1000", nil)
	w = httptest.NewRecorder()
	rWithUser.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 10, gotListLimit)
	require.Equal(t, 0, gotListOffset)

	// Public get invalid ID.
	req = httptest.NewRequest(http.MethodGet, "/pay/not-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Public get internal error.
	req = httptest.NewRequest(http.MethodGet, "/pay/"+requestID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
