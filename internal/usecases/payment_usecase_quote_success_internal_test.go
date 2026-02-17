package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func newQuoteRPCServer(t *testing.T, ethCallResults []interface{}) *httptest.Server {
	t.Helper()
	callIdx := 0
	return newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)

		res := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
		switch req["method"] {
		case "eth_chainId":
			res["result"] = "0x2105"
		case "eth_call":
			if callIdx < len(ethCallResults) {
				res["result"] = ethCallResults[callIdx]
				callIdx++
			} else {
				res["result"] = "0x"
			}
		default:
			res["result"] = "0x0"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
}

func newQuoteRPCServerWithCallError(t *testing.T) *httptest.Server {
	t.Helper()
	return newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)

		res := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
		switch req["method"] {
		case "eth_chainId":
			res["result"] = "0x2105"
		case "eth_call":
			res["error"] = map[string]interface{}{"code": -32000, "message": "execution reverted"}
		default:
			res["result"] = "0x0"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
}

func TestPaymentUsecase_QuoteBridgeFeeByType_SuccessAndEmpty(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := newQuoteRPCServer(t, []interface{}{"0x64"}) // 100
		defer srv.Close()

		client, err := blockchain.NewEVMClient(srv.URL)
		require.NoError(t, err)
		defer client.Close()

		u := &PaymentUsecase{}
		fee, err := u.quoteBridgeFeeByType(
			context.Background(),
			client,
			"0x1111111111111111111111111111111111111111",
			"eip155:42161",
			0,
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
			big.NewInt(1000),
		)
		require.NoError(t, err)
		require.Equal(t, int64(100), fee.Int64())
	})

	t.Run("empty result", func(t *testing.T) {
		srv := newQuoteRPCServer(t, []interface{}{"0x"})
		defer srv.Close()

		client, err := blockchain.NewEVMClient(srv.URL)
		require.NoError(t, err)
		defer client.Close()

		u := &PaymentUsecase{}
		_, err = u.quoteBridgeFeeByType(
			context.Background(),
			client,
			"0x1111111111111111111111111111111111111111",
			"eip155:42161",
			0,
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
			big.NewInt(1000),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty result")
	})
}

func TestPaymentUsecase_QuoteBridgeFeeByType_CallErrorRPCResponse(t *testing.T) {
	srv := newQuoteRPCServerWithCallError(t)
	defer srv.Close()

	client, err := blockchain.NewEVMClient(srv.URL)
	require.NoError(t, err)
	defer client.Close()

	u := &PaymentUsecase{}
	_, err = u.quoteBridgeFeeByType(
		context.Background(),
		client,
		"0x1111111111111111111111111111111111111111",
		"eip155:42161",
		0,
		"0x2222222222222222222222222222222222222222",
		"0x3333333333333333333333333333333333333333",
		big.NewInt(1000),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "contract call failed")
}

