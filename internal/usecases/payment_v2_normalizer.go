package usecases

import (
	"fmt"
	"strings"
)

const (
	PaymentModeRegular = "regular"
	PaymentModePrivacy = "privacy"

	BridgeOptionDefaultSentinel uint8 = 255
	BridgeOptionHyperbridge     uint8 = 0
	BridgeOptionCCIP            uint8 = 1
	BridgeOptionLayerZero       uint8 = 2
)

func normalizePaymentMode(mode *string) string {
	if mode == nil {
		return PaymentModeRegular
	}
	if strings.EqualFold(strings.TrimSpace(*mode), PaymentModePrivacy) {
		return PaymentModePrivacy
	}
	return PaymentModeRegular
}

func normalizeBridgeOption(option *uint8) (uint8, error) {
	if option == nil {
		return BridgeOptionDefaultSentinel, nil
	}
	if *option > BridgeOptionLayerZero {
		return 0, fmt.Errorf("invalid bridgeOption: %d", *option)
	}
	return *option, nil
}

func validatePrivacyFields(mode string, intentID, stealthReceiver *string) error {
	if mode != PaymentModePrivacy {
		return nil
	}
	if intentID == nil || strings.TrimSpace(*intentID) == "" {
		return fmt.Errorf("privacyIntentId is required when mode=privacy")
	}
	if stealthReceiver == nil || strings.TrimSpace(*stealthReceiver) == "" {
		return fmt.Errorf("privacyStealthReceiver is required when mode=privacy")
	}
	return nil
}
