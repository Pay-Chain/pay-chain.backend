package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/usecases"
)

type createPaymentUsecaseStub struct {
	out       *usecases.CreatePaymentOutput
	err       error
	lastInput *usecases.CreatePaymentInput
}

func (s *createPaymentUsecaseStub) CreatePayment(ctx context.Context, input *usecases.CreatePaymentInput) (*usecases.CreatePaymentOutput, error) {
	s.lastInput = input
	return s.out, s.err
}

func (s *createPaymentUsecaseStub) GetPayment(ctx context.Context, paymentID uuid.UUID) (*usecases.CreatePaymentOutput, error) {
	return s.out, s.err
}

func TestCreatePaymentHandler_ResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	out := &usecases.CreatePaymentOutput{
		PaymentID:                "0195-payment",
		MerchantID:               "0195-merchant",
		Amount:                   "2950000",
		InvoiceCurrency:          "IDRX",
		InvoiceAmount:            "50000",
		PayerSelectedChain:       "eip155:8453",
		PayerSelectedToken:       "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		PayerSelectedTokenSymbol: "USDC",
		QuotedTokenSymbol:        "USDC",
		QuotedTokenAmount:        "2.95",
		QuotedTokenAmountAtomic:  "2950000",
		QuotedTokenDecimals:      6,
		QuoteRate:                "1 IDRX = 0.000059 USDC",
		QuoteSource:              "uniswap-v4-base-usdc-idrx",
		QuoteExpiresAt:           time.Date(2026, 3, 20, 10, 20, 0, 0, time.UTC),
		DestChain:                "eip155:8453",
		DestToken:                "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		DestWallet:               "0xMerchantDestination",
		ExpireTime:               time.Date(2026, 3, 20, 10, 20, 0, 0, time.UTC),
		PaymentURL:               "https://pay.paymentkita.com/pay/0195-payment",
		PaymentCode:              "eyJ.mock",
	}
	out.PaymentInstruction.ChainID = "eip155:8453"
	out.PaymentInstruction.To = "0xGateway"
	out.PaymentInstruction.Value = "0"
	out.PaymentInstruction.Data = "0xafc93ccd"

	stub := &createPaymentUsecaseStub{out: out}
	handler := NewCreatePaymentHandler(stub)
	r := gin.New()
	r.POST("/api/v1/create-payment", func(c *gin.Context) {
		c.Set(middleware.MerchantIDKey, uuid.MustParse("0195d4b4-1e2c-7f2f-9aa1-123456789012"))
		handler.CreatePayment(c)
	})

	req, _ := http.NewRequest("POST", "/api/v1/create-payment", strings.NewReader(`{
		"chain_id":"eip155:8453",
		"selected_token":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		"pricing_type":"invoice_currency",
		"requested_amount":"50000",
		"expires_in":"unlimited"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "0195-payment", got["payment_id"])
	require.Equal(t, "0195-merchant", got["merchant_id"])
	require.Equal(t, "2950000", got["amount"])
	require.Equal(t, "IDRX", got["invoice_currency"])
	require.Equal(t, "50000", got["invoice_amount"])
	require.Equal(t, "eip155:8453", got["payer_selected_chain"])
	require.Equal(t, "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", got["payer_selected_token"])
	require.Equal(t, "USDC", got["payer_selected_token_symbol"])
	require.Equal(t, "USDC", got["quoted_token_symbol"])
	require.Equal(t, "2.95", got["quoted_token_amount"])
	require.Equal(t, "2950000", got["quoted_token_amount_atomic"])
	require.Equal(t, float64(6), got["quoted_token_decimals"])
	require.Equal(t, "eyJ.mock", got["payment_code"])
	require.NotEmpty(t, got["payment_instruction"])
	require.NotNil(t, stub.lastInput)
	require.Equal(t, "unlimited", stub.lastInput.ExpiresIn)
}

func TestCreatePaymentHandler_CreatePaymentAdmin_UsesPathMerchantID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	out := &usecases.CreatePaymentOutput{PaymentID: "0195-payment"}
	stub := &createPaymentUsecaseStub{out: out}
	handler := NewCreatePaymentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/admin/merchants/:id/create-payment", handler.CreatePaymentAdmin)

	merchantID := "0195d4b4-1e2c-7f2f-9aa1-123456789012"
	req, _ := http.NewRequest("POST", "/api/v1/admin/merchants/"+merchantID+"/create-payment", strings.NewReader(`{
		"merchant_id":"0195d4b4-1e2c-7f2f-9aa1-123456789012",
		"chain_id":"eip155:8453",
		"selected_token":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		"pricing_type":"invoice_currency",
		"requested_amount":"50000",
		"expires_in":"180"
	}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.lastInput)
	require.Equal(t, uuid.MustParse(merchantID), stub.lastInput.MerchantContextID)
	require.Equal(t, uuid.MustParse(merchantID), stub.lastInput.MerchantID)
	require.Equal(t, "180", stub.lastInput.ExpiresIn)
}

