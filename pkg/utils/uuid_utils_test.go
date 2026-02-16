package utils

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateUUIDv7(t *testing.T) {
	id := GenerateUUIDv7()
	if id.String() == "" {
		t.Fatal("expected non-empty uuid")
	}
}

func TestGenerateUUIDv7_FallbackBranch(t *testing.T) {
	orig := newUUIDv7
	t.Cleanup(func() { newUUIDv7 = orig })

	newUUIDv7 = func() (uuid.UUID, error) {
		return uuid.Nil, errors.New("v7 failed")
	}
	id := GenerateUUIDv7()
	if id == uuid.Nil {
		t.Fatal("expected v4 fallback id when v7 fails")
	}
}
