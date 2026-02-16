package blockchain

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type rpcResp struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func newEVMRPCServer(t *testing.T) *httptest.Server {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("skip: httptest server unavailable in this environment: %v", r)
		}
	}()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req rpcReq
		_ = json.NewDecoder(r.Body).Decode(&req)

		res := rpcResp{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "eth_chainId":
			res.Result = "0x2105"
		case "eth_getBalance":
			res.Result = "0xde0b6b3a7640000" // 1e18
		case "eth_call":
			// If calldata starts with balanceOf selector return 1000
			if strings.Contains(string(req.Params), "70a08231") {
				res.Result = "0x00000000000000000000000000000000000000000000000000000000000003e8"
			} else {
				res.Result = "0x1234"
			}
		case "eth_blockNumber":
			res.Result = "0x2a"
		case "eth_estimateGas":
			res.Result = "0x5208" // 21000
		case "eth_getTransactionByHash":
			res.Result = map[string]interface{}{
				"hash":             "0x1111111111111111111111111111111111111111111111111111111111111111",
				"nonce":            "0x0",
				"blockHash":        "0x2222222222222222222222222222222222222222222222222222222222222222",
				"blockNumber":      "0x1",
				"transactionIndex": "0x0",
				"from":             "0x3333333333333333333333333333333333333333",
				"to":               "0x4444444444444444444444444444444444444444",
				"value":            "0x0",
				"gas":              "0x5208",
				"gasPrice":         "0x3b9aca00",
				"input":            "0x",
				"v":                "0x1b",
				"r":                "0x1",
				"s":                "0x2",
				"type":             "0x0",
			}
		case "eth_getTransactionReceipt":
			res.Result = map[string]interface{}{
				"transactionHash":   "0x1111111111111111111111111111111111111111111111111111111111111111",
				"transactionIndex":  "0x0",
				"blockHash":         "0x2222222222222222222222222222222222222222222222222222222222222222",
				"blockNumber":       "0x1",
				"from":              "0x3333333333333333333333333333333333333333",
				"to":                "0x4444444444444444444444444444444444444444",
				"cumulativeGasUsed": "0x5208",
				"gasUsed":           "0x5208",
				"contractAddress":   nil,
				"logs":              []interface{}{},
				"logsBloom":         "0x" + strings.Repeat("0", 512),
				"status":            "0x1",
				"effectiveGasPrice": "0x3b9aca00",
			}
		default:
			res.Result = "0x0"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
}

func TestEVMClient_Methods_WithMockRPC(t *testing.T) {
	srv := newEVMRPCServer(t)
	defer srv.Close()

	client, err := NewEVMClient(srv.URL)
	require.NoError(t, err)

	chainID := client.ChainID()
	require.Equal(t, big.NewInt(8453), chainID)

	bal, err := client.GetBalance(context.Background(), "0x3333333333333333333333333333333333333333")
	require.NoError(t, err)
	require.Equal(t, "1000000000000000000", bal.String())

	tokenBal, err := client.GetTokenBalance(context.Background(), "0x4444444444444444444444444444444444444444", "0x3333333333333333333333333333333333333333")
	require.NoError(t, err)
	require.Equal(t, "1000", tokenBal.String())

	viewOut, err := client.CallView(context.Background(), "0x4444444444444444444444444444444444444444", []byte{0x12, 0x34})
	require.NoError(t, err)
	require.Equal(t, []byte{0x12, 0x34}, viewOut)

	block, err := client.GetBlockNumber(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(42), block)

	gas, err := client.EstimateGas(context.Background(), ethereum.CallMsg{To: ptrAddr(common.HexToAddress("0x4444444444444444444444444444444444444444"))})
	require.NoError(t, err)
	require.Equal(t, uint64(21000), gas)

	tx, pending, err := client.GetTransaction(context.Background(), "0x1111111111111111111111111111111111111111111111111111111111111111")
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.False(t, pending)

	receipt, err := client.GetTransactionReceipt(context.Background(), "0x1111111111111111111111111111111111111111111111111111111111111111")
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, uint64(1), receipt.Status)

	client.Close()
}

func TestClientFactory_GetEVMClient_CachePath(t *testing.T) {
	srv := newEVMRPCServer(t)
	defer srv.Close()

	f := NewClientFactory()
	c1, err := f.GetEVMClient(srv.URL)
	require.NoError(t, err)
	c2, err := f.GetEVMClient(srv.URL)
	require.NoError(t, err)
	require.Same(t, c1, c2)
	c1.Close()
}

func ptrAddr(a common.Address) *common.Address { return &a }
