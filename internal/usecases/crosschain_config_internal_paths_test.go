package usecases

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/pkg/utils"
)

type ccChainRepoStub struct {
	byID     map[uuid.UUID]*entities.Chain
	byChain  map[string]*entities.Chain
	byCAIP2  map[string]*entities.Chain
	allChain []*entities.Chain
}

func (s *ccChainRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Chain, error) {
	if c, ok := s.byID[id]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *ccChainRepoStub) GetByChainID(_ context.Context, chainID string) (*entities.Chain, error) {
	if c, ok := s.byChain[chainID]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *ccChainRepoStub) GetByCAIP2(_ context.Context, caip2 string) (*entities.Chain, error) {
	if c, ok := s.byCAIP2[caip2]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *ccChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) { return s.allChain, nil }
func (s *ccChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *ccChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return s.allChain, int64(len(s.allChain)), nil
}
func (s *ccChainRepoStub) Create(context.Context, *entities.Chain) error       { return nil }
func (s *ccChainRepoStub) Update(context.Context, *entities.Chain) error       { return nil }
func (s *ccChainRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }
func (s *ccChainRepoStub) CreateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *ccChainRepoStub) UpdateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *ccChainRepoStub) DeleteRPC(context.Context, uuid.UUID) error          { return nil }
func (s *ccChainRepoStub) GetRPCByID(context.Context, uuid.UUID) (*entities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}

type ccTokenRepoStub struct {
	byChain map[uuid.UUID][]*entities.Token
}

func (s *ccTokenRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccTokenRepoStub) GetBySymbol(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccTokenRepoStub) GetByAddress(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccTokenRepoStub) GetAll(context.Context) ([]*entities.Token, error)         { return nil, nil }
func (s *ccTokenRepoStub) GetStablecoins(context.Context) ([]*entities.Token, error) { return nil, nil }
func (s *ccTokenRepoStub) GetNative(context.Context, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccTokenRepoStub) GetTokensByChain(_ context.Context, chainID uuid.UUID, _ utils.PaginationParams) ([]*entities.Token, int64, error) {
	items := s.byChain[chainID]
	return items, int64(len(items)), nil
}
func (s *ccTokenRepoStub) GetAllTokens(context.Context, *uuid.UUID, *string, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (s *ccTokenRepoStub) Create(context.Context, *entities.Token) error { return nil }
func (s *ccTokenRepoStub) Update(context.Context, *entities.Token) error { return nil }
func (s *ccTokenRepoStub) SoftDelete(context.Context, uuid.UUID) error   { return nil }

type ccContractRepoStub struct {
	active map[string]*entities.SmartContract
}

func contractKey(chainID uuid.UUID, t entities.SmartContractType) string {
	return chainID.String() + "|" + string(t)
}

func (s *ccContractRepoStub) Create(context.Context, *entities.SmartContract) error { return nil }
func (s *ccContractRepoStub) GetByID(context.Context, uuid.UUID) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccContractRepoStub) GetByChainAndAddress(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *ccContractRepoStub) GetActiveContract(_ context.Context, chainID uuid.UUID, contractType entities.SmartContractType) (*entities.SmartContract, error) {
	if c, ok := s.active[contractKey(chainID, contractType)]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *ccContractRepoStub) GetFiltered(context.Context, *uuid.UUID, entities.SmartContractType, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *ccContractRepoStub) GetByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *ccContractRepoStub) GetAll(context.Context, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *ccContractRepoStub) Update(context.Context, *entities.SmartContract) error { return nil }
func (s *ccContractRepoStub) SoftDelete(context.Context, uuid.UUID) error           { return nil }

type rpcReqInternal struct {
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
	ID     interface{}       `json:"id"`
}

