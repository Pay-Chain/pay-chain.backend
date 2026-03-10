package usecases

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectorComputation(t *testing.T) {
	t.Run("PayRequestSelector matches expected value", func(t *testing.T) {
		assert.Equal(t, "0x87b13db6", PayRequestSelector,
			"payRequest(bytes32) selector mismatch")
	})

	t.Run("computeSelectorHex produces correct format", func(t *testing.T) {
		sel := computeSelectorHex("transfer(address,uint256)")
		assert.Equal(t, "0xa9059cbb", sel, "well-known ERC20 transfer selector")
	})

	t.Run("CreatePaymentSelector is computed from tuple signature", func(t *testing.T) {
		expected := computeSelectorHex("createPayment((bytes,bytes,address,address,address,uint256,uint256,uint256,uint8,uint8))")
		assert.Equal(t, expected, CreatePaymentSelector)
	})

	t.Run("CreatePaymentPrivateSelector is computed from tuple signature", func(t *testing.T) {
		expected := computeSelectorHex("createPaymentPrivate((bytes,bytes,address,address,address,uint256,uint256,uint256,uint8,uint8),(bytes32,address))")
		assert.Equal(t, expected, CreatePaymentPrivateSelector)
	})

	t.Run("CreatePaymentDefaultBridgeSelector is computed from tuple signature", func(t *testing.T) {
		expected := computeSelectorHex("createPaymentDefaultBridge((bytes,bytes,address,address,address,uint256,uint256,uint256,uint8,uint8))")
		assert.Equal(t, expected, CreatePaymentDefaultBridgeSelector)
	})

	t.Run("PreviewApprovalSelector is computed from tuple signature", func(t *testing.T) {
		expected := computeSelectorHex("previewApproval((bytes,bytes,address,address,address,uint256,uint256,uint256,uint8,uint8))")
		assert.Equal(t, expected, PreviewApprovalSelector)
	})

	t.Run("QuotePaymentCostSelector is computed from tuple signature", func(t *testing.T) {
		expected := computeSelectorHex("quotePaymentCost((bytes,bytes,address,address,address,uint256,uint256,uint256,uint8,uint8))")
		assert.Equal(t, expected, QuotePaymentCostSelector)
	})

	t.Run("DeployEscrowSelector is computed from signature", func(t *testing.T) {
		expected := computeSelectorHex("deployEscrow(bytes32,address,address)")
		assert.Equal(t, expected, DeployEscrowSelector)
	})
}
