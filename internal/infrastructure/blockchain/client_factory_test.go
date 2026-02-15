package blockchain

import (
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClientFactory_InitializesMaps(t *testing.T) {
	f := NewClientFactory()
	require.NotNil(t, f)
	require.NotNil(t, f.evmClients)
	require.NotNil(t, f.solanaClients)
	require.Equal(t, 0, len(f.evmClients))
}

func TestClientFactory_GetEVMClient_InvalidURL(t *testing.T) {
	f := NewClientFactory()
	_, err := f.GetEVMClient("://bad-url")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "failed to create EVM client"))
}

func TestEVMClient_ChainIDAccessor(t *testing.T) {
	id := big.NewInt(8453)
	c := &EVMClient{chainID: id}
	require.Equal(t, id, c.ChainID())
}

func TestNewEVMClient_InvalidURL(t *testing.T) {
	_, err := NewEVMClient("://bad-url")
	require.Error(t, err)
}