func buildCrosschainRPCTestServer(t *testing.T, adapter0, adapter1, adapter2 common.Address) *httptest.Server {
	t.Helper()
	defaultMethodID := "0x" + hex.EncodeToString(FallbackPayChainGatewayABI.Methods["defaultBridgeTypes"].ID)
	hasMethodID := "0x" + hex.EncodeToString(FallbackPayChainRouterAdminABI.Methods["hasAdapter"].ID)
	getMethodID := "0x" + hex.EncodeToString(FallbackPayChainRouterAdminABI.Methods["getAdapter"].ID)
	hyperConfiguredMethodID := "0x" + hex.EncodeToString(FallbackHyperbridgeSenderAdminABI.Methods["isChainConfigured"].ID)
	hyperStateMachineMethodID := "0x" + hex.EncodeToString(FallbackHyperbridgeSenderAdminABI.Methods["stateMachineIds"].ID)
	hyperDestinationMethodID := "0x" + hex.EncodeToString(FallbackHyperbridgeSenderAdminABI.Methods["destinationContracts"].ID)
	ccipSelectorMethodID := "0x" + hex.EncodeToString(FallbackCCIPSenderAdminABI.Methods["chainSelectors"].ID)
	ccipDestinationMethodID := "0x" + hex.EncodeToString(FallbackCCIPSenderAdminABI.Methods["destinationAdapters"].ID)
	lzConfiguredMethodID := "0x" + hex.EncodeToString(FallbackLayerZeroSenderAdminABI.Methods["isRouteConfigured"].ID)
	lzDstMethodID := "0x" + hex.EncodeToString(FallbackLayerZeroSenderAdminABI.Methods["dstEids"].ID)
	lzPeerMethodID := "0x" + hex.EncodeToString(FallbackLayerZeroSenderAdminABI.Methods["peers"].ID)
	lzOptionsMethodID := "0x" + hex.EncodeToString(FallbackLayerZeroSenderAdminABI.Methods["enforcedOptions"].ID)

	return newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req rpcReqInternal
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := map[string]interface{}{"jsonrpc": "2.0", "id": req.ID}

		switch req.Method {
		case "eth_chainId":
			resp["result"] = "0x2105"
		case "eth_call":
			var callObj struct {
				Data  string `json:"data"`
				Input string `json:"input"`
			}
			if len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], &callObj)
			}
			callData := callObj.Data
			if callData == "" {
				callData = callObj.Input
			}
			methodID := ""
			if len(callData) >= 10 {
				methodID = callData[:10]
			}
			switch methodID {
			case defaultMethodID:
				out, _ := FallbackPayChainGatewayABI.Methods["defaultBridgeTypes"].Outputs.Pack(uint8(0))
				resp["result"] = "0x" + hex.EncodeToString(out)
			case hasMethodID:
				unpacked, _ := FallbackPayChainRouterAdminABI.Methods["hasAdapter"].Inputs.Unpack(common.FromHex(callData)[4:])
				bridgeType, _ := unpacked[1].(uint8)
				has := bridgeType <= 2
				out, _ := FallbackPayChainRouterAdminABI.Methods["hasAdapter"].Outputs.Pack(has)
				resp["result"] = "0x" + hex.EncodeToString(out)
			case getMethodID:
				unpacked, _ := FallbackPayChainRouterAdminABI.Methods["getAdapter"].Inputs.Unpack(common.FromHex(callData)[4:])
				bridgeType, _ := unpacked[1].(uint8)
				addr := common.Address{}
				if bridgeType == 0 {
					addr = adapter0
				} else if bridgeType == 1 {
					addr = adapter1
				} else if bridgeType == 2 {
					addr = adapter2
				}
				out, _ := FallbackPayChainRouterAdminABI.Methods["getAdapter"].Outputs.Pack(addr)
				resp["result"] = "0x" + hex.EncodeToString(out)
			case hyperConfiguredMethodID:
				out, _ := FallbackHyperbridgeSenderAdminABI.Methods["isChainConfigured"].Outputs.Pack(true)
				resp["result"] = "0x" + hex.EncodeToString(out)
			case hyperStateMachineMethodID:
				out, _ := FallbackHyperbridgeSenderAdminABI.Methods["stateMachineIds"].Outputs.Pack([]byte("EVM-42161"))
				resp["result"] = "0x" + hex.EncodeToString(out)
			case hyperDestinationMethodID:
				out, _ := FallbackHyperbridgeSenderAdminABI.Methods["destinationContracts"].Outputs.Pack(common.LeftPadBytes(common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Bytes(), 32))
				resp["result"] = "0x" + hex.EncodeToString(out)
			case ccipSelectorMethodID:
				out, _ := FallbackCCIPSenderAdminABI.Methods["chainSelectors"].Outputs.Pack(uint64(4949039107694359620))
				resp["result"] = "0x" + hex.EncodeToString(out)
			case ccipDestinationMethodID:
				out, _ := FallbackCCIPSenderAdminABI.Methods["destinationAdapters"].Outputs.Pack(common.LeftPadBytes(common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc").Bytes(), 32))
				resp["result"] = "0x" + hex.EncodeToString(out)
			case lzConfiguredMethodID:
				out, _ := FallbackLayerZeroSenderAdminABI.Methods["isRouteConfigured"].Outputs.Pack(true)
				resp["result"] = "0x" + hex.EncodeToString(out)
			case lzDstMethodID:
				out, _ := FallbackLayerZeroSenderAdminABI.Methods["dstEids"].Outputs.Pack(uint32(30110))
				resp["result"] = "0x" + hex.EncodeToString(out)
			case lzPeerMethodID:
				out, _ := FallbackLayerZeroSenderAdminABI.Methods["peers"].Outputs.Pack([32]byte{1})
				resp["result"] = "0x" + hex.EncodeToString(out)
			case lzOptionsMethodID:
				out, _ := FallbackLayerZeroSenderAdminABI.Methods["enforcedOptions"].Outputs.Pack([]byte{0x01, 0x02})
				resp["result"] = "0x" + hex.EncodeToString(out)
			default:
				// quotePaymentFee and unknown views: return non-empty 32-byte to mark ready.
				resp["result"] = "0x0000000000000000000000000000000000000000000000000000000000000001"
			}
		default:
			resp["result"] = "0x0"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestCrosschainConfigUsecase_InternalHappyAndAutoFixBranch(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	adapter0 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	adapter1 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	adapter2 := common.HexToAddress("0x3333333333333333333333333333333333333333")
	srv := buildCrosschainRPCTestServer(t, adapter0, adapter1, adapter2)
	defer srv.Close()

	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, IsActive: true, RPCURL: srv.URL}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM, IsActive: true}

	chainRepo := &ccChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
		byChain: map[string]*entities.Chain{
			"8453":  source,
			"42161": dest,
		},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  source,
			"eip155:42161": dest,
		},
		allChain: []*entities.Chain{source, dest},
	}
	tokenRepo := &ccTokenRepoStub{
		byChain: map[uuid.UUID][]*entities.Token{
			sourceID: {&entities.Token{ID: uuid.New(), ChainUUID: sourceID, ContractAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
			destID:   {&entities.Token{ID: uuid.New(), ChainUUID: destID, ContractAddress: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		},
	}

	contractRepo := &ccContractRepoStub{
		active: map[string]*entities.SmartContract{
			contractKey(sourceID, entities.ContractTypeGateway):            &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x00000000000000000000000000000000000000a1", IsActive: true},
			contractKey(sourceID, entities.ContractTypeRouter):             &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeRouter, ContractAddress: "0x00000000000000000000000000000000000000b2", IsActive: true},
			contractKey(sourceID, entities.ContractTypeAdapterHyperbridge): &entities.SmartContract{ID: uuid.New(), ChainUUID: sourceID, Type: entities.ContractTypeAdapterHyperbridge, ContractAddress: adapter0.Hex(), IsActive: true},
			contractKey(destID, entities.ContractTypeAdapterHyperbridge):   &entities.SmartContract{ID: uuid.New(), ChainUUID: destID, Type: entities.ContractTypeAdapterHyperbridge, ContractAddress: "0xcccccccccccccccccccccccccccccccccccccccc", IsActive: true},
		},
	}

	factory := blockchain.NewClientFactory()
	adapterUsecase := NewOnchainAdapterUsecase(chainRepo, contractRepo, factory, "")
	u := NewCrosschainConfigUsecase(chainRepo, tokenRepo, contractRepo, factory, adapterUsecase)

	route, err := u.RecheckRoute(context.Background(), "eip155:8453", "eip155:42161")
	require.NoError(t, err)
	require.Equal(t, "READY", route.OverallStatus)
	require.True(t, route.FeeQuoteHealthy)
	require.True(t, route.AdapterRegistered)

	preflight, err := u.Preflight(context.Background(), "eip155:8453", "eip155:42161")
	require.NoError(t, err)
	require.NotNil(t, preflight)
	require.True(t, preflight.PolicyExecutable)
	require.Len(t, preflight.Bridges, 3)

	result, err := u.AutoFix(context.Background(), &AutoFixRequest{
		SourceChainID: "eip155:8453",
		DestChainID:   "eip155:42161",
		BridgeType:    nil,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Steps, 3)
	require.Equal(t, "registerAdapter", result.Steps[0].Step)
	require.Equal(t, "SKIPPED", result.Steps[0].Status)
	require.Equal(t, "setDefaultBridge", result.Steps[1].Step)
	require.Equal(t, "SKIPPED", result.Steps[1].Status)
	require.Equal(t, "setHyperbridgeConfig", result.Steps[2].Step)
	require.Equal(t, "FAILED", result.Steps[2].Status)
	require.NotEmpty(t, strings.TrimSpace(result.Steps[2].Message))
}
