package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("Password123!")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	assert.True(t, CheckPassword("Password123!", hash))
	assert.False(t, CheckPassword("WrongPass", hash))
}

func TestGenerateRandomToken(t *testing.T) {
	token, err := GenerateRandomToken(16)
	assert.NoError(t, err)
	assert.Len(t, token, 32) // hex encoded

	verifyToken, err := GenerateVerificationToken()
	assert.NoError(t, err)
	assert.Len(t, verifyToken, 32)
}