func TestPaymentUsecase_QuoteBridgeFeeByType_PackErrorAndExplicitEmptyResult(t *testing.T) {
	t.Run("nil amount panics in abi pack path", func(t *testing.T) {
		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{0x01}, nil
		})
		u := &PaymentUsecase{}
		require.Panics(t, func() {
			_, _ = u.quoteBridgeFeeByType(
				context.Background(),
				client,
				"0x1111111111111111111111111111111111111111",
				"eip155:42161",
				0,
				"0x2222222222222222222222222222222222222222",
				"0x3333333333333333333333333333333333333333",
				nil,
			)
		})
	})

	t.Run("empty result from callview hook", func(t *testing.T) {
		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{}, nil
		})
		u := &PaymentUsecase{}
		_, err := u.quoteBridgeFeeByType(
			context.Background(),
			client,
			"0x1111111111111111111111111111111111111111",
			"eip155:42161",
			0,
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
			big.NewInt(1000),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty result")
	})

	t.Run("abi type construction error", func(t *testing.T) {
		origNewType := newABIType
		t.Cleanup(func() { newABIType = origNewType })
		newABIType = func(string, string, []abi.ArgumentMarshaling) (abi.Type, error) {
			return abi.Type{}, errors.New("abi type failed")
		}

		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{0x01}, nil
		})
		u := &PaymentUsecase{}
		_, err := u.quoteBridgeFeeByType(
			context.Background(),
			client,
			"0x1111111111111111111111111111111111111111",
			"eip155:42161",
			0,
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
			big.NewInt(1000),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to build ABI tuple type")
	})

	t.Run("abi string type construction error", func(t *testing.T) {
		origNewType := newABIType
		t.Cleanup(func() { newABIType = origNewType })
		call := 0
		newABIType = func(name, alias string, args []abi.ArgumentMarshaling) (abi.Type, error) {
			call++
			if call == 2 {
				return abi.Type{}, errors.New("string type failed")
			}
			return origNewType(name, alias, args)
		}

		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{0x01}, nil
		})
		u := &PaymentUsecase{}
		_, err := u.quoteBridgeFeeByType(
			context.Background(),
			client,
			"0x1111111111111111111111111111111111111111",
			"eip155:42161",
			0,
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
			big.NewInt(1000),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to build ABI string type")
	})

	t.Run("abi uint8 type construction error", func(t *testing.T) {
		origNewType := newABIType
		t.Cleanup(func() { newABIType = origNewType })
		call := 0
		newABIType = func(name, alias string, args []abi.ArgumentMarshaling) (abi.Type, error) {
			call++
			if call == 3 {
				return abi.Type{}, errors.New("uint8 type failed")
			}
			return origNewType(name, alias, args)
		}

		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{0x01}, nil
		})
		u := &PaymentUsecase{}
		_, err := u.quoteBridgeFeeByType(
			context.Background(),
			client,
			"0x1111111111111111111111111111111111111111",
			"eip155:42161",
			0,
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
			big.NewInt(1000),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to build ABI uint8 type")
	})

	t.Run("pack error", func(t *testing.T) {
		origPack := packABIArgs
		t.Cleanup(func() { packABIArgs = origPack })
		packABIArgs = func(abi.Arguments, ...interface{}) ([]byte, error) {
			return nil, errors.New("pack failed")
		}

		client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{0x01}, nil
		})
		u := &PaymentUsecase{}
		_, err := u.quoteBridgeFeeByType(
			context.Background(),
			client,
			"0x1111111111111111111111111111111111111111",
			"eip155:42161",
			0,
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
			big.NewInt(1000),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to pack quotePaymentFee args")
	})
}

func TestPaymentUsecase_GetBridgeFeeQuote_UsesFallbackOrder(t *testing.T) {
	// first bridge returns zero (invalid), second returns fee 0x96 (150)
	srv := newQuoteRPCServer(t, []interface{}{"0x0", "0x96"})
	defer srv.Close()

	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{
		ID:      sourceID,
		ChainID: "8453",
		Type:    entities.ChainTypeEVM,
		RPCURL:  srv.URL,
	}
	dest := &entities.Chain{
		ID:      destID,
		ChainID: "42161",
		Type:    entities.ChainTypeEVM,
	}
	repo := &quoteChainRepoStub{
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  source,
			"eip155:42161": dest,
		},
		byID: map[uuid.UUID]*entities.Chain{
			sourceID: source,
			destID:   dest,
		},
	}
	u := &PaymentUsecase{
		chainRepo:     repo,
		chainResolver: NewChainResolver(repo),
		contractRepo: &quoteContractRepoStub{
			router: &entities.SmartContract{
				ContractAddress: "0x1111111111111111111111111111111111111111",
				Type:            entities.ContractTypeRouter,
			},
		},
		clientFactory: blockchain.NewClientFactory(),
		tokenRepo:     quoteTokenRepoStub{},
		routePolicyRepo: &routePolicyRepoStub{
			getByRouteFn: func(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
				return &entities.RoutePolicy{
					DefaultBridgeType: 0,
					FallbackMode:      entities.BridgeFallbackModeAutoFallback,
					FallbackOrder:     []uint8{1},
				}, nil
			},
		},
	}

	fee, err := u.getBridgeFeeQuote(
		context.Background(),
		"eip155:8453",
		"eip155:42161",
		"0x2222222222222222222222222222222222222222",
		"0x3333333333333333333333333333333333333333",
		big.NewInt(1000),
	)
	require.NoError(t, err)
	require.Equal(t, int64(150), fee.Int64())
}
