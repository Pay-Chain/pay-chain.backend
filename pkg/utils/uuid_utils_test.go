package utils

import "testing"

func TestGenerateUUIDv7(t *testing.T) {
	id := GenerateUUIDv7()
	if id.String() == "" {
		t.Fatal("expected non-empty uuid")
	}
}
