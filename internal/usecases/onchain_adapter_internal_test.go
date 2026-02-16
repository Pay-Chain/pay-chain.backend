package usecases

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestResolveRPCURL(t *testing.T) {
	require.Equal(t, "", resolveRPCURL(nil))

	chain := &entities.Chain{RPCURL: "https://main-rpc"}
	require.Equal(t, "https://main-rpc", resolveRPCURL(chain))

	chain = &entities.Chain{RPCs: []entities.ChainRPC{{URL: "https://inactive"}, {URL: "https://active", IsActive: true}}}
	require.Equal(t, "https://active", resolveRPCURL(chain))

	chain = &entities.Chain{RPCs: []entities.ChainRPC{{URL: "https://fallback-1"}, {URL: "https://fallback-2"}}}
	require.Equal(t, "https://fallback-1", resolveRPCURL(chain))
}

func TestParseHexToBytes32(t *testing.T) {
	// 20-byte address input should be left padded to bytes32
	out, err := parseHexToBytes32("0x000000000000000000000000000000000000dEaD")
	require.NoError(t, err)
	require.Len(t, out, 32)

	// Full bytes32 should pass
	out, err = parseHexToBytes32("0x1111111111111111111111111111111111111111111111111111111111111111")
	require.NoError(t, err)
	require.Len(t, out, 32)

	// Invalid length should fail
	_, err = parseHexToBytes32("0x1234")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid bytes32 length")
}

func TestMustParseABI(t *testing.T) {
	require.NotPanics(t, func() {
		_ = mustParseABI(`[{"inputs":[],"name":"ping","outputs":[],"stateMutability":"view","type":"function"}]`)
	})

	require.Panics(t, func() {
		_ = mustParseABI(`[{invalid-json}]`)
	})
}

func TestOnchainAdapterUsecase_RegisterAdapter_InvalidAddress(t *testing.T) {
	u := &OnchainAdapterUsecase{}
	_, err := u.RegisterAdapter(context.Background(), "eip155:8453", "eip155:42161", 0, "not-hex")
	require.Error(t, err)
	require.Equal(t, "invalid input", err.Error())
}

func TestOnchainAdapterUsecase_SendTx_OwnerKeyMissing(t *testing.T) {
	u := &OnchainAdapterUsecase{ownerPrivateKey: ""}
	_, err := u.sendTx(context.Background(), uuid.New(), "0x0000000000000000000000000000000000000001", abi.ABI{}, "set", "arg")
	require.Error(t, err)
	require.Equal(t, "invalid input", err.Error())
}

func TestOnchainAdapterUsecase_New(t *testing.T) {
	repo := &quoteChainRepoStub{}
	u := NewOnchainAdapterUsecase(repo, &scRepoStub{}, nil, "0xabc")
	require.NotNil(t, u)
	require.NotNil(t, u.adminOps)
	require.NotNil(t, u.chainResolver)
}

func TestOnchainAdapterUsecase_SendTx_Branches(t *testing.T) {
	t.Run("source chain not found", func(t *testing.T) {
		u := &OnchainAdapterUsecase{
			ownerPrivateKey: "0xabc123",
			chainRepo:       &quoteChainRepoStub{},
		}
		_, err := u.sendTx(context.Background(), uuid.New(), "0x0000000000000000000000000000000000000001", abi.ABI{}, "set", "arg")
		require.Error(t, err)
	})

	t.Run("no active rpc", func(t *testing.T) {
		chainID := uuid.New()
		u := &OnchainAdapterUsecase{
			ownerPrivateKey: "0xabc123",
			chainRepo: &quoteChainRepoStub{
				byID: map[uuid.UUID]*entities.Chain{
					chainID: {ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM},
				},
			},
		}
		_, err := u.sendTx(context.Background(), chainID, "0x0000000000000000000000000000000000000001", abi.ABI{}, "set", "arg")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid input")
	})

	t.Run("invalid owner private key", func(t *testing.T) {
		srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			res := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
			if req["method"] == "eth_chainId" {
				res["result"] = "0x2105"
			} else {
				res["result"] = "0x0"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(res)
		}))
		defer srv.Close()

		chainID := uuid.New()
		u := &OnchainAdapterUsecase{
			ownerPrivateKey: "not-a-private-key",
			chainRepo: &quoteChainRepoStub{
				byID: map[uuid.UUID]*entities.Chain{
					chainID: {ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: srv.URL},
				},
			},
		}
		_, err := u.sendTx(context.Background(), chainID, "0x0000000000000000000000000000000000000001", abi.ABI{}, "set", "arg")
		require.Error(t, err)
		require.Equal(t, "invalid input", err.Error())
	})
}
