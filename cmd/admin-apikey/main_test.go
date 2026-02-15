package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseUserID(t *testing.T) {
	if _, err := parseUserID(""); err == nil {
		t.Fatal("expected error for empty user id")
	}
	if _, err := parseUserID("bad-uuid"); err == nil {
		t.Fatal("expected error for invalid uuid")
	}

	id := uuid.New()
	got, err := parseUserID(id.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id {
		t.Fatalf("expected %s got %s", id, got)
	}
}

func TestResolveAPIKeyName(t *testing.T) {
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	if got := resolveAPIKeyName("custom", now); got != "custom" {
		t.Fatalf("expected custom got %s", got)
	}
	if got := resolveAPIKeyName("", now); got != "frontend-proxy-admin-20260215-120000" {
		t.Fatalf("unexpected generated name: %s", got)
	}
}
