package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestPaymentConfigHandler_ParseHelpers(t *testing.T) {
	chainID := uuid.New()
	h := &PaymentConfigHandler{}

	// patch repo behavior by embedding through methods used below
	stub := &crosschainChainRepoStub{
		getByChainID: func(_ context.Context, chain string) (*entities.Chain, error) {
			if chain == "8453" {
				return &entities.Chain{ID: chainID}, nil
			}
			return nil, errors.New("not found")
		},
		getByCAIP2: func(context.Context, string) (*entities.Chain, error) { return nil, errors.New("not used") },
	}
	h.chainRepo = stub

	id, err := h.parseChainID(context.Background(), chainID.String())
	require.NoError(t, err)
	require.Equal(t, chainID, id)

	id, err = h.parseChainID(context.Background(), "8453")
	require.NoError(t, err)
	require.Equal(t, chainID, id)

	_, err = h.parseChainID(context.Background(), "unknown")
	require.Error(t, err)

	opt, err := h.parseChainQuery(context.Background(), "")
	require.NoError(t, err)
	require.Nil(t, opt)

	opt, err = h.parseChainQuery(context.Background(), "8453")
	require.NoError(t, err)
	require.NotNil(t, opt)

	u, err := parseUUIDPtr("")
	require.NoError(t, err)
	require.Nil(t, u)
	u, err = parseUUIDPtr(chainID.String())
	require.NoError(t, err)
	require.Equal(t, chainID, *u)
	_, err = parseUUIDPtr("bad")
	require.Error(t, err)

	require.Equal(t, "0", defaultDecimal(""))
	require.Equal(t, "0", defaultDecimal("   "))
	require.Equal(t, "12.3", defaultDecimal("12.3"))
}
