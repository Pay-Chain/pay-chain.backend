package models

import "testing"

func TestTableNames(t *testing.T) {
	if got := (PaymentBridge{}).TableName(); got != "payment_bridge" {
		t.Fatalf("unexpected PaymentBridge table name: %s", got)
	}
	if got := (RoutePolicy{}).TableName(); got != "route_policies" {
		t.Fatalf("unexpected RoutePolicy table name: %s", got)
	}
	if got := (StargateConfig{}).TableName(); got != "stargate_configs" {
		t.Fatalf("unexpected StargateConfig table name: %s", got)
	}
	if got := (PaymentQuote{}).TableName(); got != "payment_quotes" {
		t.Fatalf("unexpected PaymentQuote table name: %s", got)
	}
	if got := (PartnerPaymentSession{}).TableName(); got != "partner_payment_sessions" {
		t.Fatalf("unexpected PartnerPaymentSession table name: %s", got)
	}
	if got := (MerchantSettlementProfile{}).TableName(); got != "merchant_settlement_profiles" {
		t.Fatalf("unexpected MerchantSettlementProfile table name: %s", got)
	}
}
