package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

type routePolicyRepoCreateErrStub struct {
	createErr error
	item      *entities.RoutePolicy
}

func (s *routePolicyRepoCreateErrStub) GetByID(_ context.Context, id uuid.UUID) (*entities.RoutePolicy, error) {
	if s.item != nil {
		return s.item, nil
	}
	return &entities.RoutePolicy{ID: id}, nil
}
func (s *routePolicyRepoCreateErrStub) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *routePolicyRepoCreateErrStub) List(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.RoutePolicy, int64, error) {
	return nil, 0, nil
}
func (s *routePolicyRepoCreateErrStub) Create(_ context.Context, item *entities.RoutePolicy) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.item = item
	return nil
}
func (s *routePolicyRepoCreateErrStub) Update(_ context.Context, item *entities.RoutePolicy) error { s.item = item; return nil }
func (s *routePolicyRepoCreateErrStub) Delete(context.Context, uuid.UUID) error                     { return nil }

func TestCrosschainPolicyHandler_CreateUpdate_ExtraBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sourceID := uuid.New()
	destID := uuid.New()
	peerHex := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	chainRepo := &crosschainChainRepoStub{
		getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
			switch chainID {
			case "8453":
				return &entities.Chain{ID: sourceID}, nil
			case "42161":
				return &entities.Chain{ID: destID}, nil
			default:
				return nil, domainerrors.ErrNotFound
			}
		},
		getByCAIP2: func(context.Context, string) (*entities.Chain, error) { return nil, domainerrors.ErrNotFound },
	}
	routeRepo := &routePolicyRepoCreateErrStub{createErr: errors.New("create failed")}
	h := NewCrosschainPolicyHandler(routeRepo, layerZeroRepoNoop{}, chainRepo)

	r := gin.New()
	r.POST("/route", h.CreateRoutePolicy)
	r.PUT("/lz/:id", h.UpdateLayerZeroConfig)

	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(`{"sourceChainId":"bad","destChainId":"42161","defaultBridgeType":0}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid source chain, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(`{"sourceChainId":"8453","destChainId":"bad","defaultBridgeType":0}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid dest chain, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(`{"sourceChainId":"8453","destChainId":"42161","defaultBridgeType":0}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when repo create fails, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/lz/"+uuid.New().String(), strings.NewReader(`{"sourceChainId":"bad","destChainId":"42161","dstEid":1,"peerHex":"`+peerHex+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid source chain in update lz, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/lz/"+uuid.New().String(), strings.NewReader(`{"sourceChainId":"8453","destChainId":"bad","dstEid":1,"peerHex":"`+peerHex+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid dest chain in update lz, got %d body=%s", w.Code, w.Body.String())
	}
}

