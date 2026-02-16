package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/interfaces/http/handlers"
)

func TestRegisterAPIV1Routes_RegistersKeyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	registerAPIV1Routes(r, routeDeps{
		authHandler:                &handlers.AuthHandler{},
		paymentHandler:             &handlers.PaymentHandler{},
		merchantHandler:            &handlers.MerchantHandler{},
		walletHandler:              &handlers.WalletHandler{},
		chainHandler:               &handlers.ChainHandler{},
		tokenHandler:               &handlers.TokenHandler{},
		smartContractHandler:       &handlers.SmartContractHandler{},
		paymentRequestHandler:      &handlers.PaymentRequestHandler{},
		webhookHandler:             &handlers.WebhookHandler{},
		adminHandler:               &handlers.AdminHandler{},
		teamHandler:                &handlers.TeamHandler{},
		apiKeyHandler:              &handlers.ApiKeyHandler{},
		paymentAppHandler:          &handlers.PaymentAppHandler{},
		paymentConfigHandler:       &handlers.PaymentConfigHandler{},
		onchainAdapterHandler:      &handlers.OnchainAdapterHandler{},
		contractConfigAuditHandler: &handlers.ContractConfigAuditHandler{},
		crosschainConfigHandler:    &handlers.CrosschainConfigHandler{},
		crosschainPolicyHandler:    &handlers.CrosschainPolicyHandler{},
		rpcHandler:                 &handlers.RpcHandler{},
		dualAuthMiddleware: func(c *gin.Context) {
			c.Next()
		},
	})

	routes := r.Routes()
	if len(routes) < 40 {
		t.Fatalf("expected many routes registered, got %d", len(routes))
	}

	expects := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/auth/login"},
		{"GET", "/api/v1/auth/me"},
		{"POST", "/api/v1/payments"},
		{"GET", "/api/v1/payments/:id"},
		{"GET", "/api/v1/pay/:id"},
		{"POST", "/api/v1/wallets/connect"},
		{"GET", "/api/v1/admin/stats"},
		{"POST", "/api/v1/admin/onchain-adapters/register"},
		{"POST", "/api/v1/admin/layerzero-configs"},
	}

	for _, exp := range expects {
		found := false
		for _, route := range routes {
			if route.Method == exp.method && route.Path == exp.path {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("route %s %s not registered", exp.method, exp.path)
		}
	}
}

func TestRegisterAPIV1Routes_RouteResponds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerHealthRoute(r)
	registerAPIV1Routes(r, routeDeps{
		authHandler:        &handlers.AuthHandler{},
		dualAuthMiddleware: func(c *gin.Context) { c.Next() },
	})

	// Smoke: unrelated helper route still works after route registration.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
