package usecases

import "testing"

func TestNormalizePaymentMode(t *testing.T) {
	t.Run("nil defaults to regular", func(t *testing.T) {
		got := normalizePaymentMode(nil)
		if got != PaymentModeRegular {
			t.Fatalf("expected %s, got %s", PaymentModeRegular, got)
		}
	})

	t.Run("privacy is normalized case-insensitive", func(t *testing.T) {
		mode := "PrIvAcY"
		got := normalizePaymentMode(&mode)
		if got != PaymentModePrivacy {
			t.Fatalf("expected %s, got %s", PaymentModePrivacy, got)
		}
	})
}

func TestNormalizeBridgeOption(t *testing.T) {
	t.Run("nil uses sentinel", func(t *testing.T) {
		got, err := normalizeBridgeOption(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != BridgeOptionDefaultSentinel {
			t.Fatalf("expected %d, got %d", BridgeOptionDefaultSentinel, got)
		}
	})

	t.Run("valid options pass", func(t *testing.T) {
		for _, option := range []uint8{BridgeOptionHyperbridge, BridgeOptionCCIP, BridgeOptionStargate, BridgeOptionHBTokenGateway} {
			option := option
			got, err := normalizeBridgeOption(&option)
			if err != nil {
				t.Fatalf("unexpected error for %d: %v", option, err)
			}
			if got != option {
				t.Fatalf("expected %d, got %d", option, got)
			}
		}
	})

	t.Run("invalid option rejects", func(t *testing.T) {
		option := uint8(9)
		_, err := normalizeBridgeOption(&option)
		if err == nil {
			t.Fatalf("expected error for invalid bridge option")
		}
	})
}

func TestValidatePrivacyFields(t *testing.T) {
	t.Run("regular ignores privacy fields", func(t *testing.T) {
		if err := validatePrivacyFields(PaymentModeRegular, nil, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("privacy requires intent and stealth receiver", func(t *testing.T) {
		if err := validatePrivacyFields(PaymentModePrivacy, nil, nil); err == nil {
			t.Fatalf("expected error for missing privacy fields")
		}

		intent := "intent-1"
		if err := validatePrivacyFields(PaymentModePrivacy, &intent, nil); err == nil {
			t.Fatalf("expected error for missing stealth receiver")
		}

		stealth := "0xabc"
		if err := validatePrivacyFields(PaymentModePrivacy, nil, &stealth); err == nil {
			t.Fatalf("expected error for missing intent")
		}
	})

	t.Run("privacy with complete fields passes", func(t *testing.T) {
		intent := "intent-1"
		stealth := "0xabc"
		if err := validatePrivacyFields(PaymentModePrivacy, &intent, &stealth); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
