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
	// Legacy payment request flow selector (non-gateway usage may still reference it).
	PayRequestSelector = computeSelectorHex("payRequest(bytes32)")

	// V2 request tuple:
	// (bytes destChainIdBytes,bytes receiverBytes,address sourceToken,address bridgeTokenSource,address destToken,
	//  uint256 amountInSource,uint256 minBridgeAmountOut,uint256 minDestAmountOut,uint8 mode,uint8 bridgeOption)
	PaymentRequestV2Tuple = "(bytes,bytes,address,address,address,uint256,uint256,uint256,uint8,uint8)"

	// Private routing tuple: (bytes32 intentId,address stealthReceiver)
	PrivateRoutingTuple = "(bytes32,address)"

	// Final V2-only gateway selectors.
	CreatePaymentSelector              = computeSelectorHex("createPayment(" + PaymentRequestV2Tuple + ")")
	CreatePaymentPrivateSelector       = computeSelectorHex("createPaymentPrivate(" + PaymentRequestV2Tuple + "," + PrivateRoutingTuple + ")")
	CreatePaymentDefaultBridgeSelector = computeSelectorHex("createPaymentDefaultBridge(" + PaymentRequestV2Tuple + ")")
	PreviewApprovalSelector            = computeSelectorHex("previewApproval(" + PaymentRequestV2Tuple + ")")
	QuotePaymentCostSelector           = computeSelectorHex("quotePaymentCost(" + PaymentRequestV2Tuple + ")")
	DeployEscrowSelector               = computeSelectorHex("deployEscrow(bytes32,address,address)")

	// Backward-compat aliases for old constant names in tests/helpers.
	CreatePaymentV2Selector              = CreatePaymentSelector
	CreatePaymentPrivateV2Selector       = CreatePaymentPrivateSelector
	CreatePaymentV2DefaultBridgeSelector = CreatePaymentDefaultBridgeSelector
	PreviewApprovalV2Selector            = PreviewApprovalSelector
	QuotePaymentCostV2Selector           = QuotePaymentCostSelector
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
