package usecases

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestApiKeyHelpers_GenerateRandomHex_Error(t *testing.T) {
	orig := apiKeyRandRead
	t.Cleanup(func() { apiKeyRandRead = orig })

	apiKeyRandRead = func(_ []byte) (int, error) {
		return 0, errors.New("rand failed")
	}

	_, err := generateRandomHex(32)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rand failed")
}

func TestApiKeyHelpers_Encrypt_ReadNonceError(t *testing.T) {
	orig := apiKeyRandReader
	t.Cleanup(func() { apiKeyRandReader = orig })
	apiKeyRandReader = errReader{}

	u := &ApiKeyUsecase{encryptionKey: []byte("0123456789abcdef0123456789abcdef")}
	_, err := u.encrypt("secret")
	require.Error(t, err)
}

func TestApiKeyHelpers_Decrypt_MalformedCiphertext(t *testing.T) {
	u := &ApiKeyUsecase{encryptionKey: []byte("0123456789abcdef0123456789abcdef")}
	_, err := u.decrypt("aa")
	require.Error(t, err)
	require.Contains(t, err.Error(), "malformed ciphertext")
}

var _ io.Reader = errReader{}
