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

type partnerQuoteUsecaseStub struct {
	out *usecases.CreatePartnerQuoteOutput
	err error
}

func (s *partnerQuoteUsecaseStub) CreateQuote(ctx context.Context, input *usecases.CreatePartnerQuoteInput) (*usecases.CreatePartnerQuoteOutput, error) {
	return s.out, s.err
}

func TestPartnerQuoteHandler_CreateQuote_ResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewPartnerQuoteHandler(&partnerQuoteUsecaseStub{
		out: &usecases.CreatePartnerQuoteOutput{
			QuoteID:             "0195-test",
			InvoiceCurrency:     "IDRX",
			InvoiceAmount:       "50000000000",
			SelectedChain:       "eip155:8453",
			SelectedToken:       "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
			SelectedTokenSymbol: "USDC",
			QuotedAmount:        "2950000",
			QuoteDecimals:       6,
			QuoteRate:           "0.000059",
			PriceSource:         "uniswap-v4-base-usdc-idrx",
			Route:               "IDRX->USDC",
			SlippageBps:         100,
			RateTimestamp:       time.Date(2026, 3, 20, 10, 15, 0, 0, time.UTC),
			QuoteExpiresAt:      time.Date(2026, 3, 20, 10, 20, 0, 0, time.UTC),
		},
	})

	r := gin.New()
	r.POST("/api/v1/partner/quotes", func(c *gin.Context) {
		c.Set(middleware.MerchantIDKey, uuid.MustParse("0195d4b4-1e2c-7f2f-9aa1-123456789012"))
		handler.CreateQuote(c)
	})

	req, _ := http.NewRequest("POST", "/api/v1/partner/quotes", strings.NewReader(`{
			"invoice_currency":"IDRX",
			"invoice_amount":"50000000000",
			"selected_chain":"eip155:8453",
			"selected_token":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
			"dest_wallet":"0xMerchantDestination"
		}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "0195-test", got["quote_id"])
	require.Equal(t, "IDRX", got["invoice_currency"])
	require.Equal(t, "50000000000", got["invoice_amount"])
	require.Equal(t, "eip155:8453", got["selected_chain"])
	require.Equal(t, "USDC", got["selected_token_symbol"])
	require.Equal(t, "2950000", got["quoted_amount"])
	require.Equal(t, float64(6), got["quote_decimals"])
	require.Equal(t, "0.000059", got["quote_rate"])
	require.Equal(t, "uniswap-v4-base-usdc-idrx", got["price_source"])
	require.Equal(t, "IDRX->USDC", got["route"])
	require.Equal(t, float64(100), got["slippage_bps"])
	require.NotEmpty(t, got["rate_timestamp"])
	require.NotEmpty(t, got["quote_expires_at"])
}
