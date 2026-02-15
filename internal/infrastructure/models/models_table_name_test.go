package models

import "testing"

func TestTableNames(t *testing.T) {
	if got := (PaymentBridge{}).TableName(); got != "payment_bridge" {
		t.Fatalf("unexpected PaymentBridge table name: %s", got)
	}
	if got := (RoutePolicy{}).TableName(); got != "route_policies" {
		t.Fatalf("unexpected RoutePolicy table name: %s", got)
	}
	if got := (LayerZeroConfig{}).TableName(); got != "layerzero_configs" {
		t.Fatalf("unexpected LayerZeroConfig table name: %s", got)
	}
}
