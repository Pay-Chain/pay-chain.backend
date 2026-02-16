package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
)

type merchantRepoStub struct {
	byUser map[uuid.UUID]*entities.Merchant
}

func newMerchantRepoStub() *merchantRepoStub {
	return &merchantRepoStub{byUser: map[uuid.UUID]*entities.Merchant{}}
}

func (s *merchantRepoStub) Create(_ context.Context, merchant *entities.Merchant) error {
	if merchant.ID == uuid.Nil {
		merchant.ID = uuid.New()
	}
	if merchant.CreatedAt.IsZero() {
		merchant.CreatedAt = time.Now()
	}
	s.byUser[merchant.UserID] = merchant
	return nil
}

func (s *merchantRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Merchant, error) {
	for _, m := range s.byUser {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, domainerrors.ErrNotFound
}

func (s *merchantRepoStub) GetByUserID(_ context.Context, userID uuid.UUID) (*entities.Merchant, error) {
	m, ok := s.byUser[userID]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return m, nil
}

func (s *merchantRepoStub) Update(context.Context, *entities.Merchant) error { return nil }
func (s *merchantRepoStub) UpdateStatus(_ context.Context, id uuid.UUID, status entities.MerchantStatus) error {
	for _, m := range s.byUser {
		if m.ID == id {
			m.Status = status
			return nil
		}
	}
	return domainerrors.ErrNotFound
}
func (s *merchantRepoStub) SoftDelete(context.Context, uuid.UUID) error { return nil }
func (s *merchantRepoStub) List(context.Context) ([]*entities.Merchant, error) {
	out := make([]*entities.Merchant, 0, len(s.byUser))
	for _, m := range s.byUser {
		out = append(out, m)
	}
	return out, nil
}

type merchantUserRepoStub struct {
	users map[uuid.UUID]*entities.User
}

func (s merchantUserRepoStub) Create(context.Context, *entities.User) error { return nil }
func (s merchantUserRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.User, error) {
	u, ok := s.users[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return u, nil
}
func (s merchantUserRepoStub) GetByEmail(context.Context, string) (*entities.User, error) {
	return nil, domainerrors.ErrNotFound
}
func (s merchantUserRepoStub) Update(context.Context, *entities.User) error                    { return nil }
func (s merchantUserRepoStub) UpdatePassword(context.Context, uuid.UUID, string) error         { return nil }
func (s merchantUserRepoStub) SoftDelete(context.Context, uuid.UUID) error                     { return nil }
func (s merchantUserRepoStub) List(context.Context, string) ([]*entities.User, error)          { return nil, nil }

func TestMerchantHandler_ApplyAndGetStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	repo := newMerchantRepoStub()

	uc := usecases.NewMerchantUsecase(
		repo,
		merchantUserRepoStub{
			users: map[uuid.UUID]*entities.User{
				userID: {ID: userID, Email: "m@paychain.io", Role: entities.UserRoleUser},
			},
		},
	)
	h := NewMerchantHandler(uc)

	r := gin.New()
	withUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
	r.POST("/merchants/apply", withUser, h.ApplyMerchant)
	r.GET("/merchants/status", withUser, h.GetMerchantStatus)

	applyBody := []byte(`{"merchantType":"UMKM","businessName":"Warung Sukses","businessEmail":"warung@example.com","taxId":"NPWP-123","businessAddress":"Jalan Mawar"}`)
	req := httptest.NewRequest(http.MethodPost, "/merchants/apply", bytes.NewReader(applyBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("submitted successfully")) {
		t.Fatalf("unexpected apply response: %s", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/merchants/status", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var status entities.MerchantStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("unmarshal status response: %v", err)
	}
	if status.Status != entities.MerchantStatusPending || status.BusinessName != "Warung Sukses" {
		t.Fatalf("unexpected status response: %+v", status)
	}
}

func TestMerchantHandler_Apply_InvalidTypeMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	repo := newMerchantRepoStub()

	uc := usecases.NewMerchantUsecase(
		repo,
		merchantUserRepoStub{
			users: map[uuid.UUID]*entities.User{
				userID: {ID: userID, Email: "m@paychain.io", Role: entities.UserRoleUser},
			},
		},
	)
	h := NewMerchantHandler(uc)

	r := gin.New()
	r.POST("/merchants/apply", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		h.ApplyMerchant(c)
	})

	body := []byte(`{"merchantType":"INVALID","businessName":"Warung Sukses","businessEmail":"warung@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/merchants/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestMerchantHandler_GetStatus_ApprovedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	repo := newMerchantRepoStub()
	repo.byUser[userID] = &entities.Merchant{
		ID:           uuid.New(),
		UserID:       userID,
		BusinessName: "Approved Biz",
		MerchantType: entities.MerchantTypeCorporate,
		Status:       entities.MerchantStatusActive,
		VerifiedAt:   ptrNow(),
	}

	uc := usecases.NewMerchantUsecase(
		repo,
		merchantUserRepoStub{
			users: map[uuid.UUID]*entities.User{
				userID: {ID: userID, Email: "m@paychain.io", Role: entities.UserRoleUser},
			},
		},
	)
	h := NewMerchantHandler(uc)

	r := gin.New()
	r.GET("/merchants/status", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		h.GetMerchantStatus(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/merchants/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("account is active")) {
		t.Fatalf("unexpected status message: %s", w.Body.String())
	}
}

func ptrNow() *time.Time {
	now := time.Now()
	return &now
}

var _ = null.String{}
