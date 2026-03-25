package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestApplyCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	applyCORSMiddleware(r)
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	// with origin
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("unexpected allow-origin: %s", got)
	}

	// allowed netlify origin
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://paymentkita.netlify.app")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://paymentkita.netlify.app" {
		t.Fatalf("unexpected allow-origin for netlify: %s", got)
	}

	// allowed dompet-ku origin
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://api-dompet-ku.excitech.id")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://api-dompet-ku.excitech.id" {
		t.Fatalf("unexpected allow-origin for dompet-ku: %s", got)
	}

	// disallowed origin should not be reflected
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected empty allow-origin for disallowed origin, got %s", got)
	}

	// options preflight
	req = httptest.NewRequest(http.MethodOptions, "/x", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestRegisterHealthRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerHealthRoute(r)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["status"] != "ok" || body["service"] != "payment-kita-backend" || body["version"] != "0.2.0" {
		t.Fatalf("unexpected health payload: %+v", body)
	}
}
