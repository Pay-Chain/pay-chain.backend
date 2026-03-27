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

type partnerPaymentSessionUsecaseStub struct {
	out        *usecases.CreatePartnerPaymentSessionOutput
	getOut     *usecases.GetPartnerPaymentSessionOutput
	resolveOut *usecases.ResolvePartnerPaymentCodeOutput
	err        error
}

func (s *partnerPaymentSessionUsecaseStub) CreateSession(ctx context.Context, input *usecases.CreatePartnerPaymentSessionInput) (*usecases.CreatePartnerPaymentSessionOutput, error) {
	return s.out, s.err
}

func (s *partnerPaymentSessionUsecaseStub) GetSession(ctx context.Context, sessionID uuid.UUID) (*usecases.GetPartnerPaymentSessionOutput, error) {
	return s.getOut, s.err
}

func (s *partnerPaymentSessionUsecaseStub) ResolvePaymentCode(ctx context.Context, input *usecases.ResolvePartnerPaymentCodeInput) (*usecases.ResolvePartnerPaymentCodeOutput, error) {
	return s.resolveOut, s.err
}

func TestPartnerPaymentSessionHandler_CreateSession_ResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	out := &usecases.CreatePartnerPaymentSessionOutput{
		PaymentID:       "0195-payment",
		MerchantID:      "0195-merchant",
		InvoiceCurrency: "IDRX",
		InvoiceAmount:   "50000000000",
		Amount:          "2950000",
		AmountDecimals:  6,
		DestChain:       "eip155:8453",
		DestToken:       "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		DestWallet:      "0xMerchantDestination",
		ExpireTime:      time.Date(2026, 3, 20, 10, 20, 0, 0, time.UTC),
		PaymentURL:      "https://pay.paymentkita.com/pay/0195-payment",
		PaymentCode:     "eyJ.mock",
		Status:          "PENDING",
	}
	out.PaymentInstruction.ChainID = "eip155:8453"
	out.PaymentInstruction.To = "0xGateway"
	out.PaymentInstruction.Value = "0"
	out.PaymentInstruction.Data = "0xafc93ccd"
	out.Quote.QuoteID = "0195-quote"
	out.Quote.PriceSource = "uniswap-v4-base-usdc-idrx"
	out.Quote.QuoteRate = "0.000059"
	out.Quote.QuoteExpiresAt = time.Date(2026, 3, 20, 10, 20, 0, 0, time.UTC)

	handler := NewPartnerPaymentSessionHandler(&partnerPaymentSessionUsecaseStub{out: out}, nil, nil)

	r := gin.New()
	r.POST("/api/v1/partner/payment-sessions", func(c *gin.Context) {
		c.Set(middleware.MerchantIDKey, uuid.MustParse("0195d4b4-1e2c-7f2f-9aa1-123456789012"))
		handler.CreateSession(c)
	})

	req, _ := http.NewRequest("POST", "/api/v1/partner/payment-sessions", strings.NewReader(`{
			"quote_id":"0195d4b4-1e2c-7f2f-9aa1-123456789012",
			"dest_wallet":"0xMerchantDestination"
		}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "0195-payment", got["payment_id"])
	require.Equal(t, "0195-merchant", got["merchant_id"])
	require.Equal(t, "IDRX", got["invoice_currency"])
	require.Equal(t, "50000000000", got["invoice_amount"])
	require.Equal(t, "2950000", got["amount"])
	require.Equal(t, float64(6), got["amount_decimals"])
	require.Equal(t, "eip155:8453", got["dest_chain"])
	require.Equal(t, "0xMerchantDestination", got["dest_wallet"])
	require.Equal(t, "https://pay.paymentkita.com/pay/0195-payment", got["payment_url"])
	require.Equal(t, "eyJ.mock", got["payment_code"])
	require.Equal(t, "PENDING", got["status"])
	require.NotEmpty(t, got["payment_instruction"])
	require.NotEmpty(t, got["quote"])
}

func TestPartnerPaymentSessionHandler_GetSession_ResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	getOut := &usecases.GetPartnerPaymentSessionOutput{
		PaymentID:      "0195-payment",
		Status:         "PENDING",
		Amount:         "2950000",
		AmountDecimals: 6,
		DestChain:      "eip155:8453",
		DestToken:      "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		DestWallet:     "0xMerchantDestination",
		ExpiresAt:      time.Date(2026, 3, 20, 10, 20, 0, 0, time.UTC),
		PaymentURL:     "https://pay.paymentkita.com/pay/0195-payment",
		PaymentCode:    "eyJ.mock",
	}
	getOut.PaymentInstruction.ChainID = "eip155:8453"
	getOut.PaymentInstruction.To = "0xGateway"
	getOut.PaymentInstruction.Value = "0"
	getOut.PaymentInstruction.Data = "0xafc93ccd"

	handler := NewPartnerPaymentSessionHandler(&partnerPaymentSessionUsecaseStub{getOut: getOut}, nil, nil)

	r := gin.New()
	r.GET("/api/v1/partner/payment-sessions/:id", handler.GetSession)

	req, _ := http.NewRequest("GET", "/api/v1/partner/payment-sessions/0195d4b4-1e2c-7f2f-9aa1-123456789012", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "0195-payment", got["payment_id"])
	require.Equal(t, "PENDING", got["status"])
	require.Equal(t, "2950000", got["amount"])
	require.Equal(t, float64(6), got["amount_decimals"])
	require.NotEmpty(t, got["payment_instruction"])
}

func TestPartnerPaymentSessionHandler_ResolvePaymentCode_ResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resolveOut := &usecases.ResolvePartnerPaymentCodeOutput{
		PaymentID:      "0195-payment",
		MerchantID:     "0195-merchant",
		Amount:         "2950000",
		AmountDecimals: 6,
		DestChain:      "eip155:8453",
		DestToken:      "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		DestWallet:     "0xMerchantDestination",
		ExpiresAt:      time.Date(2026, 3, 20, 10, 20, 0, 0, time.UTC),
	}
	resolveOut.PaymentInstruction.ChainID = "eip155:8453"
	resolveOut.PaymentInstruction.To = "0xGateway"
	resolveOut.PaymentInstruction.Value = "0"
	resolveOut.PaymentInstruction.Data = "0xafc93ccd"

	handler := NewPartnerPaymentSessionHandler(&partnerPaymentSessionUsecaseStub{resolveOut: resolveOut}, nil, nil)

	r := gin.New()
	r.POST("/api/v1/partner/payment-sessions/resolve-code", handler.ResolvePaymentCode)

	req, _ := http.NewRequest("POST", "/api/v1/partner/payment-sessions/resolve-code", strings.NewReader(`{"payment_code":"eyJ.mock"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "0195-payment", got["payment_id"])
	require.Equal(t, "0195-merchant", got["merchant_id"])
	require.Equal(t, "2950000", got["amount"])
	require.Equal(t, float64(6), got["amount_decimals"])
	require.NotEmpty(t, got["payment_instruction"])
}
