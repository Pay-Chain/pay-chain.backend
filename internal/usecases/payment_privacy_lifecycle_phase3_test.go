package usecases

import (
	"testing"

	"payment-kita.backend/internal/domain/entities"
)

func TestDerivePaymentPrivacyLifecycle_ClaimedIsResolved(t *testing.T) {
	payment := &entities.Payment{Status: entities.PaymentStatusCompleted}
	events := []*entities.PaymentEvent{
		{EventType: entities.PaymentEventType("PRIVACY_ESCROW_CLAIMED")},
	}

	stage, isPrivacy, _, _ := derivePaymentPrivacyLifecycle(payment, events)
	if stage != entities.PrivacyLifecycleResolved {
		t.Fatalf("expected %s, got %s", entities.PrivacyLifecycleResolved, stage)
	}
	if !isPrivacy {
		t.Fatalf("expected privacy candidate true")
	}
}

func TestDerivePaymentPrivacyLifecycle_RefundedIsResolved(t *testing.T) {
	payment := &entities.Payment{Status: entities.PaymentStatusFailed}
	events := []*entities.PaymentEvent{
		{EventType: entities.PaymentEventType("PRIVACY_ESCROW_REFUNDED")},
	}

	stage, isPrivacy, _, _ := derivePaymentPrivacyLifecycle(payment, events)
	if stage != entities.PrivacyLifecycleResolved {
		t.Fatalf("expected %s, got %s", entities.PrivacyLifecycleResolved, stage)
	}
	if !isPrivacy {
		t.Fatalf("expected privacy candidate true")
	}
}

func TestDerivePaymentPrivacyLifecycle_ForwardFailureIsClaimableOrRefundable(t *testing.T) {
	failedPayment := &entities.Payment{Status: entities.PaymentStatusFailed}
	events := []*entities.PaymentEvent{
		{EventType: entities.PaymentEventType("PRIVACY_FORWARD_FAILED")},
	}

	stage, isPrivacy, _, _ := derivePaymentPrivacyLifecycle(failedPayment, events)
	if stage != entities.PrivacyLifecycleRefundable {
		t.Fatalf("expected %s, got %s", entities.PrivacyLifecycleRefundable, stage)
	}
	if !isPrivacy {
		t.Fatalf("expected privacy candidate true")
	}

	processingPayment := &entities.Payment{Status: entities.PaymentStatusProcessing}
	stage, isPrivacy, _, _ = derivePaymentPrivacyLifecycle(processingPayment, events)
	if stage != entities.PrivacyLifecycleClaimable {
		t.Fatalf("expected %s, got %s", entities.PrivacyLifecycleClaimable, stage)
	}
	if !isPrivacy {
		t.Fatalf("expected privacy candidate true")
	}
}
