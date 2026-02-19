package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

type adminUserRepoStub struct {
	listFn func(ctx context.Context, search string) ([]*entities.User, error)
}

func (s *adminUserRepoStub) Create(context.Context, *entities.User) error { return nil }
func (s *adminUserRepoStub) GetByID(context.Context, uuid.UUID) (*entities.User, error) {
	return nil, nil
}
func (s *adminUserRepoStub) GetByEmail(context.Context, string) (*entities.User, error) {
	return nil, nil
}
func (s *adminUserRepoStub) Update(context.Context, *entities.User) error            { return nil }
func (s *adminUserRepoStub) UpdatePassword(context.Context, uuid.UUID, string) error { return nil }
func (s *adminUserRepoStub) SoftDelete(context.Context, uuid.UUID) error             { return nil }
func (s *adminUserRepoStub) List(ctx context.Context, search string) ([]*entities.User, error) {
	return s.listFn(ctx, search)
}

type adminMerchantRepoStub struct {
	listFn   func(ctx context.Context) ([]*entities.Merchant, error)
	getByID  func(ctx context.Context, id uuid.UUID) (*entities.Merchant, error)
	updateFn func(ctx context.Context, id uuid.UUID, status entities.MerchantStatus) error
}

func (s *adminMerchantRepoStub) Create(context.Context, *entities.Merchant) error { return nil }
func (s *adminMerchantRepoStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.Merchant, error) {
	return s.getByID(ctx, id)
}
func (s *adminMerchantRepoStub) GetByUserID(context.Context, uuid.UUID) (*entities.Merchant, error) {
	return nil, nil
}
func (s *adminMerchantRepoStub) Update(context.Context, *entities.Merchant) error { return nil }
func (s *adminMerchantRepoStub) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.MerchantStatus) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, id, status)
	}
	return nil
}
func (s *adminMerchantRepoStub) SoftDelete(context.Context, uuid.UUID) error { return nil }
func (s *adminMerchantRepoStub) List(ctx context.Context) ([]*entities.Merchant, error) {
	return s.listFn(ctx)
}

type adminPaymentRepoStub struct{}

func (adminPaymentRepoStub) Create(context.Context, *entities.Payment) error { return nil }
func (adminPaymentRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Payment, error) {
	return nil, nil
}
func (adminPaymentRepoStub) GetByUserID(context.Context, uuid.UUID, int, int) ([]*entities.Payment, int, error) {
	return nil, 0, nil
}
func (adminPaymentRepoStub) GetByMerchantID(context.Context, uuid.UUID, int, int) ([]*entities.Payment, int, error) {
	return nil, 0, nil
}
func (adminPaymentRepoStub) UpdateStatus(context.Context, uuid.UUID, entities.PaymentStatus) error {
	return nil
}
func (adminPaymentRepoStub) UpdateDestTxHash(context.Context, uuid.UUID, string) error { return nil }
func (adminPaymentRepoStub) MarkRefunded(context.Context, uuid.UUID) error             { return nil }
func (adminPaymentRepoStub) Update(context.Context, *entities.Payment) error           { return nil }

func TestAdminHandler_ListAndUpdateStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	merchantID := uuid.New()

	h := NewAdminHandler(
		&adminUserRepoStub{
			listFn: func(_ context.Context, search string) ([]*entities.User, error) {
				if search != "abc" {
					t.Fatalf("unexpected search %s", search)
				}
				return []*entities.User{{ID: uuid.New(), Email: "u@paychain.io"}}, nil
			},
		},
		&adminMerchantRepoStub{
			listFn: func(context.Context) ([]*entities.Merchant, error) {
				return []*entities.Merchant{{ID: merchantID, BusinessName: "Biz"}}, nil
			},
			getByID: func(_ context.Context, id uuid.UUID) (*entities.Merchant, error) {
				if id == merchantID {
					return &entities.Merchant{ID: merchantID}, nil
				}
				return nil, domainerrors.ErrNotFound
			},
		},
		adminPaymentRepoStub{},
	)

	r := gin.New()
	r.GET("/users", h.ListUsers)
	r.GET("/merchants", h.ListMerchants)
	r.PUT("/merchants/:id/status", h.UpdateMerchantStatus)

	req := httptest.NewRequest(http.MethodGet, "/users?search=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "u@paychain.io")

	req = httptest.NewRequest(http.MethodGet, "/merchants", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Biz")

	req = httptest.NewRequest(http.MethodPut, "/merchants/"+merchantID.String()+"/status", strings.NewReader(`{"status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Merchant status updated")

	req = httptest.NewRequest(http.MethodPut, "/merchants/"+uuid.NewString()+"/status", strings.NewReader(`{"status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}
