package usecases

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/pkg/utils"
)

type quoteChainRepoStub struct {
	byID    map[uuid.UUID]*entities.Chain
	byChain map[string]*entities.Chain
	byCAIP2 map[string]*entities.Chain
}

func (s *quoteChainRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Chain, error) {
	if c, ok := s.byID[id]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *quoteChainRepoStub) GetByChainID(_ context.Context, chainID string) (*entities.Chain, error) {
	if c, ok := s.byChain[chainID]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *quoteChainRepoStub) GetByCAIP2(_ context.Context, caip2 string) (*entities.Chain, error) {
	if c, ok := s.byCAIP2[caip2]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *quoteChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) { return nil, nil }
func (s *quoteChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *quoteChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *quoteChainRepoStub) Create(context.Context, *entities.Chain) error { return nil }
func (s *quoteChainRepoStub) Update(context.Context, *entities.Chain) error { return nil }
func (s *quoteChainRepoStub) Delete(context.Context, uuid.UUID) error       { return nil }

type quoteContractRepoStub struct {
	router *entities.SmartContract
	err    error
}

func (s *quoteContractRepoStub) Create(context.Context, *entities.SmartContract) error { return nil }
func (s *quoteContractRepoStub) GetByID(context.Context, uuid.UUID) (*entities.SmartContract, error) {
	return nil, errors.New("not found")
}
func (s *quoteContractRepoStub) GetByChainAndAddress(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
	return nil, errors.New("not found")
}
func (s *quoteContractRepoStub) GetActiveContract(_ context.Context, _ uuid.UUID, typ entities.SmartContractType) (*entities.SmartContract, error) {
	if typ == entities.ContractTypeRouter {
		if s.err != nil {
			return nil, s.err
		}
		if s.router != nil {
			return s.router, nil
		}
	}
	return nil, errors.New("not found")
}
func (s *quoteContractRepoStub) GetFiltered(context.Context, *uuid.UUID, entities.SmartContractType, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *quoteContractRepoStub) GetByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *quoteContractRepoStub) GetAll(context.Context, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *quoteContractRepoStub) Update(context.Context, *entities.SmartContract) error { return nil }
func (s *quoteContractRepoStub) SoftDelete(context.Context, uuid.UUID) error            { return nil }

type quoteTokenRepoStub struct{}

