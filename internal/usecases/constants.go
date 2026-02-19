package usecases

import (
	"encoding/hex"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// computeSelectorHex computes the 4-byte EVM function selector from a canonical
// function signature and returns it as a "0x"-prefixed hex string.
func computeSelectorHex(sig string) string {
	return "0x" + hex.EncodeToString(crypto.Keccak256([]byte(sig))[:4])
}

// EVM Function Selectors — computed at init from canonical signatures.
var (
	// createPayment(bytes,bytes,address,address,uint256) -> 0x83f7cae3
	CreatePaymentSelector = computeSelectorHex("createPayment(bytes,bytes,address,address,uint256)")

	// createPaymentWithSlippage(bytes,bytes,address,address,uint256,uint256) — for Fix #3
	CreatePaymentWithSlippageSelector = computeSelectorHex("createPaymentWithSlippage(bytes,bytes,address,address,uint256,uint256)")

	// payRequest(bytes32) -> 0x87b13db6
	PayRequestSelector = computeSelectorHex("payRequest(bytes32)")
)

// Fee configuration
const DefaultPercentageFee = 0.003 // 0.3%
const DefaultFixedFeeUSD = 0.50    // $0.50
const DefaultBridgeFeeFlat = 0.10  // $0.10

// Expiry durations
const PaymentExpiryDuration = 1 * time.Hour
const PaymentRequestExpiryMinutes = 15

// EVM Technical Constants
const EVMWordSize = 32
const EVMWordSizeHex = 64
