package usecases

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
)

func TestPaymentUsecase_Helpers(t *testing.T) {
	u := &PaymentUsecase{}

	assert.False(t, u.shouldRequireEvmApproval(""))
	assert.False(t, u.shouldRequireEvmApproval("native"))
	assert.False(t, u.shouldRequireEvmApproval("0x0000000000000000000000000000000000000000"))
	assert.True(t, u.shouldRequireEvmApproval("0x1234567890123456789012345678901234567890"))

	assert.NotEmpty(t, u.buildErc20ApproveHex("0x1234567890123456789012345678901234567890", "1000"))
	assert.Empty(t, u.buildErc20ApproveHex("0x1234567890123456789012345678901234567890", "invalid"))

	assert.Equal(t, "Hyperbridge", bridgeTypeToName(0))
	assert.Equal(t, "CCIP", bridgeTypeToName(1))
	assert.Equal(t, "LayerZero", bridgeTypeToName(2))
	assert.Equal(t, "Hyperbridge", bridgeTypeToName(99))

	assert.Equal(t, uint8(1), bridgeNameToType("CCIP"))
	assert.Equal(t, uint8(2), bridgeNameToType("LayerZero"))
	assert.Equal(t, uint8(0), bridgeNameToType("Hyperbridge"))
	assert.Equal(t, uint8(0), bridgeNameToType("unknown"))
}

func TestPaymentUsecase_BuildTransactionPayloadHelpers(t *testing.T) {
	u := &PaymentUsecase{}
	sourceChainID := uuid.New()
	destChainID := uuid.New()

	p := &entities.Payment{
		ID:                 uuid.New(),
		SourceChainID:      sourceChainID,
		DestChainID:        destChainID,
		SourceTokenAddress: "0x1111111111111111111111111111111111111111",
		DestTokenAddress:   "0x2222222222222222222222222222222222222222",
		ReceiverAddress:    "0x3333333333333333333333333333333333333333",
		SourceAmount:       "1000",
		DestChain:          &entities.Chain{ChainID: "42161", Type: entities.ChainTypeEVM},
	}

	evmHex := u.buildEvmPaymentHex(p, "eip155:42161")
	assert.NotEmpty(t, evmHex)
	assert.Contains(t, evmHex, "0x")

	svmBase58 := u.buildSvmPaymentBase58(p)
	assert.NotEmpty(t, svmBase58)
}
