package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminHandler_BasicValidationBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(nil, nil, nil)

	r := gin.New()
	r.GET("/stats", h.GetStats)
	r.PUT("/merchants/:id/status", h.UpdateMerchantStatus)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/merchants/bad-id/status", strings.NewReader(`{"status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSmartContractHandler_ValidationBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSmartContractHandler(nil, nil)

	r := gin.New()
	r.POST("/contracts", h.CreateSmartContract)
	r.GET("/contracts/:id", h.GetSmartContract)
	r.GET("/contracts/lookup", h.GetContractByChainAndAddress)
	r.DELETE("/contracts/:id", h.DeleteSmartContract)
	r.PUT("/contracts/:id", h.UpdateSmartContract)

	req := httptest.NewRequest(http.MethodPost, "/contracts", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/contracts/bad-id", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/contracts/lookup", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/contracts/bad-id", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/contracts/bad-id", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
