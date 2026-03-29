package usecases

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveCreatePaymentExpiresAt_Default(t *testing.T) {
	now := time.Date(2026, time.March, 29, 2, 0, 0, 0, time.UTC)

	expiresAt, unlimited, err := resolveCreatePaymentExpiresAt("", now)
	require.NoError(t, err)
	require.False(t, unlimited)
	require.Equal(t, now.Add(defaultCreatePaymentTTL), expiresAt)
}

func TestResolveCreatePaymentExpiresAt_CustomSeconds(t *testing.T) {
	now := time.Date(2026, time.March, 29, 2, 0, 0, 0, time.UTC)

	expiresAt, unlimited, err := resolveCreatePaymentExpiresAt("90", now)
	require.NoError(t, err)
	require.False(t, unlimited)
	require.Equal(t, now.Add(90*time.Second), expiresAt)
}

func TestResolveCreatePaymentExpiresAt_Unlimited(t *testing.T) {
	now := time.Date(2026, time.March, 29, 2, 0, 0, 0, time.UTC)

	expiresAt, unlimited, err := resolveCreatePaymentExpiresAt("unlimited", now)
	require.NoError(t, err)
	require.True(t, unlimited)
	require.True(t, isUnlimitedExpiryTime(expiresAt))
}

func TestResolveCreatePaymentExpiresAt_Invalid(t *testing.T) {
	now := time.Date(2026, time.March, 29, 2, 0, 0, 0, time.UTC)

	_, _, err := resolveCreatePaymentExpiresAt("abc", now)
	require.Error(t, err)

	_, _, err = resolveCreatePaymentExpiresAt("10", now)
	require.Error(t, err)

	_, _, err = resolveCreatePaymentExpiresAt("90000", now)
	require.Error(t, err)
}
