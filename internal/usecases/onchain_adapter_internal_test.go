package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
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
	validKey := "0x4c0883a69102937d6231471b5dbb6204fe51296170827931e8f95f6f8d5d2f66"

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

	t.Run("chain id rpc error", func(t *testing.T) {
		srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			res := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
			if req["method"] == "eth_chainId" {
				res["error"] = map[string]interface{}{"code": -32000, "message": "rpc down"}
			} else {
				res["result"] = "0x0"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(res)
		}))
		defer srv.Close()

		chainID := uuid.New()
		u := &OnchainAdapterUsecase{
			ownerPrivateKey: validKey,
			chainRepo: &quoteChainRepoStub{
				byID: map[uuid.UUID]*entities.Chain{
					chainID: {ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: srv.URL},
				},
			},
		}
		parsed := mustParseABI(`[{"inputs":[{"internalType":"uint256","name":"x","type":"uint256"}],"name":"setValue","outputs":[],"stateMutability":"nonpayable","type":"function"}]`)
		_, err := u.sendTx(context.Background(), chainID, "0x0000000000000000000000000000000000000001", parsed, "setValue", 1)
		require.Error(t, err)
		require.Contains(t, strings.ToLower(err.Error()), "rpc down")
	})

	t.Run("transact method missing", func(t *testing.T) {
		srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			res := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
			if req["method"] == "eth_chainId" {
				res["result"] = "0x1"
			} else {
				res["result"] = "0x0"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(res)
		}))
		defer srv.Close()

		chainID := uuid.New()
		u := &OnchainAdapterUsecase{
			ownerPrivateKey: validKey,
			chainRepo: &quoteChainRepoStub{
				byID: map[uuid.UUID]*entities.Chain{
					chainID: {ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: srv.URL},
				},
			},
		}
		parsed := mustParseABI(`[{"inputs":[{"internalType":"uint256","name":"x","type":"uint256"}],"name":"setValue","outputs":[],"stateMutability":"nonpayable","type":"function"}]`)
		_, err := u.sendTx(context.Background(), chainID, "0x0000000000000000000000000000000000000001", parsed, "unknownMethod", 1)
		require.Error(t, err)
		require.Contains(t, strings.ToLower(err.Error()), "method")
	})

	t.Run("transact hook success", func(t *testing.T) {
		srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			res := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
			if req["method"] == "eth_chainId" {
				res["result"] = "0x1"
			} else {
				res["result"] = "0x0"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(res)
		}))
		defer srv.Close()

		orig := performContractTransact
		t.Cleanup(func() { performContractTransact = orig })
		performContractTransact = func(_ *ethclient.Client, _ string, _ abi.ABI, _ *bind.TransactOpts, _ string, _ ...interface{}) (string, error) {
			return "0xdeadbeef", nil
		}

		chainID := uuid.New()
		u := &OnchainAdapterUsecase{
			ownerPrivateKey: validKey,
			chainRepo: &quoteChainRepoStub{
				byID: map[uuid.UUID]*entities.Chain{
					chainID: {ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: srv.URL},
				},
			},
		}
		parsed := mustParseABI(`[{"inputs":[{"internalType":"uint256","name":"x","type":"uint256"}],"name":"setValue","outputs":[],"stateMutability":"nonpayable","type":"function"}]`)
		txHash, err := u.sendTx(context.Background(), chainID, "0x0000000000000000000000000000000000000001", parsed, "setValue", 1)
		require.NoError(t, err)
		require.Equal(t, "0xdeadbeef", txHash)
	})

	t.Run("executeOnchainTx hook branches", func(t *testing.T) {
		origExec := executeOnchainTx
		t.Cleanup(func() { executeOnchainTx = origExec })

		chainID := uuid.New()
		u := &OnchainAdapterUsecase{
			ownerPrivateKey: validKey,
			chainRepo: &quoteChainRepoStub{
				byID: map[uuid.UUID]*entities.Chain{
					chainID: {ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "mock://chain"},
				},
			},
		}
		parsed := mustParseABI(`[{"inputs":[{"internalType":"uint256","name":"x","type":"uint256"}],"name":"setValue","outputs":[],"stateMutability":"nonpayable","type":"function"}]`)

		executeOnchainTx = func(context.Context, string, string, string, abi.ABI, string, ...interface{}) (string, error) {
			return "", errors.New("tx failed")
		}
		_, err := u.sendTx(context.Background(), chainID, "0x0000000000000000000000000000000000000001", parsed, "setValue", 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "tx failed")

		executeOnchainTx = func(context.Context, string, string, string, abi.ABI, string, ...interface{}) (string, error) {
			return "0xabc", nil
		}
		tx, err := u.sendTx(context.Background(), chainID, "0x0000000000000000000000000000000000000001", parsed, "setValue", 1)
		require.NoError(t, err)
		require.Equal(t, "0xabc", tx)
	})
}

