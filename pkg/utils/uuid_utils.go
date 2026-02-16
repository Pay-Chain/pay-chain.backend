package utils

import (
	"github.com/google/uuid"
)

var newUUIDv7 = uuid.NewV7

// GenerateUUIDv7 generates a new UUID v7
func GenerateUUIDv7() uuid.UUID {
	id, err := newUUIDv7()
	if err != nil {
		// Fallback to v4 if v7 fails (highly unlikely)
		return uuid.New()
	}
	return id
}
