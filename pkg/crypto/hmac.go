package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateHMAC generates an HMAC-SHA256 signature for the given message and secret
func GenerateHMAC(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyHMAC verifies if the given signature matches the message and secret
func VerifyHMAC(message, secret, signature string) bool {
	validSignature := GenerateHMAC(message, secret)
	return hmac.Equal([]byte(validSignature), []byte(signature))
}
