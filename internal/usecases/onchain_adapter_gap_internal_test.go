package usecases

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func parseABIForOnchainGapTest(t *testing.T, raw string) abi.ABI {
	t.Helper()
	parsed, err := abi.JSON(strings.NewReader(raw))
	require.NoError(t, err)
	return parsed
}

func encodeABIReturnForOnchainGapTest(t *testing.T, parsed abi.ABI, method string, values ...interface{}) string {
	t.Helper()
	out, err := parsed.Methods[method].Outputs.Pack(values...)
	require.NoError(t, err)
	return "0x" + hex.EncodeToString(out)
}

func newTestEVMClientWithError(t *testing.T, msg string) *blockchain.EVMClient {
	t.Helper()
	return blockchain.NewEVMClientWithCallView(nil, func(context.Context, string, []byte) ([]byte, error) {
		return nil, errors.New(msg)
	})
}

func TestOnchainAdapterUsecase_ResolveEVMContextCore_SourceChainGetByIDError(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	chainRepo := &quoteChainRepoStub{
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  {ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM},
			"eip155:42161": {ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM},
		},
	}

	u := &OnchainAdapterUsecase{
		chainRepo:     chainRepo,
		contractRepo:  &quoteContractRepoStub{},
		chainResolver: NewChainResolver(chainRepo),
	}

	_, _, _, _, _, err := u.resolveEVMContextCore(context.Background(), "eip155:8453", "eip155:42161")
	require.Error(t, err)
}

func TestOnchainAdapterUsecase_CallTypedView_InvalidReturnTypeDirect(t *testing.T) {
	parsed := parseABIForOnchainGapTest(t, `[
		{"inputs":[],"name":"flag","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
	]`)
	client := newTestEVMClient(t, []string{
		encodeABIReturnForOnchainGapTest(t, parsed, "flag", uint8(1)),
	})

	_, err := callTypedView[bool](context.Background(), client, "0x0000000000000000000000000000000000000001", parsed, "flag")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid flag return type")
}

func TestOnchainAdapterUsecase_CallTypedView_PackAndDecodeError(t *testing.T) {
	t.Run("pack error", func(t *testing.T) {
		parsed := parseABIForOnchainGapTest(t, `[
			{"inputs":[{"internalType":"uint256","name":"x","type":"uint256"}],"name":"f","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
		]`)
		client := newTestEVMClient(t, []string{"0x01"})
		_, err := callTypedView[bool](context.Background(), client, "0x0000000000000000000000000000000000000001", parsed, "f", "not-a-number")
		require.Error(t, err)
	})

	t.Run("decode error", func(t *testing.T) {
		parsed := parseABIForOnchainGapTest(t, `[
			{"inputs":[],"name":"flag","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
		]`)
		client := newTestEVMClient(t, []string{"0x"})
		_, err := callTypedView[bool](context.Background(), client, "0x0000000000000000000000000000000000000001", parsed, "flag")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode flag")
	})

	t.Run("unpack error with non-empty payload", func(t *testing.T) {
		parsed := parseABIForOnchainGapTest(t, `[
			{"inputs":[],"name":"flag","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
		]`)
		client := newTestEVMClient(t, []string{"0x01"})
		_, err := callTypedView[bool](context.Background(), client, "0x0000000000000000000000000000000000000001", parsed, "flag")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode flag")
	})

	t.Run("call error", func(t *testing.T) {
		parsed := parseABIForOnchainGapTest(t, `[
			{"inputs":[],"name":"flag","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
		]`)
		client := newTestEVMClientWithError(t, "rpc call failed")
		_, err := callTypedView[bool](context.Background(), client, "0x0000000000000000000000000000000000000001", parsed, "flag")
		require.Error(t, err)
		require.Contains(t, err.Error(), "rpc call failed")
	})
}

func TestOnchainAdapterUsecase_CallTypedView_Success(t *testing.T) {
	parsed := parseABIForOnchainGapTest(t, `[
		{"inputs":[],"name":"flag","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`)
	client := newTestEVMClient(t, []string{
		encodeABIReturnForOnchainGapTest(t, parsed, "flag", true),
	})

	value, err := callTypedView[bool](context.Background(), client, "0x0000000000000000000000000000000000000001", parsed, "flag")
	require.NoError(t, err)
	require.True(t, value)
}

func TestOnchainAdapterUsecase_CallTypedView_EmptyOutputsBranch(t *testing.T) {
	parsed := parseABIForOnchainGapTest(t, `[
		{"inputs":[],"name":"noop","outputs":[],"stateMutability":"view","type":"function"}
	]`)
	client := newTestEVMClient(t, []string{"0x"})

	_, err := callTypedView[bool](context.Background(), client, "0x0000000000000000000000000000000000000001", parsed, "noop")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode noop")
}

func TestOnchainAdapterUsecase_ParseHexToBytes32_WithoutPrefix(t *testing.T) {
	_, err := parseHexToBytes32("1111111111111111111111111111111111111111")
	require.NoError(t, err)
}

func TestOnchainAdapterUsecase_ExecuteOnchainTx_Errors(t *testing.T) {
	t.Run("dial error", func(t *testing.T) {
		_, err := executeOnchainTx(
			context.Background(),
			"http://127.0.0.1:0",
			"0x4c0883a69102937d6231471b5dbb6204fe51296170827931e8f95f6f8d5d2f66",
			"0x0000000000000000000000000000000000000001",
			parseABIForOnchainGapTest(t, `[]`),
			"noop",
		)
		require.Error(t, err)
	})

	t.Run("invalid owner private key", func(t *testing.T) {
		srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			resp := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"], "result": "0x1"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		_, err := executeOnchainTx(
			context.Background(),
			srv.URL,
			"not-a-private-key",
			"0x0000000000000000000000000000000000000001",
			parseABIForOnchainGapTest(t, `[]`),
			"noop",
		)
		require.Error(t, err)
		require.Contains(t, strings.ToLower(err.Error()), "invalid input")
	})

	t.Run("chain id rpc error", func(t *testing.T) {
		srv := newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			resp := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
			if req["method"] == "eth_chainId" {
				resp["error"] = map[string]interface{}{"code": -32000, "message": "chain id unavailable"}
			} else {
				resp["result"] = "0x0"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		_, err := executeOnchainTx(
			context.Background(),
			srv.URL,
			"0x4c0883a69102937d6231471b5dbb6204fe51296170827931e8f95f6f8d5d2f66",
			common.HexToAddress("0x0000000000000000000000000000000000000001").Hex(),
			parseABIForOnchainGapTest(t, `[]`),
			"noop",
		)
		require.Error(t, err)
		require.Contains(t, strings.ToLower(err.Error()), "chain id unavailable")
	})
}
