package usecases

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/pkg/utils"
)

type approvalChainRepoStub struct {
	chain *entities.Chain
}

func (s *approvalChainRepoStub) GetByID(_ context.Context, _ uuid.UUID) (*entities.Chain, error) {
	if s.chain == nil {
		return nil, domainerrors.ErrNotFound
	}
	return s.chain, nil
}
func (s *approvalChainRepoStub) GetByChainID(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *approvalChainRepoStub) GetByCAIP2(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *approvalChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) { return nil, nil }
func (s *approvalChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *approvalChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *approvalChainRepoStub) Create(context.Context, *entities.Chain) error { return nil }
func (s *approvalChainRepoStub) Update(context.Context, *entities.Chain) error { return nil }
func (s *approvalChainRepoStub) Delete(context.Context, uuid.UUID) error       { return nil }

type rpcReqPU struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
	ID      interface{}       `json:"id"`
}

type rpcRespPU struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
}

func newPaymentRPCServer(t *testing.T, callHandler func(callIndex int, data string) string) *httptest.Server {
	t.Helper()
	var (
		callMu sync.Mutex
		callIx int
	)
	return newSafeHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req rpcReqPU
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		res := rpcRespPU{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "eth_chainId":
			res.Result = "0x2105"
		case "eth_call":
			var payload struct {
				Data string `json:"data"`
			}
			if len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], &payload)
			}
			callMu.Lock()
			callIx++
			idx := callIx
			callMu.Unlock()
			res.Result = callHandler(idx, strings.ToLower(payload.Data))
		default:
			res.Result = "0x0"
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(res))
	}))
}

func mustPackOutputs(t *testing.T, outputTypes []string, values ...interface{}) string {
	t.Helper()
	args := make(abi.Arguments, 0, len(outputTypes))
	for _, typ := range outputTypes {
		abiType, err := abi.NewType(typ, "", nil)
		require.NoError(t, err)
		args = append(args, abi.Argument{Type: abiType})
	}
	out, err := args.Pack(values...)
	require.NoError(t, err)
	return "0x" + hex.EncodeToString(out)
}

func TestPaymentUsecase_CalculateOnchainApprovalAmount_QuoteSuccess(t *testing.T) {
	srv := newPaymentRPCServer(t, func(callIndex int, _ string) string {
		if callIndex == 1 {
			return mustPackOutputs(t, []string{"uint256", "uint256"}, big.NewInt(1100), big.NewInt(100))
		}
		return "0x"
	})
	defer srv.Close()

	chainID := uuid.New()
	u := &PaymentUsecase{
		chainRepo:     &approvalChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: srv.URL}},
		clientFactory: blockchain.NewClientFactory(),
	}

	payment := &entities.Payment{SourceChainID: chainID, SourceAmount: "1000", TotalCharged: "1050"}
	amount, err := u.calculateOnchainApprovalAmount(payment, "0x1111111111111111111111111111111111111111")
	require.NoError(t, err)
	require.Equal(t, "1100", amount)
}

func TestPaymentUsecase_CalculateOnchainApprovalAmount_FallbackFeePath(t *testing.T) {
	srv := newPaymentRPCServer(t, func(callIndex int, _ string) string {
		switch callIndex {
		case 1:
			return "0x" // force quote decode fail => fallback path
		case 2:
			return mustPackOutputs(t, []string{"uint256"}, big.NewInt(50))
		case 3:
			return mustPackOutputs(t, []string{"uint256"}, big.NewInt(100)) // 1%
		default:
			return "0x"
		}
	})
	defer srv.Close()

	chainID := uuid.New()
	u := &PaymentUsecase{
		chainRepo:     &approvalChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: srv.URL}},
		clientFactory: blockchain.NewClientFactory(),
	}

	payment := &entities.Payment{SourceChainID: chainID, SourceAmount: "1000", TotalCharged: "1000"}
	amount, err := u.calculateOnchainApprovalAmount(payment, "0x1111111111111111111111111111111111111111")
	require.NoError(t, err)
	require.Equal(t, "1050", amount)
}

func TestPaymentUsecase_ResolveVaultAddressForApproval_FromGatewayView(t *testing.T) {
	vaultAddress := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	padded := common.LeftPadBytes(vaultAddress.Bytes(), 32)

	srv := newPaymentRPCServer(t, func(callIndex int, _ string) string {
		if callIndex == 1 {
			return "0x" + hex.EncodeToString(padded)
		}
		return "0x"
	})
	defer srv.Close()

	chainID := uuid.New()
	u := &PaymentUsecase{
		contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
			return nil, domainerrors.ErrNotFound
		}},
		chainRepo:     &approvalChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: srv.URL}},
		clientFactory: blockchain.NewClientFactory(),
	}

	got := u.resolveVaultAddressForApproval(chainID, "0x1111111111111111111111111111111111111111")
	require.Equal(t, vaultAddress.Hex(), got)
}

