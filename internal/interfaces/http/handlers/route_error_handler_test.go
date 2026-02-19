package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/usecases"
)

type routeErrorServiceStub struct {
	result *usecases.RouteErrorDiagnostics
	err    error
}

func (s routeErrorServiceStub) GetRouteError(_ context.Context, _ string, _ string) (*usecases.RouteErrorDiagnostics, error) {
	return s.result, s.err
}

func TestRouteErrorHandler_GetRouteError_Success(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &RouteErrorHandler{
		service: routeErrorServiceStub{
			result: &usecases.RouteErrorDiagnostics{
				SourceChainID: "eip155:8453",
				Gateway:       "0xabc",
				PaymentIDHex:  "0x01",
				Decoded: usecases.RouteErrorDecoded{
					RawHex:  "0x",
					Message: "no route error recorded",
				},
			},
		},
	}

	r := gin.New()
	r.GET("/diag/:paymentId", h.GetRouteError)

	req := httptest.NewRequest(http.MethodGet, "/diag/0x01?sourceChainId=eip155:8453", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRouteErrorHandler_GetRouteError_MissingSourceChainID(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &RouteErrorHandler{service: routeErrorServiceStub{}}
	r := gin.New()
	r.GET("/diag/:paymentId", h.GetRouteError)

	req := httptest.NewRequest(http.MethodGet, "/diag/0x01", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRouteErrorHandler_GetRouteError_UsecaseError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &RouteErrorHandler{
		service: routeErrorServiceStub{
			err: domainerrors.BadRequest("invalid sourceChainId"),
		},
	}
	r := gin.New()
	r.GET("/diag/:paymentId", h.GetRouteError)

	req := httptest.NewRequest(http.MethodGet, "/diag/0x01?sourceChainId=bad", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
