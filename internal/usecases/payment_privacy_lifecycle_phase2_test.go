package usecases

import (
	"testing"

	"payment-kita.backend/internal/domain/entities"
)

func TestDerivePaymentPrivacyLifecycle_ForwardRequestedMeansSettledToStealth(t *testing.T) {
	payment := &entities.Payment{
		Status: entities.PaymentStatusProcessing,
	}
	events := []*entities.PaymentEvent{
		{EventType: entities.PaymentEventType("PRIVACY_PAYMENT_CREATED")},
		{EventType: entities.PaymentEventType("PRIVACY_FORWARD_REQUESTED")},
	}

	stage, isPrivacy, _, _ := derivePaymentPrivacyLifecycle(payment, events)
	if stage != entities.PrivacyLifecycleSettledToStealth {
		t.Fatalf("expected %s, got %s", entities.PrivacyLifecycleSettledToStealth, stage)
	}
	if !isPrivacy {
		t.Fatalf("expected privacy candidate true")
	}
}

func TestDerivePaymentPrivacyLifecycle_CompletedWithoutForwardCompletedStaysSettled(t *testing.T) {
	payment := &entities.Payment{
		Status: entities.PaymentStatusCompleted,
	}
	events := []*entities.PaymentEvent{
		{EventType: entities.PaymentEventType("PRIVACY_PAYMENT_CREATED")},
		{EventType: entities.PaymentEventTypeDestinationTxHash},
	}

	stage, isPrivacy, _, reason := derivePaymentPrivacyLifecycle(payment, events)
	if stage != entities.PrivacyLifecycleSettledToStealth {
		t.Fatalf("expected %s, got %s", entities.PrivacyLifecycleSettledToStealth, stage)
	}
	if !isPrivacy {
		t.Fatalf("expected privacy candidate true")
	}
	if reason == "" {
		t.Fatalf("expected non-empty reason while waiting forward confirmation")
	}
}

func TestDerivePaymentPrivacyLifecycle_PendingWithPreparedSignal(t *testing.T) {
	payment := &entities.Payment{
		Status: entities.PaymentStatusPending,
	}
	events := []*entities.PaymentEvent{
		{EventType: entities.PaymentEventType("PRIVACY_PAYMENT_CREATED")},
	}

	stage, isPrivacy, _, _ := derivePaymentPrivacyLifecycle(payment, events)
	if stage != entities.PrivacyLifecyclePendingOnSource {
		t.Fatalf("expected %s, got %s", entities.PrivacyLifecyclePendingOnSource, stage)
	}
	if !isPrivacy {
		t.Fatalf("expected privacy candidate true")
	}
}