func TestCreatePaymentHandler_CreatePaymentAdmin_MerchantIDMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &createPaymentUsecaseStub{out: &usecases.CreatePaymentOutput{PaymentID: "0195-payment"}}
	handler := NewCreatePaymentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/admin/merchants/:id/create-payment", handler.CreatePaymentAdmin)

	req, _ := http.NewRequest("POST", "/api/v1/admin/merchants/0195d4b4-1e2c-7f2f-9aa1-123456789012/create-payment", strings.NewReader(`{
		"merchant_id":"0195d4b4-1e2c-7f2f-9aa1-000000000000",
		"chain_id":"eip155:8453",
		"selected_token":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		"pricing_type":"invoice_currency",
		"requested_amount":"50000"
	}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Nil(t, stub.lastInput)
}

func TestCreatePaymentHandler_CreatePaymentAdmin_InvalidPathMerchantID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &createPaymentUsecaseStub{out: &usecases.CreatePaymentOutput{PaymentID: "0195-payment"}}
	handler := NewCreatePaymentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/admin/merchants/:id/create-payment", handler.CreatePaymentAdmin)

	req, _ := http.NewRequest("POST", "/api/v1/admin/merchants/not-a-uuid/create-payment", strings.NewReader(`{
		"chain_id":"eip155:8453",
		"selected_token":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		"pricing_type":"invoice_currency",
		"requested_amount":"50000"
	}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Nil(t, stub.lastInput)
}

func TestCreatePaymentHandler_GetPayment_ResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	out := &usecases.CreatePaymentOutput{
		PaymentID:                "0195-payment",
		MerchantID:               "0195-merchant",
		Amount:                   "2950000",
		InvoiceCurrency:          "IDRX",
		InvoiceAmount:            "50000",
		PayerSelectedChain:       "eip155:8453",
		PayerSelectedToken:       "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		PayerSelectedTokenSymbol: "USDC",
		QuotedTokenSymbol:        "USDC",
		QuotedTokenAmount:        "2.95",
		QuotedTokenAmountAtomic:  "2950000",
		QuotedTokenDecimals:      6,
		QuoteRate:                "1 IDRX = 0.000059 USDC",
		QuoteSource:              "uniswap-v4-base-usdc-idrx",
		QuoteExpiresAt:           time.Date(2026, 3, 20, 10, 20, 0, 0, time.UTC),
		DestChain:                "eip155:8453",
		DestToken:                "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		DestWallet:               "0xMerchantDestination",
		ExpireTime:               time.Date(2026, 3, 20, 10, 20, 0, 0, time.UTC),
		PaymentURL:               "https://pay.paymentkita.com/pay/0195-payment",
		PaymentCode:              "eyJ.mock",
	}
	out.PaymentInstruction.ChainID = "eip155:8453"
	out.PaymentInstruction.To = "0xGateway"
	out.PaymentInstruction.Value = "0"
	out.PaymentInstruction.Data = "0xafc93ccd"

	handler := NewCreatePaymentHandler(&createPaymentUsecaseStub{out: out})
	r := gin.New()
	r.GET("/api/v1/create-payment/:id", handler.GetPayment)

	req, _ := http.NewRequest("GET", "/api/v1/create-payment/0195d4b4-1e2c-7f2f-9aa1-123456789012", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "0195-payment", got["payment_id"])
	require.Equal(t, "USDC", got["quoted_token_symbol"])
	require.Equal(t, "eyJ.mock", got["payment_code"])
}
