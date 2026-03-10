package usecases

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"payment-kita.backend/internal/domain/entities"
)

func TestPreparePrivacyRouting_RejectsManualStealthMismatchWhenFactoryPredictionAvailable(t *testing.T) {
	original := predictEscrowStealthAddressFn
	t.Cleanup(func() {
		predictEscrowStealthAddressFn = original
	})

	expected := common.HexToAddress("0x1111111111111111111111111111111111111111")
	predictEscrowStealthAddressFn = func(_ string, _ common.Address) (common.Address, bool, error) {
		return expected, true, nil
	}

	mode := "privacy"
	intentID := "intent-abc"
	manualStealth := "0x2222222222222222222222222222222222222222"
	input := &entities.CreatePaymentAppInput{
		Mode:                   &mode,
		ReceiverAddress:        "0x3cE4b16B6761306dB79B2c4fb89106e3A3747550",
		PrivacyIntentID:        &intentID,
		PrivacyStealthReceiver: &manualStealth,
	}

	_, _, err := preparePrivacyRouting(input)
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "must match factory predicted escrow address") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreparePrivacyRouting_AutoFillsStealthFromFactoryPrediction(t *testing.T) {
	original := predictEscrowStealthAddressFn
	t.Cleanup(func() {
		predictEscrowStealthAddressFn = original
	})

	expected := common.HexToAddress("0x3333333333333333333333333333333333333333")
	predictEscrowStealthAddressFn = func(_ string, _ common.Address) (common.Address, bool, error) {
		return expected, true, nil
	}

	mode := "privacy"
	input := &entities.CreatePaymentAppInput{
		Mode:            &mode,
		ReceiverAddress: "0x3cE4b16B6761306dB79B2c4fb89106e3A3747550",
	}

	intentID, stealth, err := preparePrivacyRouting(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(intentID) == "" {
		t.Fatalf("intent id should be auto-generated")
	}
	if !strings.EqualFold(stealth, expected.Hex()) {
		t.Fatalf("unexpected stealth: got %s want %s", stealth, expected.Hex())
	}
}