func (quoteTokenRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Token, error) { return nil, domainerrors.ErrNotFound }
func (quoteTokenRepoStub) GetBySymbol(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (quoteTokenRepoStub) GetByAddress(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (quoteTokenRepoStub) GetAll(context.Context) ([]*entities.Token, error)                { return nil, nil }
func (quoteTokenRepoStub) GetStablecoins(context.Context) ([]*entities.Token, error)         { return nil, nil }
func (quoteTokenRepoStub) GetNative(context.Context, uuid.UUID) (*entities.Token, error)     { return nil, domainerrors.ErrNotFound }
func (quoteTokenRepoStub) GetTokensByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return []*entities.Token{}, 0, nil
}
func (quoteTokenRepoStub) GetAllTokens(context.Context, *uuid.UUID, *string, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (quoteTokenRepoStub) Create(context.Context, *entities.Token) error { return nil }
func (quoteTokenRepoStub) Update(context.Context, *entities.Token) error { return nil }
func (quoteTokenRepoStub) SoftDelete(context.Context, uuid.UUID) error   { return nil }

func TestPaymentUsecase_GetBridgeFeeQuote_ErrorBranches(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM}
	router := &entities.SmartContract{ContractAddress: "0x1111111111111111111111111111111111111111", Type: entities.ContractTypeRouter}

	t.Run("invalid source", func(t *testing.T) {
		u := &PaymentUsecase{chainRepo: &quoteChainRepoStub{}, chainResolver: NewChainResolver(&quoteChainRepoStub{})}
		_, err := u.getBridgeFeeQuote(context.Background(), "eip155:8453", "eip155:42161", "0x1", "0x2", big.NewInt(1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "source chain config not found")
	})

	t.Run("invalid destination", func(t *testing.T) {
		repo := &quoteChainRepoStub{byCAIP2: map[string]*entities.Chain{"eip155:8453": source}, byID: map[uuid.UUID]*entities.Chain{sourceID: source}}
		u := &PaymentUsecase{chainRepo: repo, chainResolver: NewChainResolver(repo)}
		_, err := u.getBridgeFeeQuote(context.Background(), "eip155:8453", "eip155:42161", "0x1", "0x2", big.NewInt(1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "dest chain config not found")
	})

	t.Run("router missing", func(t *testing.T) {
		repo := &quoteChainRepoStub{
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
			byID:    map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
		}
		u := &PaymentUsecase{
			chainRepo:     repo,
			chainResolver: NewChainResolver(repo),
			contractRepo:  &quoteContractRepoStub{err: errors.New("missing router")},
		}
		_, err := u.getBridgeFeeQuote(context.Background(), "eip155:8453", "eip155:42161", "0x1", "0x2", big.NewInt(1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "active router not found")
	})

	t.Run("no rpc available", func(t *testing.T) {
		repo := &quoteChainRepoStub{
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
			byID:    map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
		}
		u := &PaymentUsecase{
			chainRepo:     repo,
			chainResolver: NewChainResolver(repo),
			contractRepo:  &quoteContractRepoStub{router: router},
			clientFactory: blockchain.NewClientFactory(),
			tokenRepo:     quoteTokenRepoStub{},
		}
		_, err := u.getBridgeFeeQuote(context.Background(), "eip155:8453", "eip155:42161", "0x1", "0x2", big.NewInt(1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "no RPC endpoints available")
	})

	t.Run("all rpc failed", func(t *testing.T) {
		sourceWithBadRPC := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCs: []entities.ChainRPC{{URL: "://bad-1", IsActive: true}, {URL: "://bad-2", IsActive: true}}}
		repo := &quoteChainRepoStub{
			byCAIP2: map[string]*entities.Chain{"eip155:8453": sourceWithBadRPC, "eip155:42161": dest},
			byID:    map[uuid.UUID]*entities.Chain{sourceID: sourceWithBadRPC, destID: dest},
		}
		u := &PaymentUsecase{
			chainRepo:      repo,
			chainResolver:  NewChainResolver(repo),
			contractRepo:   &quoteContractRepoStub{router: router},
			clientFactory:  blockchain.NewClientFactory(),
			tokenRepo:      quoteTokenRepoStub{},
			routePolicyRepo: nil,
		}
		_, err := u.getBridgeFeeQuote(context.Background(), "eip155:8453", "eip155:42161", "0x1", "0x2", big.NewInt(1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to connect to any RPC endpoint")
	})

	t.Run("source chain lookup by id failed", func(t *testing.T) {
		repo := &quoteChainRepoStub{
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
			byID:    map[uuid.UUID]*entities.Chain{destID: dest}, // source intentionally missing
		}
		u := &PaymentUsecase{
			chainRepo:     repo,
			chainResolver: NewChainResolver(repo),
			contractRepo:  &quoteContractRepoStub{router: router},
			clientFactory: blockchain.NewClientFactory(),
		}
		_, err := u.getBridgeFeeQuote(context.Background(), "eip155:8453", "eip155:42161", "0x1", "0x2", big.NewInt(1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "source chain not found")
	})

	t.Run("all bridge quote invalid", func(t *testing.T) {
		rpcURL := "mock://quote-invalid"
		sourceWithRPC := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM, RPCURL: rpcURL}
		repo := &quoteChainRepoStub{
			byCAIP2: map[string]*entities.Chain{"eip155:8453": sourceWithRPC, "eip155:42161": dest},
			byID:    map[uuid.UUID]*entities.Chain{sourceID: sourceWithRPC, destID: dest},
		}

		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(rpcURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{0x00}, nil // fee == 0 -> invalid quote
		}))

		u := &PaymentUsecase{
			chainRepo:      repo,
			chainResolver:  NewChainResolver(repo),
			contractRepo:   &quoteContractRepoStub{router: router},
			clientFactory:  factory,
			routePolicyRepo: nil,
		}
		_, err := u.getBridgeFeeQuote(context.Background(), "eip155:8453", "eip155:42161", "0x1", "0x2", big.NewInt(100))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid fee quote")
	})
}
