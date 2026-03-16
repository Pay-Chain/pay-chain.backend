package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJWEService_EncryptDecrypt(t *testing.T) {
	// 32-byte key for AES256GCM
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	svc, err := NewJWEService(key)
	assert.NoError(t, err)

	payload := JWEPayload{
		SessionID:  "sid_12345",
		Amount:     "100.50",
		MerchantID: "mid_abc",
		Currency:   "USDC",
		ExpiresAt:  time.Now().Add(1 * time.Hour).Unix(),
	}

	// 1. Encrypt
	token, err := svc.Encrypt(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 2. Decrypt
	decrypted, err := svc.Decrypt(token)
	assert.NoError(t, err)
	assert.NotNil(t, decrypted)

	// 3. Compare
	assert.Equal(t, payload.SessionID, decrypted.SessionID)
	assert.Equal(t, payload.Amount, decrypted.Amount)
	assert.Equal(t, payload.MerchantID, decrypted.MerchantID)
	assert.Equal(t, payload.Currency, decrypted.Currency)
	assert.Equal(t, payload.ExpiresAt, decrypted.ExpiresAt)
}

func TestJWEService_InvalidKey(t *testing.T) {
	_, err := NewJWEService([]byte("short"))
	assert.Error(t, err)
}

func TestJWEService_Tampering(t *testing.T) {
	key := make([]byte, 32)
	svc, _ := NewJWEService(key)

	payload := JWEPayload{SessionID: "test"}
	token, _ := svc.Encrypt(payload)

	// Corrupt the token (swap characters in the last part - the auth tag)
	bytesToken := []byte(token)
	if len(bytesToken) > 10 {
		bytesToken[len(bytesToken)-1] = 'A'
		bytesToken[len(bytesToken)-2] = 'B'
	}
	corruptedToken := string(bytesToken)

	_, err := svc.Decrypt(corruptedToken)
	assert.Error(t, err)
}

func TestJWEService_DifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	key1[0] = 1
	key2 := make([]byte, 32)
	key2[0] = 2

	svc1, _ := NewJWEService(key1)
	svc2, _ := NewJWEService(key2)

	payload := JWEPayload{SessionID: "test"}
	token, _ := svc1.Encrypt(payload)

	_, err := svc2.Decrypt(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decryption failed")
}
