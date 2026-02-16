package usecases

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type rpcRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
	ID      interface{}       `json:"id"`
}

func newTestEVMClient(t *testing.T, callResults []string) *blockchain.EVMClient {
	t.Helper()

	var (
		mu    sync.Mutex
		index int
	)

	var srv *httptest.Server
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("skip: httptest server unavailable in this environment: %v", r)
			}
		}()
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req rpcRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "eth_chainId":
			resp.Result = "0x1"
		case "eth_call":
			mu.Lock()
			if index < len(callResults) {
				resp.Result = callResults[index]
				index++
			} else {
				resp.Result = "0x"
			}
			mu.Unlock()
		default:
			resp.Result = "0x0"
		}

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
		}))
	}()
	t.Cleanup(srv.Close)

	client, err := blockchain.NewEVMClient(srv.URL)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

func mustEncodeOutput(t *testing.T, rawABI, method string, values ...interface{}) string {
	t.Helper()
	parsed, err := abi.JSON(stringsReader(rawABI))
	require.NoError(t, err)
	packed, err := parsed.Methods[method].Outputs.Pack(values...)
	require.NoError(t, err)
	return "0x" + common.Bytes2Hex(packed)
}

func stringsReader(raw string) *strings.Reader {
	return strings.NewReader(raw)
}

func TestCallViewHelpers_Success(t *testing.T) {
	const rawABI = `[
		{"inputs":[],"name":"getBool","outputs":[{"type":"bool"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"getU8","outputs":[{"type":"uint8"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"getU64","outputs":[{"type":"uint64"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"getAddr","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"getBytes","outputs":[{"type":"bytes"}],"stateMutability":"view","type":"function"}
	]`

	expectedAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	expectedBytes := []byte{0xde, 0xad, 0xbe, 0xef}
	client := newTestEVMClient(t, []string{
		mustEncodeOutput(t, rawABI, "getBool", true),
		mustEncodeOutput(t, rawABI, "getU8", uint8(7)),
		mustEncodeOutput(t, rawABI, "getU64", uint64(42)),
		mustEncodeOutput(t, rawABI, "getAddr", expectedAddr),
		mustEncodeOutput(t, rawABI, "getBytes", expectedBytes),
	})

	vBool, err := callBoolView(context.Background(), client, expectedAddr.Hex(), rawABI, "getBool")
	require.NoError(t, err)
	require.True(t, vBool)

	vU8, err := callUint8View(context.Background(), client, expectedAddr.Hex(), rawABI, "getU8")
	require.NoError(t, err)
	require.Equal(t, uint8(7), vU8)

	vU64, err := callUint64View(context.Background(), client, expectedAddr.Hex(), rawABI, "getU64")
	require.NoError(t, err)
	require.Equal(t, uint64(42), vU64)

	vAddr, err := callAddressView(context.Background(), client, expectedAddr.Hex(), rawABI, "getAddr")
	require.NoError(t, err)
	require.Equal(t, expectedAddr, vAddr)

	vBytes, err := callBytesView(context.Background(), client, expectedAddr.Hex(), rawABI, "getBytes")
	require.NoError(t, err)
	require.Equal(t, expectedBytes, vBytes)
}

func TestCallViewHelpers_ParseAndDecodeError(t *testing.T) {
	client := newTestEVMClient(t, []string{"0x"})

	_, err := callBoolView(context.Background(), client, common.Address{}.Hex(), "not-json", "x")
	require.Error(t, err)

	const rawABI = `[{"inputs":[],"name":"getBool","outputs":[{"type":"bool"}],"stateMutability":"view","type":"function"}]`
	_, err = callBoolView(context.Background(), client, common.Address{}.Hex(), rawABI, "getBool")
	require.Error(t, err)
}
