package usecases

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
)

func TestPaymentUsecase_BuildEvmPaymentHex_SlippageSelector(t *testing.T) {
	u := &PaymentUsecase{}

	p := &entities.Payment{
		SourceTokenAddress: "0x1111111111111111111111111111111111111111",
		DestTokenAddress:   "0x2222222222222222222222222222222222222222",
		ReceiverAddress:    "0x3333333333333333333333333333333333333333",
		SourceAmount:       "1000",
	}

	// Case 1: No slippage (minAmountOut == 0) -> Standard CreatePaymentSelector
	hex1 := u.buildEvmPaymentHex(p, "eip155:42161", big.NewInt(0))
	assert.NotEmpty(t, hex1)
	// CreatePaymentSelector = 0x83f7cae3
	assert.True(t, strings.HasPrefix(hex1, "0x83f7cae3"), "Expected standard selector 0x83f7cae3, got %s", hex1[:10])

	// Case 2: With slippage (minAmountOut > 0) -> CreatePaymentWithSlippageSelector
	hex2 := u.buildEvmPaymentHex(p, "eip155:42161", big.NewInt(900))
	assert.NotEmpty(t, hex2)
	// CreatePaymentWithSlippageSelector = 0xb28c3d9b
	assert.True(t, strings.HasPrefix(hex2, "0xb28c3d9b"), "Expected slippage selector 0xb28c3d9b, got %s", hex2[:10])

	// Verify length difference (slippage hex should be longer by 32 bytes for minAmountOut)
	bytes1, _ := hexutil.Decode(hex1)
	bytes2, _ := hexutil.Decode(hex2)
	assert.Equal(t, len(bytes1)+32, len(bytes2), "Slippage payload should have one more uint256 argument")
}
