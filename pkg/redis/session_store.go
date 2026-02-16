package redis

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"
)

// SessionData holds the data stored in the session
type SessionData struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// SessionStore handles session storage in Redis with encryption
type SessionStore struct {
	encryptionKey []byte
}

var (
	setSessionValue = Set
	getSessionValue = Get
	delSessionValue = Del
)

// NewSessionStore creates a new session store
func NewSessionStore(encryptionKeyHex string) (*SessionStore, error) {
	key, err := hex.DecodeString(encryptionKeyHex)
	if err != nil {
		return nil, errors.New("invalid encryption key hex")
	}
	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes (64 hex chars)")
	}
	return &SessionStore{encryptionKey: key}, nil
}

// CreateSession stores encrypted session data in Redis
func (s *SessionStore) CreateSession(ctx context.Context, sessionID string, data *SessionData, expiration time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	encryptedData, err := s.encrypt(jsonData)
	if err != nil {
		return err
	}

	return setSessionValue(ctx, "session:"+sessionID, encryptedData, expiration)
}

// GetSession retrieves and decrypts session data from Redis
func (s *SessionStore) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
	encryptedDataStr, err := getSessionValue(ctx, "session:"+sessionID)
	if err != nil {
		return nil, err
	}

	decryptedData, err := s.decrypt(encryptedDataStr)
	if err != nil {
		return nil, err
	}

	var data SessionData
	if err := json.Unmarshal(decryptedData, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// DeleteSession removes a session from Redis
func (s *SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	return delSessionValue(ctx, "session:"+sessionID)
}

func (s *SessionStore) encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return hex.EncodeToString(ciphertext), nil
}

func (s *SessionStore) decrypt(ciphertextHex string) ([]byte, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
