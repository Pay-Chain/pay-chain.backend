package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/interfaces/http/middleware"
)

func TestPaymentHandler_ListPayments_PaginationNormalization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	var gotPage, gotLimit int
	h := NewPaymentHandler(paymentServiceStub{
		createFn: func(context.Context, uuid.UUID, *entities.CreatePaymentInput) (*entities.CreatePaymentResponse, error) { return nil, nil },
		getFn:    func(context.Context, uuid.UUID) (*entities.Payment, error) { return nil, nil },
		listFn: func(_ context.Context, _ uuid.UUID, page, limit int) ([]*entities.Payment, int, error) {
			gotPage, gotLimit = page, limit
			return []*entities.Payment{}, 0, nil
		},
		eventsFn: func(context.Context, uuid.UUID) ([]*entities.PaymentEvent, error) { return nil, nil },
	})

	r := gin.New()
	r.GET("/payments", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		h.ListPayments(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/payments?page=0&limit=500", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, gotPage)
	require.Equal(t, 10, gotLimit)
}
