package services

import (
	"payment-kita.backend/internal/domain/services"
	"payment-kita.backend/pkg/crypto"
)

type hmacServiceImpl struct{}

// NewHMACService creates a new HMACService implementation
func NewHMACService() services.HMACService {
	return &hmacServiceImpl{}
}

func (s *hmacServiceImpl) Generate(message, secret string) string {
	return crypto.GenerateHMAC(message, secret)
}

func (s *hmacServiceImpl) Verify(message, secret, signature string) bool {
	return crypto.VerifyHMAC(message, secret, signature)
}
