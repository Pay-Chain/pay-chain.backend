package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/usecases"
)

func TestCrosschainConfigHandler_BulkRoutes_ErrorItems(t *testing.T) {
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
	r.POST("/recheck-bulk", h.RecheckBulk)
	r.POST("/autofix-bulk", h.AutoFixBulk)

	recheckBody := `{"routes":[{"sourceChainId":"eip155:8453","destChainId":"eip155:42161"}]}`
	req := httptest.NewRequest(http.MethodPost, "/recheck-bulk", strings.NewReader(recheckBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "RECHECK_FAILED") {
		t.Fatalf("expected RECHECK_FAILED item, got body=%s", rec.Body.String())
	}

	autofixBody := `{"routes":[{"sourceChainId":"eip155:8453","destChainId":"eip155:42161"}]}`
	req = httptest.NewRequest(http.MethodPost, "/autofix-bulk", strings.NewReader(autofixBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "\"autoFix\"") {
		t.Fatalf("expected autoFix failed step, got body=%s", rec.Body.String())
	}
}

