package blockchain

import (
	"context"
	"fmt"
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

func TestClientFactory_RegisterEVMClient(t *testing.T) {
	f := NewClientFactory()
	const rpcURL = "mock://rpc"
	injected := NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
		return []byte{0x01}, nil
	})

	f.RegisterEVMClient(rpcURL, injected)
	got, err := f.GetEVMClient(rpcURL)
	require.NoError(t, err)
	require.Same(t, injected, got)
}

func TestNewEVMClientWithCallView_DefaultChainIDAndCall(t *testing.T) {
	called := false
	client := NewEVMClientWithCallView(nil, func(_ context.Context, to string, data []byte) ([]byte, error) {
		called = true
		require.Equal(t, "0x1111111111111111111111111111111111111111", to)
		require.Equal(t, []byte{0xaa}, data)
		return []byte{0xbb}, nil
	})

	require.Equal(t, int64(1), client.ChainID().Int64())
	out, err := client.CallView(context.Background(), "0x1111111111111111111111111111111111111111", []byte{0xaa})
	require.NoError(t, err)
	require.Equal(t, []byte{0xbb}, out)
	require.True(t, called)

	clientErr := NewEVMClientWithCallView(big.NewInt(10), func(context.Context, string, []byte) ([]byte, error) {
		return nil, fmt.Errorf("boom")
	})
	_, err = clientErr.CallView(context.Background(), "0x1111111111111111111111111111111111111111", []byte{0xaa})
	require.Error(t, err)
}
