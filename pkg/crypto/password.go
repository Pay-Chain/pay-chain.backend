package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the default bcrypt cost
	DefaultCost = 12
)

var (
	bcryptGenerateFromPassword = bcrypt.GenerateFromPassword
	randomRead                 = rand.Read
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcryptGenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword compares a password with a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomToken generates a random token of specified length
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := randomRead(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateVerificationToken generates a 32-character verification token
func GenerateVerificationToken() (string, error) {
	return GenerateRandomToken(16) // 16 bytes = 32 hex characters
}
