package usecases

import (
	"testing"

	"payment-kita.backend/internal/domain/entities"
)

func TestParseOnchainPaymentID(t *testing.T) {
	valid := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	got, normalized, err := parseOnchainPaymentID(valid)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if normalized != valid {
		t.Fatalf("expected normalized %s, got %s", valid, normalized)
	}
	if got[0] != 0xaa || got[31] != 0xaa {
		t.Fatalf("unexpected parsed bytes")
	}

	if _, _, err := parseOnchainPaymentID("0x1234"); err == nil {
		t.Fatalf("expected error for invalid bytes32 length")
	}
}

func TestResolveOnchainPaymentID_FromEventMetadata(t *testing.T) {
	events := []*entities.PaymentEvent{
		{
			EventType: entities.PaymentEventType("PAYMENT_FAILED"),
			Metadata: map[string]interface{}{
				"reason": "PRIVACY_FORWARD_FAILED",
				"debug": map[string]interface{}{
					"onchainPaymentId": "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
			},
		},
	}

	parsed, normalized, err := resolveOnchainPaymentID(events, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if normalized != "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("unexpected normalized id: %s", normalized)
	}
	if parsed[0] != 0xbb || parsed[31] != 0xbb {
		t.Fatalf("unexpected parsed bytes")
	}
}

func TestPrivacyRecoveryMethodSpec(t *testing.T) {
	methodSig, selector, err := privacyRecoveryMethodSpec(entities.PrivacyRecoveryActionRetry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if methodSig == "" || selector == "" {
		t.Fatalf("expected non-empty method and selector")
	}
	if methodSig != "retryPrivacyForward(bytes32)" {
		t.Fatalf("unexpected method signature: %s", methodSig)
	}
}

func TestValidatePrivacyRecoveryStage(t *testing.T) {
	if err := validatePrivacyRecoveryStage(entities.PrivacyLifecycleClaimable, entities.PrivacyRecoveryActionClaim); err != nil {
		t.Fatalf("expected claimable stage to allow claim: %v", err)
	}

	if err := validatePrivacyRecoveryStage(entities.PrivacyLifecycleForwardedFinal, entities.PrivacyRecoveryActionRefund); err == nil {
		t.Fatalf("expected refund on forwarded final stage to fail")
	}
}
