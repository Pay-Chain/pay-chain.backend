package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseABI_Invalid(t *testing.T) {
	_, err := parseABI(`[{invalid-json}]`)
	require.Error(t, err)
}

func TestCallViewHelpers_InvalidABI(t *testing.T) {
	ctx := context.Background()
	_, err := callBoolView(ctx, nil, "0x0", `[{invalid-json}]`, "x")
	require.Error(t, err)

	_, err = callUint8View(ctx, nil, "0x0", `[{invalid-json}]`, "x")
	require.Error(t, err)

	_, err = callUint64View(ctx, nil, "0x0", `[{invalid-json}]`, "x")
	require.Error(t, err)

	_, err = callAddressView(ctx, nil, "0x0", `[{invalid-json}]`, "x")
	require.Error(t, err)

	_, err = callBytesView(ctx, nil, "0x0", `[{invalid-json}]`, "x")
	require.Error(t, err)
}
