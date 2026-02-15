package main

import "testing"

func TestResolvePassword(t *testing.T) {
	if got := resolvePassword(nil); got != "The.Conqueror-45" {
		t.Fatalf("unexpected default password: %s", got)
	}
	if got := resolvePassword([]string{"abc"}); got != "abc" {
		t.Fatalf("unexpected arg password: %s", got)
	}
}

func TestGenerateHash(t *testing.T) {
	hash, err := generateHash("my-pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
}
