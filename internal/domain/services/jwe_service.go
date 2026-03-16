package services

import (
	"encoding/json"
	"fmt"

	"github.com/go-jose/go-jose/v3"
)

// JWEPayload represents the structure encoded into the payment_code
type JWEPayload struct {
	SessionID  string `json:"sid"`
	Amount     string `json:"amt"`
	MerchantID string `json:"mid"`
	Currency   string `json:"cur"`
	ExpiresAt  int64  `json:"exp"`
}

type JWEService interface {
	Encrypt(payload JWEPayload) (string, error)
	Decrypt(jweString string) (*JWEPayload, error)
}

type jweService struct {
	encryptionKey []byte // 32 bytes for A256GCM
}

const (
	AES256KeySize = 32
)

func NewJWEService(key []byte) (JWEService, error) {
	if len(key) != AES256KeySize {
		return nil, fmt.Errorf("invalid key size: %d, expected %d", len(key), AES256KeySize)
	}
	return &jweService{encryptionKey: key}, nil
}

func (s *jweService) Encrypt(payload JWEPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Recipient for direct encryption (A256GCM)
	recipient := jose.Recipient{
		Algorithm: jose.DIRECT,
		Key:       s.encryptionKey,
	}

	encrypter, err := jose.NewEncrypter(
		jose.A256GCM,
		recipient,
		&jose.EncrypterOptions{Compression: jose.DEFLATE},
	)
	if err != nil {
		return "", fmt.Errorf("failed to create encrypter: %w", err)
	}

	object, err := encrypter.Encrypt(data)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}

	return object.CompactSerialize()
}

func (s *jweService) Decrypt(jweString string) (*JWEPayload, error) {
	object, err := jose.ParseEncrypted(jweString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWE: %w", err)
	}

	decrypted, err := object.Decrypt(s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	var payload JWEPayload
	if err := json.Unmarshal(decrypted, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &payload, nil
}
