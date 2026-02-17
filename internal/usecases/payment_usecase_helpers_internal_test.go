package usecases

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
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

func TestPaymentUsecase_BuildHex_HookErrorBranches(t *testing.T) {
	origNewType := newABIType
	origPack := packABIArgs
	t.Cleanup(func() {
		newABIType = origNewType
		packABIArgs = origPack
	})

	u := &PaymentUsecase{}

	t.Run("approve hex new type error", func(t *testing.T) {
		newABIType = func(string, string, []abi.ArgumentMarshaling) (abi.Type, error) {
			return abi.Type{}, errors.New("type failed")
		}
		assert.Empty(t, u.buildErc20ApproveHex("0x1234567890123456789012345678901234567890", "1000"))
	})

	t.Run("approve hex pack error", func(t *testing.T) {
		newABIType = origNewType
		packABIArgs = func(abi.Arguments, ...interface{}) ([]byte, error) {
			return nil, errors.New("pack failed")
		}
		assert.Empty(t, u.buildErc20ApproveHex("0x1234567890123456789012345678901234567890", "1000"))
	})

	t.Run("approve hex second new type error", func(t *testing.T) {
		call := 0
		newABIType = func(string, string, []abi.ArgumentMarshaling) (abi.Type, error) {
			call++
			if call == 2 {
				return abi.Type{}, errors.New("second type failed")
			}
			return origNewType("address", "", nil)
		}
		assert.Empty(t, u.buildErc20ApproveHex("0x1234567890123456789012345678901234567890", "1000"))
	})

	t.Run("evm payment hex new type error", func(t *testing.T) {
		newABIType = func(string, string, []abi.ArgumentMarshaling) (abi.Type, error) {
			return abi.Type{}, errors.New("type failed")
		}
		assert.Empty(t, u.buildEvmPaymentHex(&entities.Payment{
			SourceTokenAddress: "0x1111111111111111111111111111111111111111",
			DestTokenAddress:   "0x2222222222222222222222222222222222222222",
			ReceiverAddress:    "0x3333333333333333333333333333333333333333",
			SourceAmount:       "1000",
		}, "eip155:8453"))
	})

	t.Run("evm payment hex second and third new type errors", func(t *testing.T) {
		basePayment := &entities.Payment{
			SourceTokenAddress: "0x1111111111111111111111111111111111111111",
			DestTokenAddress:   "0x2222222222222222222222222222222222222222",
			ReceiverAddress:    "0x3333333333333333333333333333333333333333",
			SourceAmount:       "1000",
		}

		call := 0
		newABIType = func(string, string, []abi.ArgumentMarshaling) (abi.Type, error) {
			call++
			switch call {
			case 1:
				return origNewType("bytes", "", nil)
			case 2:
				return abi.Type{}, errors.New("address type failed")
			default:
				return origNewType("uint256", "", nil)
			}
		}
		assert.Empty(t, u.buildEvmPaymentHex(basePayment, "eip155:8453"))

		call = 0
		newABIType = func(string, string, []abi.ArgumentMarshaling) (abi.Type, error) {
			call++
			switch call {
			case 1:
				return origNewType("bytes", "", nil)
			case 2:
				return origNewType("address", "", nil)
			case 3:
				return abi.Type{}, errors.New("uint type failed")
			default:
				return origNewType("bytes", "", nil)
			}
		}
		assert.Empty(t, u.buildEvmPaymentHex(basePayment, "eip155:8453"))
	})

	t.Run("evm payment hex receiver pack fail fallback and final pack fail", func(t *testing.T) {
		newABIType = origNewType
		basePayment := &entities.Payment{
			SourceTokenAddress: "0x1111111111111111111111111111111111111111",
			DestTokenAddress:   "0x2222222222222222222222222222222222222222",
			ReceiverAddress:    "0x3333333333333333333333333333333333333333",
			SourceAmount:       "1000",
		}

		call := 0
		packABIArgs = func(args abi.Arguments, values ...interface{}) ([]byte, error) {
			call++
			// First call is receiver-address encoding path; fail to hit fallback bytes path.
			if call == 1 {
				return nil, errors.New("receiver encode failed")
			}
			return origPack(args, values...)
		}
		hexOut := u.buildEvmPaymentHex(basePayment, "eip155:8453")
		assert.NotEmpty(t, hexOut)

		packABIArgs = func(abi.Arguments, ...interface{}) ([]byte, error) {
			return nil, errors.New("final pack failed")
		}
		assert.Empty(t, u.buildEvmPaymentHex(basePayment, "eip155:8453"))
	})
}
