package usecases

import "time"

// EVM Function Selectors
// These are the first 4 bytes of the Keccak-256 hash of the function signature.

// CreatePaymentSelector is for createPayment(bytes destChainId, bytes receiver, address sourceToken, address destToken, uint256 amount)
// keccak256("createPayment(bytes,bytes,address,address,uint256)") -> 0x83f7cae3
const CreatePaymentSelector = "0x83f7cae3"

// PayRequestSelector is for payRequest(bytes32 requestId)
// keccak256("payRequest(bytes32)") -> 0x87b13db6
const PayRequestSelector = "0x87b13db6"

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
