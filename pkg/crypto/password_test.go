package crypto

import (
	"errors"
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

func TestHashPasswordAndGenerateRandomToken_ErrorBranches(t *testing.T) {
	origBcrypt := bcryptGenerateFromPassword
	origRandRead := randomRead
	t.Cleanup(func() {
		bcryptGenerateFromPassword = origBcrypt
		randomRead = origRandRead
	})

	bcryptGenerateFromPassword = func([]byte, int) ([]byte, error) {
		return nil, errors.New("bcrypt failed")
	}
	_, err := HashPassword("Password123!")
	assert.Error(t, err)

	bcryptGenerateFromPassword = origBcrypt
	randomRead = func([]byte) (int, error) {
		return 0, errors.New("rand failed")
	}
	_, err = GenerateRandomToken(16)
	assert.Error(t, err)
}
