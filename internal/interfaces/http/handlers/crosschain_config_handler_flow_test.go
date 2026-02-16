package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/usecases"
)

func TestCrosschainConfigHandler_RecheckAutoFixPreflight_UsecaseErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ucase := usecases.NewCrosschainConfigUsecase(
		&cfgChainRepoStub{},
		cfgTokenRepoStub{},
		cfgContractRepoStub{},
		nil,
		usecases.NewOnchainAdapterUsecase(&cfgChainRepoStub{}, cfgContractRepoStub{}, nil, ""),
	)
	h := NewCrosschainConfigHandler(ucase)

	r := gin.New()
	r.POST("/recheck", h.Recheck)
	r.POST("/autofix", h.AutoFix)
	r.GET("/preflight", h.Preflight)

	body := `{"sourceChainId":"eip155:8453","destChainId":"eip155:42161"}`

	req := httptest.NewRequest(http.MethodPost, "/recheck", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from recheck usecase error, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/autofix", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from autofix usecase error, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/preflight?sourceChainId=eip155:8453&destChainId=eip155:42161", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from preflight usecase error, got %d body=%s", rec.Code, rec.Body.String())
	}
}

