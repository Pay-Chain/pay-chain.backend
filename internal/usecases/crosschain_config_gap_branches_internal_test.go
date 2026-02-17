package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/pkg/utils"
)

func newCrosschainFeeRPCServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := map[string]interface{}{"jsonrpc": "2.0", "id": req["id"]}
		switch req["method"] {
		case "eth_chainId":
			resp["result"] = "0x2105"
		case "eth_call":
			resp["result"] = "0x01"
		default:
			resp["result"] = "0x0"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestCrosschainConfigUsecase_GapBranches(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}

	t.Run("overview with explicit dest and invalid inputs", func(t *testing.T) {
		chainRepo := &ccChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
			byCAIP2: map[string]*entities.Chain{
				"eip155:8453":  source,
				"eip155:42161": dest,
			},
			allChain: []*entities.Chain{source, dest},
		}
		u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType:      0,
					HasAdapterDefault:      true,
					AdapterDefaultType:     "0x1111111111111111111111111111111111111111",
					HasAdapterType0:        true,
					AdapterType0:           "0x1111111111111111111111111111111111111111",
					HyperbridgeConfigured:  true,
					CCIPDestinationAdapter: "0x",
				}, nil
			},
		})

		_, err := u.Overview(context.Background(), "", "invalid-dest", utils.PaginationParams{Page: 1, Limit: 10})
		require.Error(t, err)

		out, err := u.Overview(context.Background(), "eip155:8453", "eip155:42161", utils.PaginationParams{Page: 1, Limit: 10})
		require.NoError(t, err)
		require.Len(t, out.Items, 1)
		require.Equal(t, "eip155:42161", out.Items[0].DestChainID)
	})

	t.Run("recheck route invalid source and chain fetch failures", func(t *testing.T) {
		chainRepo := &ccChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
			byCAIP2: map[string]*entities.Chain{
				"eip155:8453":  source,
				"eip155:42161": dest,
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{}, nil
			},
		})

		_, err := u.RecheckRoute(context.Background(), "invalid-source", "eip155:42161")
		require.Error(t, err)

		u.chainRepo = &ccChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{destID: dest},
			byCAIP2: map[string]*entities.Chain{
				"eip155:8453":  source,
				"eip155:42161": dest,
			},
		}
		_, err = u.RecheckRoute(context.Background(), "eip155:8453", "eip155:42161")
		require.Error(t, err)

		u.chainRepo = &ccChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{sourceID: source},
			byCAIP2: map[string]*entities.Chain{
				"eip155:8453":  source,
				"eip155:42161": dest,
			},
		}
		_, err = u.RecheckRoute(context.Background(), "eip155:8453", "eip155:42161")
		require.Error(t, err)
	})

	t.Run("preflight route error and per-bridge route config errors", func(t *testing.T) {
		chainRepo := &ccChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
			byCAIP2: map[string]*entities.Chain{
				"eip155:8453":  source,
				"eip155:42161": dest,
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return nil, errors.New("status failed")
			},
		})
		_, err := u.Preflight(context.Background(), "eip155:8453", "eip155:42161")
		require.Error(t, err)

		u = NewCrosschainConfigUsecase(chainRepo, &ccTokenRepoStub{}, &ccContractRepoStub{}, nil, &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType: 0,
					HasAdapterType0:   true,
					AdapterType0:      "0x1111111111111111111111111111111111111111",
					// HyperbridgeConfigured false to hit routeConfigured false branch.
					HasAdapterType2: true,
					AdapterType2:    "0x2222222222222222222222222222222222222222",
					// LayerZeroConfigured false to hit routeConfigured false branch.
				}, nil
			},
		})
		preflight, err := u.Preflight(context.Background(), "eip155:8453", "eip155:42161")
		require.NoError(t, err)
		require.Equal(t, "HYPERBRIDGE_NOT_CONFIGURED", preflight.Bridges[0].ErrorCode)
		require.Equal(t, "LAYERZERO_NOT_CONFIGURED", preflight.Bridges[2].ErrorCode)
	})

	t.Run("preflight ready row and policy executable", func(t *testing.T) {
		rpcServer := newCrosschainFeeRPCServer(t)
		defer rpcServer.Close()
		sourceWithRPC := &entities.Chain{
			ID:      sourceID,
			ChainID: "8453",
			Name:    "Base",
			Type:    entities.ChainTypeEVM,
			IsActive: true,
			RPCURL:  rpcServer.URL,
		}
		chainRepo := &ccChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{sourceID: sourceWithRPC, destID: dest},
			byCAIP2: map[string]*entities.Chain{
				"eip155:8453":  sourceWithRPC,
				"eip155:42161": dest,
			},
		}
		tokenRepo := &ccTokenRepoStub{
			byChain: map[uuid.UUID][]*entities.Token{
				sourceID: {&entities.Token{ContractAddress: "0x1111111111111111111111111111111111111111"}},
				destID:   {&entities.Token{ContractAddress: "0x2222222222222222222222222222222222222222"}},
			},
		}
		contractRepo := &ccContractRepoStub{
			active: map[string]*entities.SmartContract{
				contractKey(sourceID, entities.ContractTypeRouter): {
					ChainUUID:       sourceID,
					Type:            entities.ContractTypeRouter,
					ContractAddress: "0x9999999999999999999999999999999999999999",
					IsActive:        true,
				},
			},
		}
		u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, blockchain.NewClientFactory(), &crosschainAdapterStub{
			statusFn: func(context.Context, string, string) (*OnchainAdapterStatus, error) {
				return &OnchainAdapterStatus{
					DefaultBridgeType:      0,
					HasAdapterDefault:      true,
					AdapterDefaultType:     "0x1111111111111111111111111111111111111111",
					HasAdapterType0:        true,
					AdapterType0:           "0x1111111111111111111111111111111111111111",
					HyperbridgeConfigured:  true,
					CCIPDestinationAdapter: "0x",
				}, nil
			},
		})

		preflight, err := u.Preflight(context.Background(), "eip155:8453", "eip155:42161")
		require.NoError(t, err)
		require.True(t, preflight.Bridges[0].Ready)
		require.True(t, preflight.PolicyExecutable)
	})

	t.Run("check fee quote health client create error", func(t *testing.T) {
		sourceBadRPC := &entities.Chain{
			ID:      sourceID,
			ChainID: "8453",
			Type:    entities.ChainTypeEVM,
			RPCURL:  "://bad-rpc-url",
		}
		u := NewCrosschainConfigUsecase(
			&ccChainRepoStub{},
			&ccTokenRepoStub{byChain: map[uuid.UUID][]*entities.Token{
				sourceID: {&entities.Token{ContractAddress: "0x1111111111111111111111111111111111111111"}},
				destID:   {&entities.Token{ContractAddress: "0x2222222222222222222222222222222222222222"}},
			}},
			&ccContractRepoStub{active: map[string]*entities.SmartContract{
				contractKey(sourceID, entities.ContractTypeRouter): {ContractAddress: "0x9999999999999999999999999999999999999999"},
			}},
			blockchain.NewClientFactory(),
			&crosschainAdapterStub{},
		)
		require.False(t, u.checkFeeQuoteHealth(context.Background(), sourceBadRPC, dest, 0))
	})
}
