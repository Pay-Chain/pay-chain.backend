package usecases

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"payment-kita.backend/internal/domain/entities"
)

func TestPaymentUsecase_BuildEvmPaymentHex_SlippageSelector(t *testing.T) {
	u := &PaymentUsecase{}

	p := &entities.Payment{
		SourceTokenAddress: "0x1111111111111111111111111111111111111111",
		DestTokenAddress:   "0x2222222222222222222222222222222222222222",
		ReceiverAddress:    "0x3333333333333333333333333333333333333333",
		SourceAmount:       "1000",
	}

	// Case 1: No slippage (minAmountOut == 0)
	hex1 := u.buildEvmPaymentHex(p, "eip155:42161", big.NewInt(0))
	assert.NotEmpty(t, hex1)
	assert.True(t, strings.HasPrefix(hex1, CreatePaymentSelector), "Expected selector %s, got %s", CreatePaymentSelector, hex1[:10])

	// Case 2: With slippage (minAmountOut > 0) still uses V2 createPayment selector.
	hex2 := u.buildEvmPaymentHex(p, "eip155:42161", big.NewInt(900))
	assert.NotEmpty(t, hex2)
	assert.True(t, strings.HasPrefix(hex2, CreatePaymentSelector), "Expected selector %s, got %s", CreatePaymentSelector, hex2[:10])

	// V2 payload size should be stable because minDestAmountOut is always part of tuple.
	bytes1, _ := hexutil.Decode(hex1)
	bytes2, _ := hexutil.Decode(hex2)
	assert.Equal(t, len(bytes1), len(bytes2), "V2 payload length should be consistent")
}
