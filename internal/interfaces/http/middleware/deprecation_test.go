package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestDeprecationMiddleware_SetsHeaders(t *testing.T) {
	resetLegacyEndpointObservabilityForTests()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(MerchantIDKey, uuid.MustParse("0195d4b4-1e2c-7f2f-9aa1-123456789012"))
		c.Next()
	})
	r.Use(DeprecationMiddleware(DeprecationOptions{
		Replacement:    "/api/v1/create-payment",
		Sunset:         time.Date(2026, time.June, 30, 23, 59, 59, 0, time.UTC),
		EndpointFamily: "legacy_payment_requests",
	}))
	r.GET("/legacy", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/legacy", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Deprecation") != "true" {
		t.Fatalf("expected deprecation header")
	}
	if rec.Header().Get("X-Deprecated-Replaced-By") != "/api/v1/create-payment" {
		t.Fatalf("unexpected replacement header: %s", rec.Header().Get("X-Deprecated-Replaced-By"))
	}
	if rec.Header().Get("Link") == "" || rec.Header().Get("Sunset") == "" {
		t.Fatalf("expected link and sunset headers")
	}

	snapshot := GetLegacyEndpointObservabilitySnapshot()
	if snapshot.Summary.TrackedEndpoints != 1 || snapshot.Summary.TotalHits != 1 {
		t.Fatalf("unexpected snapshot summary: %+v", snapshot.Summary)
	}
	if len(snapshot.Endpoints) != 1 || snapshot.Endpoints[0].EndpointFamily != "legacy_payment_requests" {
		t.Fatalf("unexpected snapshot endpoints: %+v", snapshot.Endpoints)
	}
}

func TestDeprecationMiddleware_DisabledModeBlocksRoute(t *testing.T) {
	resetLegacyEndpointObservabilityForTests()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DeprecationMiddleware(DeprecationOptions{
		Replacement:    "/api/v1/partner/payment-sessions/resolve-code",
		Sunset:         time.Date(2026, time.June, 30, 23, 59, 59, 0, time.UTC),
		EndpointFamily: "legacy_resolve_code",
		Mode:           LegacyModeDisabled,
	}))
	r.GET("/legacy", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/legacy", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d", rec.Code)
	}
	if rec.Header().Get("X-Legacy-Endpoint-Mode") != LegacyModeDisabled {
		t.Fatalf("expected disabled mode header")
	}
}
