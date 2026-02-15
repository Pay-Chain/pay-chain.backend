package main

import (
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
