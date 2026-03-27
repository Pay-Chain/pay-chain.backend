package usecases

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSimulatorQuoteResponse_TopLevel(t *testing.T) {
	amountOut, priceSource, routeSummary, err := parseSimulatorQuoteResponse([]byte(`{
		"amount_out":"2950000",
		"price_source":"simulator-v1",
		"route":"USDC->IDRT"
	}`))
	require.NoError(t, err)
	require.NotNil(t, amountOut)
	require.Equal(t, "2950000", amountOut.String())
	require.Equal(t, "simulator-v1", priceSource)
	require.Equal(t, "USDC->IDRT", routeSummary)
}

func TestParseSimulatorQuoteResponse_NestedData(t *testing.T) {
	amountOut, priceSource, routeSummary, err := parseSimulatorQuoteResponse([]byte(`{
		"data":{
			"amountOut":"7429000"
		}
	}`))
	require.NoError(t, err)
	require.NotNil(t, amountOut)
	require.Equal(t, "7429000", amountOut.String())
	require.Equal(t, "simulator-fallback", priceSource)
	require.Equal(t, "", routeSummary)
}

func TestParseSimulatorQuoteResponse_Invalid(t *testing.T) {
	_, _, _, err := parseSimulatorQuoteResponse([]byte(`{"price_source":"x"}`))
	require.Error(t, err)
}
