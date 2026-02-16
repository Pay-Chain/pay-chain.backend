package jwt

import (
	"testing"
	"time"

	gjwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJWTService_GenerateAndValidate(t *testing.T) {
	svc := NewJWTService("secret", time.Minute, 2*time.Minute)
	userID := uuid.New()

	pair, err := svc.GenerateTokenPair(userID, "test@mail.com", "USER")
	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)

	claims, err := svc.ValidateToken(pair.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "test@mail.com", claims.Email)
	assert.Equal(t, "USER", claims.Role)
}

func TestJWTService_ValidateInvalidToken(t *testing.T) {
	svc := NewJWTService("secret", time.Minute, 2*time.Minute)

	_, err := svc.ValidateToken("not-a-token")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTService_ValidateExpiredToken(t *testing.T) {
	svc := NewJWTService("secret", -time.Second, -time.Second)
	userID := uuid.New()

	pair, err := svc.GenerateTokenPair(userID, "expired@mail.com", "USER")
	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)

	_, err = svc.ValidateToken(pair.AccessToken)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestJWTService_ValidateWrongSigningMethod(t *testing.T) {
	svc := NewJWTService("secret", time.Minute, 2*time.Minute)

	claims := gjwt.MapClaims{
		"userId": uuid.NewString(),
		"email":  "x@y.z",
		"role":   "USER",
		"exp":    time.Now().Add(time.Minute).Unix(),
		"iat":    time.Now().Unix(),
		"nbf":    time.Now().Unix(),
	}
	unsigned := gjwt.NewWithClaims(gjwt.SigningMethodNone, claims)
	tokenStr, err := unsigned.SignedString(gjwt.UnsafeAllowNoneSignatureType)
	assert.NoError(t, err)

	_, err = svc.ValidateToken(tokenStr)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}
