package usecases

import (
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"payment-kita.backend/internal/domain/entities"
)

func TestPaymentUsecase_BuildEvmPaymentHexWithInput_V2Selectors(t *testing.T) {
	u := &PaymentUsecase{}
	basePayment := &entities.Payment{
		SourceTokenAddress: "0x1111111111111111111111111111111111111111",
		DestTokenAddress:   "0x2222222222222222222222222222222222222222",
		ReceiverAddress:    "0x3333333333333333333333333333333333333333",
		SourceAmount:       "1000",
	}

	t.Run("regular explicit bridge option uses createPayment", func(t *testing.T) {
		mode := "regular"
		bridge := uint8(1)
		input := &entities.CreatePaymentInput{
			Mode:         &mode,
			BridgeOption: &bridge,
		}

		hexOut := u.buildEvmPaymentHexWithInput(basePayment, "eip155:137", big.NewInt(0), input)
		require.NotEmpty(t, hexOut)
		require.True(t, strings.HasPrefix(hexOut, CreatePaymentSelector))
	})

	t.Run("regular default bridge uses createPaymentDefaultBridge", func(t *testing.T) {
		mode := "regular"
		input := &entities.CreatePaymentInput{
			Mode:         &mode,
			BridgeOption: nil, // nullable => default bridge function
		}

		hexOut := u.buildEvmPaymentHexWithInput(basePayment, "eip155:137", big.NewInt(0), input)
		require.NotEmpty(t, hexOut)
		require.True(t, strings.HasPrefix(hexOut, CreatePaymentDefaultBridgeSelector))
	})

	t.Run("privacy uses createPaymentPrivate", func(t *testing.T) {
		mode := "privacy"
		intent := "order-12345"
		stealth := "0x4444444444444444444444444444444444444444"
		input := &entities.CreatePaymentInput{
			Mode:                   &mode,
			BridgeOption:           nil, // should still select private function
			PrivacyIntentID:        &intent,
			PrivacyStealthReceiver: &stealth,
		}

		hexOut := u.buildEvmPaymentHexWithInput(basePayment, "eip155:137", big.NewInt(0), input)
		require.NotEmpty(t, hexOut)
		require.True(t, strings.HasPrefix(hexOut, CreatePaymentPrivateSelector))
	})

	t.Run("invalid bridge option returns empty calldata", func(t *testing.T) {
		mode := "regular"
		bridge := uint8(9)
		input := &entities.CreatePaymentInput{
			Mode:         &mode,
			BridgeOption: &bridge,
		}

		hexOut := u.buildEvmPaymentHexWithInput(basePayment, "eip155:137", big.NewInt(0), input)
		require.Empty(t, hexOut)
	})

	t.Run("privacy without required fields returns empty calldata", func(t *testing.T) {
		mode := "privacy"
		input := &entities.CreatePaymentInput{
			Mode: &mode,
		}

		hexOut := u.buildEvmPaymentHexWithInput(basePayment, "eip155:137", big.NewInt(0), input)
		require.Empty(t, hexOut)
	})
}

