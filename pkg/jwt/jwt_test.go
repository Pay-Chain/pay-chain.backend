package jwt

import (
	"testing"
	"time"

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