func TestPaymentUsecase_ResolveVaultAddressForApproval_FallbackBranches(t *testing.T) {
	t.Run("chain repo miss", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo:  &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) { return nil, domainerrors.ErrNotFound }},
			chainRepo:     &approvalChainRepoStub{chain: nil},
			clientFactory: blockchain.NewClientFactory(),
		}
		got := u.resolveVaultAddressForApproval(uuid.New(), "0x1111111111111111111111111111111111111111")
		require.Equal(t, "", got)
	})

	t.Run("fallback to active rpc list with bad url", func(t *testing.T) {
		chainID := uuid.New()
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return nil, domainerrors.ErrNotFound
			}},
			chainRepo: &approvalChainRepoStub{chain: &entities.Chain{
				ID:     chainID,
				RPCURL: "",
				RPCs:   []entities.ChainRPC{{URL: "://bad", IsActive: true}},
			}},
			clientFactory: blockchain.NewClientFactory(),
		}
		got := u.resolveVaultAddressForApproval(chainID, "0x1111111111111111111111111111111111111111")
		require.Equal(t, "", got)
	})

	t.Run("short call output", func(t *testing.T) {
		srv := newPaymentRPCServer(t, func(_ int, _ string) string { return "0x01" })
		defer srv.Close()

		chainID := uuid.New()
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return nil, domainerrors.ErrNotFound
			}},
			chainRepo: &approvalChainRepoStub{chain: &entities.Chain{
				ID:     chainID,
				RPCURL: srv.URL,
			}},
			clientFactory: blockchain.NewClientFactory(),
		}
		got := u.resolveVaultAddressForApproval(chainID, "0x1111111111111111111111111111111111111111")
		require.Equal(t, "", got)
	})
}

func TestPaymentUsecase_CalculateOnchainApprovalAmount_ErrorBranches(t *testing.T) {
	t.Run("invalid payment input", func(t *testing.T) {
		u := &PaymentUsecase{}
		_, err := u.calculateOnchainApprovalAmount(nil, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid payment or gateway address")
	})

	t.Run("invalid source amount", func(t *testing.T) {
		u := &PaymentUsecase{}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceAmount: "not-number",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid source amount")
	})

	t.Run("source chain missing", func(t *testing.T) {
		u := &PaymentUsecase{
			chainRepo:     &approvalChainRepoStub{chain: nil},
			clientFactory: blockchain.NewClientFactory(),
		}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: uuid.New(),
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to resolve source chain")
	})

	t.Run("no active rpc", func(t *testing.T) {
		chainID := uuid.New()
		u := &PaymentUsecase{
			chainRepo: &approvalChainRepoStub{chain: &entities.Chain{
				ID:     chainID,
				RPCURL: "",
			}},
			clientFactory: blockchain.NewClientFactory(),
		}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no active source chain rpc url")
	})

	t.Run("invalid fixed fee decode", func(t *testing.T) {
		srv := newPaymentRPCServer(t, func(callIndex int, _ string) string {
			switch callIndex {
			case 1:
				return "0x" // quote path fails, fallback starts
			case 2:
				return "0x" // fixed fee decode fails
			default:
				return "0x"
			}
		})
		defer srv.Close()

		chainID := uuid.New()
		u := &PaymentUsecase{
			chainRepo:     &approvalChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: srv.URL}},
			clientFactory: blockchain.NewClientFactory(),
		}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode FIXED_BASE_FEE")
	})

	t.Run("fallback on invalid total charged keeps amount", func(t *testing.T) {
		srv := newPaymentRPCServer(t, func(callIndex int, _ string) string {
			switch callIndex {
			case 1:
				return "0x" // quote path fails, fallback starts
			case 2:
				return mustPackOutputs(t, []string{"uint256"}, big.NewInt(30))
			case 3:
				return mustPackOutputs(t, []string{"uint256"}, big.NewInt(50)) // 0.5%
			default:
				return "0x"
			}
		})
		defer srv.Close()

		chainID := uuid.New()
		u := &PaymentUsecase{
			chainRepo:     &approvalChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: srv.URL}},
			clientFactory: blockchain.NewClientFactory(),
		}
		amount, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "invalid",
		}, "0x1111111111111111111111111111111111111111")
		require.NoError(t, err)
		require.Equal(t, "1030", amount)
	})
}
