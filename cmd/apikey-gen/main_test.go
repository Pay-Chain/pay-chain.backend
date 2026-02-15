package main

import (
	"strings"
	"testing"
)

func TestGenerateRandomHex(t *testing.T) {
	v, err := generateRandomHex(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v) != 32 {
		t.Fatalf("expected len 32 got %d", len(v))
	}

	v2, err := generateRandomHex(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v2) != 2 {
		t.Fatalf("expected len 2 got %d", len(v2))
	}
}

func TestValidateInputs(t *testing.T) {
	if err := validateInputs("live", 32); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateInputs("bad", 32); err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if err := validateInputs("test", 3); err == nil {
		t.Fatal("expected error for odd hex len")
	}
}

func TestBuildCredentials(t *testing.T) {
	apiKey, secretKey, err := buildCredentials("test", 32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(apiKey, "pk_test_") {
		t.Fatalf("unexpected api key format: %s", apiKey)
	}
	if !strings.HasPrefix(secretKey, "sk_test_") {
		t.Fatalf("unexpected secret key format: %s", secretKey)
	}
}