func TestOnchainAdapterUsecase_AdminGetAdapterClosureBranches(t *testing.T) {
	ctx := context.Background()
	destCAIP2 := "eip155:42161"
	routerAddress := "0x00000000000000000000000000000000000000b2"
	sourceID := uuid.New()
	bridgeType := uint8(0)

	t.Run("client factory missing", func(t *testing.T) {
		u := NewOnchainAdapterUsecase(&quoteChainRepoStub{}, &scRepoStub{}, nil, "0xabc")
		_, err := u.adminOps.getAdapter(ctx, sourceID, routerAddress, destCAIP2, bridgeType)
		require.Error(t, err)
		require.Contains(t, strings.ToLower(err.Error()), "invalid input")
	})

	t.Run("source chain not found", func(t *testing.T) {
		u := NewOnchainAdapterUsecase(&quoteChainRepoStub{}, &scRepoStub{}, blockchain.NewClientFactory(), "0xabc")
		_, err := u.adminOps.getAdapter(ctx, sourceID, routerAddress, destCAIP2, bridgeType)
		require.Error(t, err)
	})

	t.Run("no active rpc", func(t *testing.T) {
		repo := &quoteChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{
				sourceID: {ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM},
			},
		}
		u := NewOnchainAdapterUsecase(repo, &scRepoStub{}, blockchain.NewClientFactory(), "0xabc")
		_, err := u.adminOps.getAdapter(ctx, sourceID, routerAddress, destCAIP2, bridgeType)
		require.Error(t, err)
		require.Contains(t, strings.ToLower(err.Error()), "invalid input")
	})

	t.Run("get evm client failed", func(t *testing.T) {
		repo := &quoteChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{
				sourceID: {ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: "://bad-rpc"},
			},
		}
		u := NewOnchainAdapterUsecase(repo, &scRepoStub{}, blockchain.NewClientFactory(), "0xabc")
		_, err := u.adminOps.getAdapter(ctx, sourceID, routerAddress, destCAIP2, bridgeType)
		require.Error(t, err)
	})

	t.Run("callGetAdapter decode failed", func(t *testing.T) {
		rpcURL := "mock://decode-failed"
		repo := &quoteChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{
				sourceID: {ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: rpcURL},
			},
		}
		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(rpcURL, blockchain.NewEVMClientWithCallView(nil, func(context.Context, string, []byte) ([]byte, error) {
			return []byte{0x01}, nil
		}))
		u := NewOnchainAdapterUsecase(repo, &scRepoStub{}, factory, "0xabc")
		_, err := u.adminOps.getAdapter(ctx, sourceID, routerAddress, destCAIP2, bridgeType)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode getAdapter")
	})

	t.Run("success", func(t *testing.T) {
		rpcURL := "mock://get-adapter-ok"
		want := common.HexToAddress("0x1111111111111111111111111111111111111111")
		repo := &quoteChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{
				sourceID: {ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: rpcURL},
			},
		}
		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(rpcURL, blockchain.NewEVMClientWithCallView(nil, func(context.Context, string, []byte) ([]byte, error) {
			return payChainRouterAdminABI.Methods["getAdapter"].Outputs.Pack(want)
		}))
		u := NewOnchainAdapterUsecase(repo, &scRepoStub{}, factory, "0xabc")
		got, err := u.adminOps.getAdapter(ctx, sourceID, routerAddress, destCAIP2, bridgeType)
		require.NoError(t, err)
		require.Equal(t, want.Hex(), got)
	})
}
