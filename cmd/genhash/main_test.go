package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGeneratePasswordHash(t *testing.T) {
	hash, err := generatePasswordHash("AdminPayChain2026!", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("AdminPayChain2026!")); err != nil {
		t.Fatalf("hash mismatch: %v", err)
	}
}

func TestGeneratePasswordHash_InvalidCost(t *testing.T) {
	// Bcrypt rejects cost outside allowed range.
	_, err := generatePasswordHash("AdminPayChain2026!", 100)
	if err == nil {
		t.Fatal("expected error for invalid bcrypt cost")
	}
}

func TestMain_PrintsHash(t *testing.T) {
	origStdout := os.Stdout
	defer func() { os.Stdout = origStdout }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	main()

	_ = w.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(r)
	if !strings.Contains(out.String(), "$2") {
		t.Fatalf("expected bcrypt hash output, got: %s", out.String())
	}
}
