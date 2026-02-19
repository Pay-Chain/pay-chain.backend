package usecases

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectorComputation(t *testing.T) {
	t.Run("CreatePaymentSelector matches expected value", func(t *testing.T) {
		assert.Equal(t, "0x83f7cae3", CreatePaymentSelector,
			"createPayment(bytes,bytes,address,address,uint256) selector mismatch")
	})

	t.Run("PayRequestSelector matches expected value", func(t *testing.T) {
		assert.Equal(t, "0x87b13db6", PayRequestSelector,
			"payRequest(bytes32) selector mismatch")
	})

	t.Run("CreatePaymentWithSlippageSelector is computed", func(t *testing.T) {
		assert.NotEmpty(t, CreatePaymentWithSlippageSelector)
		assert.True(t, len(CreatePaymentWithSlippageSelector) == 10, // "0x" + 8 hex chars
			"selector should be 10 characters (0x + 4 bytes hex)")
	})

	t.Run("computeSelectorHex produces correct format", func(t *testing.T) {
		sel := computeSelectorHex("transfer(address,uint256)")
		assert.Equal(t, "0xa9059cbb", sel, "well-known ERC20 transfer selector")
	})
}
