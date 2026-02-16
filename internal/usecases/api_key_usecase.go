package usecases

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
)

var (
	apiKeyRandRead  = rand.Read
	apiKeyRandReader io.Reader = rand.Reader
)

type ApiKeyUsecase struct {
	apiKeyRepo    repositories.ApiKeyRepository
	userRepo      repositories.UserRepository
	encryptionKey []byte // 32 bytes for AES-256
}

func NewApiKeyUsecase(
	apiKeyRepo repositories.ApiKeyRepository,
	userRepo repositories.UserRepository,
	encryptionKeyHex string,
) *ApiKeyUsecase {
	key, err := hex.DecodeString(encryptionKeyHex)
	if err != nil || len(key) != 32 {
		// Fallback or panic in production? For now, we panic if config is bad to prevent insecure starts
		// But here we might just log error or assume caller checked.
		// If hex decode fails, we might panic.
		if len(encryptionKeyHex) == 32 {
			key = []byte(encryptionKeyHex) // Allow raw bytes if not hex encoded? No, stick to hex.
		}
	}

	return &ApiKeyUsecase{
		apiKeyRepo:    apiKeyRepo,
		userRepo:      userRepo,
		encryptionKey: key,
	}
}

func (u *ApiKeyUsecase) CreateApiKey(ctx context.Context, userID uuid.UUID, input *entities.CreateApiKeyInput) (*entities.CreateApiKeyResponse, error) {
	// Generate Key and Secret
	// pk_live_<32 hex chars>
	// sk_live_<32 hex chars>
	apiKeyRaw, err := generateRandomHex(32)
	if err != nil {
		return nil, domainerrors.InternalServerError("failed to generate key")
	}
	apiKey := "pk_live_" + apiKeyRaw

	secretKeyRaw, err := generateRandomHex(32)
	if err != nil {
		return nil, domainerrors.InternalServerError("failed to generate secret")
	}
	secretKey := "sk_live_" + secretKeyRaw

	// Hash Key (SHA256)
	keyHash := sha256Hex([]byte(apiKey))

	// Encrypt Secret (AES-GCM)
	secretEncrypted, err := u.encrypt(secretKey)
	if err != nil {
		return nil, domainerrors.InternalServerError("failed to encrypt secret")
	}

	// Mask Secret
	secretMasked := "****" + secretKey[len(secretKey)-4:]

	entity := &entities.ApiKey{
		UserID:          userID,
		Name:            input.Name,
		KeyPrefix:       "pk_live_",
		KeyHash:         keyHash,
		SecretEncrypted: secretEncrypted,
		SecretMasked:    secretMasked,
		Permissions:     input.Permissions,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := u.apiKeyRepo.Create(ctx, entity); err != nil {
		return nil, err
	}

	return &entities.CreateApiKeyResponse{
		ID:        entity.ID,
		Name:      entity.Name,
		ApiKey:    apiKey,    // Shown once
		SecretKey: secretKey, // Shown once
		CreatedAt: entity.CreatedAt,
	}, nil
}

func (u *ApiKeyUsecase) ValidateApiKey(
	ctx context.Context,
	apiKey string,
	signature string,
	timestamp string,
	method string,
	path string,
	bodyHash string,
) (*entities.User, error) {
	// 1. Verify timestamp Freshness (+/- 5 min)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, domainerrors.Unauthorized("invalid timestamp")
	}
	now := time.Now().Unix()
	if math.Abs(float64(now-ts)) > 300 { // 5 minutes
		return nil, domainerrors.Unauthorized("request timestamp expired")
	}

	// 2. Lookup API Key
	keyHash := sha256Hex([]byte(apiKey))
	keyEntity, err := u.apiKeyRepo.FindByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, domainerrors.Unauthorized("invalid api key")
	}
	if !keyEntity.IsActive {
		return nil, domainerrors.Unauthorized("api key inactive")
	}

	// 3. Decrypt Secret
	secretKey, err := u.decrypt(keyEntity.SecretEncrypted)
	if err != nil {
		return nil, domainerrors.InternalServerError("failed to decrypt secret")
	}

	// 4. Verify Signature
	// StringToSign = timestamp + method + path + bodyHash
	// path should be the full request URI (e.g. /v1/payments)
	stringToSign := fmt.Sprintf("%s%s%s%s", timestamp, method, path, bodyHash)
	expectedSignature := hmacSha256Hex(secretKey, stringToSign)

	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) { // Constant time compare? Hex strings vs hex strings is usually fine if length constant, but hmac.Equal expects []byte
		return nil, domainerrors.Unauthorized("invalid signature")
	}

	// 5. Update LastUsedAt (Async/Fire-and-forget ideally, but sync is fine for now)
	nowTime := time.Now()
	keyEntity.LastUsedAt = &nowTime
	_ = u.apiKeyRepo.Update(ctx, keyEntity)

	// 6. Return User
	if keyEntity.User == nil {
		// Should have been preloaded, if not, fetch
		user, err := u.userRepo.GetByID(ctx, keyEntity.UserID)
		if err != nil {
			return nil, domainerrors.InternalServerError("api key owner not found")
		}
		return user, nil
	}

	return keyEntity.User, nil
}

// ValidateSignatureForJWT verifies signature using USER'S active API keys
// Returns nil if signature valid for ANY active key, else error
func (u *ApiKeyUsecase) ValidateSignatureForJWT(
	ctx context.Context,
	userID uuid.UUID,
	signature string,
	timestamp string,
	method string,
	path string,
	bodyHash string,
) error {
	// 1. Verify Timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return domainerrors.Unauthorized("invalid timestamp")
	}
	if math.Abs(float64(time.Now().Unix()-ts)) > 300 {
		return domainerrors.Unauthorized("request timestamp expired")
	}

	// 2. Find User's active keys
	keys, err := u.apiKeyRepo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return domainerrors.Unauthorized("no api keys found for signature verification")
	}

	// 3. Try validating against each active key's secret
	// This allows key rotation/multiple keys
	stringToSign := fmt.Sprintf("%s%s%s%s", timestamp, method, path, bodyHash)

	for _, k := range keys {
		if !k.IsActive {
			continue
		}
		secret, err := u.decrypt(k.SecretEncrypted)
		if err != nil {
			continue // Skip bad keys
		}

		expected := hmacSha256Hex(secret, stringToSign)
		if hmac.Equal([]byte(expected), []byte(signature)) {
			// Valid!
			now := time.Now()
			k.LastUsedAt = &now
			_ = u.apiKeyRepo.Update(ctx, k)
			return nil
		}
	}

	return domainerrors.Unauthorized("invalid signature")
}

func (u *ApiKeyUsecase) ListApiKeys(ctx context.Context, userID uuid.UUID) ([]*entities.ApiKey, error) {
	return u.apiKeyRepo.FindByUserID(ctx, userID)
}

func (u *ApiKeyUsecase) RevokeApiKey(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	key, err := u.apiKeyRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if key.UserID != userID {
		return domainerrors.Forbidden("not owner of api key")
	}

	return u.apiKeyRepo.Delete(ctx, id)
}

// Helpers

func generateRandomHex(n int) (string, error) {
	bytes := make([]byte, n/2) // n is hex chars, so bytes is n/2
	if _, err := apiKeyRandRead(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSha256Hex(secret, data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (u *ApiKeyUsecase) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(u.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(apiKeyRandReader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func (u *ApiKeyUsecase) decrypt(ciphertextHex string) (string, error) {
	data, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(u.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("malformed ciphertext")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
