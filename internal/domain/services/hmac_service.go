package services

// HMACService provides methods for generating and verifying HMAC signatures
type HMACService interface {
	Generate(message, secret string) string
	Verify(message, secret, signature string) bool
}
