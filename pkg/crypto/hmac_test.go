package crypto

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestHMAC(t *testing.T) {
	message := "hello world"
	secret := "my-secret-key"
	
	signature := GenerateHMAC(message, secret)
	assert.NotEmpty(t, signature)
	
	// Verification
	assert.True(t, VerifyHMAC(message, secret, signature))
	
	// Tamper message
	assert.False(t, VerifyHMAC("tampered", secret, signature))
	
	// Tamper secret
	assert.False(t, VerifyHMAC(message, "wrong-secret", signature))
	
	// Tamper signature
	assert.False(t, VerifyHMAC(message, secret, "wrong-signature"))
}
