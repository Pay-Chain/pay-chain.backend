package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

type paymentBridgeRepoStub struct {
	items map[uuid.UUID]*entities.PaymentBridge
}

func newPaymentBridgeRepoStub() *paymentBridgeRepoStub {
	return &paymentBridgeRepoStub{items: map[uuid.UUID]*entities.PaymentBridge{}}
}

func (s *paymentBridgeRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.PaymentBridge, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return &entities.PaymentBridge{ID: item.ID, Name: item.Name}, nil
}

func (s *paymentBridgeRepoStub) GetByName(_ context.Context, _ string) (*entities.PaymentBridge, error) {
	return nil, domainerrors.ErrNotFound
}

func (s *paymentBridgeRepoStub) List(_ context.Context, _ utils.PaginationParams) ([]*entities.PaymentBridge, int64, error) {
	out := make([]*entities.PaymentBridge, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, &entities.PaymentBridge{ID: item.ID, Name: item.Name})
	}
	return out, int64(len(out)), nil
}

func (s *paymentBridgeRepoStub) Create(_ context.Context, bridge *entities.PaymentBridge) error {
	s.items[bridge.ID] = &entities.PaymentBridge{ID: bridge.ID, Name: bridge.Name}
	return nil
}

func (s *paymentBridgeRepoStub) Update(_ context.Context, bridge *entities.PaymentBridge) error {
	if _, ok := s.items[bridge.ID]; !ok {
		return domainerrors.ErrNotFound
	}
	s.items[bridge.ID] = &entities.PaymentBridge{ID: bridge.ID, Name: bridge.Name}
	return nil
}

func (s *paymentBridgeRepoStub) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.items[id]; !ok {
		return domainerrors.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func TestPaymentConfigHandler_PaymentBridgeCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bridgeRepo := newPaymentBridgeRepoStub()
	h := NewPaymentConfigHandler(bridgeRepo, nil, nil, nil, nil)

	r := gin.New()
	r.POST("/payment-bridges", h.CreatePaymentBridge)
	r.GET("/payment-bridges", h.ListPaymentBridges)
	r.PUT("/payment-bridges/:id", h.UpdatePaymentBridge)
	r.DELETE("/payment-bridges/:id", h.DeletePaymentBridge)

	// Create
	createBody := []byte(`{"name":"CCIP"}`)
	req := httptest.NewRequest(http.MethodPost, "/payment-bridges", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created struct {
		Bridge entities.PaymentBridge `json:"bridge"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.Bridge.Name != "CCIP" {
		t.Fatalf("expected bridge name CCIP, got %s", created.Bridge.Name)
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/payment-bridges", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Update
	updateBody := []byte(`{"name":"Hyperbridge"}`)
	req = httptest.NewRequest(http.MethodPut, "/payment-bridges/"+created.Bridge.ID.String(), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/payment-bridges/"+created.Bridge.ID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPaymentConfigHandler_HelperFunctions(t *testing.T) {
	if got := defaultDecimal(""); got != "0" {
		t.Fatalf("expected 0, got %s", got)
	}
	if got := defaultDecimal("0.01"); got != "0.01" {
		t.Fatalf("expected 0.01, got %s", got)
	}

	id, err := parseUUIDPtr(uuid.NewString())
	if err != nil || id == nil {
		t.Fatalf("expected valid uuid ptr, got err=%v", err)
	}

	nilID, err := parseUUIDPtr("")
	if err != nil || nilID != nil {
		t.Fatalf("expected nil uuid ptr for empty input, got id=%v err=%v", nilID, err)
	}
}
