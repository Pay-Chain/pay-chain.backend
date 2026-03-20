package usecases

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	domainentities "payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/pkg/utils"
)

type partnerQuoteRepoStub struct {
	created *domainentities.PaymentQuote
	err     error
}

func (s *partnerQuoteRepoStub) Create(ctx context.Context, quote *domainentities.PaymentQuote) error {
	s.created = quote
	return s.err
}
func (s *partnerQuoteRepoStub) GetByID(context.Context, uuid.UUID) (*domainentities.PaymentQuote, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *partnerQuoteRepoStub) UpdateStatus(context.Context, uuid.UUID, domainentities.PaymentQuoteStatus) error {
	return nil
}
func (s *partnerQuoteRepoStub) MarkUsed(context.Context, uuid.UUID) error { return nil }
func (s *partnerQuoteRepoStub) GetExpiredActive(context.Context, int) ([]*domainentities.PaymentQuote, error) {
	return nil, nil
}
func (s *partnerQuoteRepoStub) ExpireQuotes(context.Context, []uuid.UUID) error { return nil }

type partnerQuoteTokenRepoStub struct {
	byAddress map[string]*domainentities.Token
	bySymbol  map[string]*domainentities.Token
}

func (s *partnerQuoteTokenRepoStub) GetByID(context.Context, uuid.UUID) (*domainentities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *partnerQuoteTokenRepoStub) GetBySymbol(ctx context.Context, symbol string, chainID uuid.UUID) (*domainentities.Token, error) {
	if tok, ok := s.bySymbol[symbol]; ok {
		return tok, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *partnerQuoteTokenRepoStub) GetByAddress(ctx context.Context, address string, chainID uuid.UUID) (*domainentities.Token, error) {
	if tok, ok := s.byAddress[address]; ok {
		return tok, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *partnerQuoteTokenRepoStub) GetAll(context.Context) ([]*domainentities.Token, error) {
	return nil, nil
}
func (s *partnerQuoteTokenRepoStub) GetStablecoins(context.Context) ([]*domainentities.Token, error) {
	return nil, nil
}
func (s *partnerQuoteTokenRepoStub) GetNative(context.Context, uuid.UUID) (*domainentities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *partnerQuoteTokenRepoStub) GetTokensByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*domainentities.Token, int64, error) {
	return nil, 0, nil
}
func (s *partnerQuoteTokenRepoStub) GetAllTokens(context.Context, *uuid.UUID, *string, utils.PaginationParams) ([]*domainentities.Token, int64, error) {
	return nil, 0, nil
}
func (s *partnerQuoteTokenRepoStub) Create(context.Context, *domainentities.Token) error { return nil }
func (s *partnerQuoteTokenRepoStub) Update(context.Context, *domainentities.Token) error { return nil }
func (s *partnerQuoteTokenRepoStub) SoftDelete(context.Context, uuid.UUID) error         { return nil }

type partnerQuoteChainRepoStub struct {
	chain *domainentities.Chain
}

func (s *partnerQuoteChainRepoStub) GetByID(context.Context, uuid.UUID) (*domainentities.Chain, error) {
	return s.chain, nil
}
func (s *partnerQuoteChainRepoStub) GetByChainID(context.Context, string) (*domainentities.Chain, error) {
	return s.chain, nil
}
func (s *partnerQuoteChainRepoStub) GetByCAIP2(context.Context, string) (*domainentities.Chain, error) {
	return s.chain, nil
}
func (s *partnerQuoteChainRepoStub) GetAll(context.Context) ([]*domainentities.Chain, error) {
	return nil, nil
}
func (s *partnerQuoteChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*domainentities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *partnerQuoteChainRepoStub) GetRPCByID(context.Context, uuid.UUID) (*domainentities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *partnerQuoteChainRepoStub) CreateRPC(context.Context, *domainentities.ChainRPC) error {
	return nil
}
func (s *partnerQuoteChainRepoStub) UpdateRPC(context.Context, *domainentities.ChainRPC) error {
	return nil
}
func (s *partnerQuoteChainRepoStub) DeleteRPC(context.Context, uuid.UUID) error { return nil }
func (s *partnerQuoteChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*domainentities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *partnerQuoteChainRepoStub) Create(context.Context, *domainentities.Chain) error { return nil }
func (s *partnerQuoteChainRepoStub) Update(context.Context, *domainentities.Chain) error { return nil }
func (s *partnerQuoteChainRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }

func TestPartnerQuoteUsecase_CreateQuote_SupportedPair(t *testing.T) {
	chainID := uuid.New()
	quoteRepo := &partnerQuoteRepoStub{}
	tokenRepo := &partnerQuoteTokenRepoStub{
		byAddress: map[string]*domainentities.Token{
			"0xusdc": {ChainUUID: chainID, ContractAddress: "0xusdc", Symbol: "USDC", Decimals: 6, IsActive: true},
			"0xidrx": {ChainUUID: chainID, ContractAddress: "0xidrx", Symbol: "IDRX", Decimals: 2, IsActive: true},
		},
		bySymbol: map[string]*domainentities.Token{
			"USDC": {ChainUUID: chainID, ContractAddress: "0xusdc", Symbol: "USDC", Decimals: 6, IsActive: true},
			"IDRX": {ChainUUID: chainID, ContractAddress: "0xidrx", Symbol: "IDRX", Decimals: 2, IsActive: true},
		},
	}
	chainRepo := &partnerQuoteChainRepoStub{
		chain: &domainentities.Chain{ID: chainID, ChainID: "8453", Name: "Base", Type: domainentities.ChainTypeEVM},
	}

	uc := NewPartnerQuoteUsecase(quoteRepo, tokenRepo, chainRepo, nil)
	uc.routeSupportFn = func(ctx context.Context, chainID uuid.UUID, tokenIn string, tokenOut string) (*TokenRouteSupportStatus, error) {
		return &TokenRouteSupportStatus{
			Exists:       true,
			IsDirect:     true,
			Path:         []string{"0xidrx", "0xusdc"},
			Executable:   true,
			UniversalV4:  "0xrouterv4",
			SwapRouterV3: "",
		}, nil
	}
	uc.swapQuoteFn = func(ctx context.Context, chainID uuid.UUID, tokenIn string, tokenOut string, amountIn *big.Int) (*big.Int, error) {
		return big.NewInt(2950000), nil
	}

	out, err := uc.CreateQuote(context.Background(), &CreatePartnerQuoteInput{
		MerchantID:      uuid.New(),
		InvoiceCurrency: "IDRX",
		InvoiceAmount:   "5000000",
		SelectedChain:   "eip155:8453",
		SelectedToken:   "0xusdc",
		DestWallet:      "0xmerchant",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, "USDC", out.SelectedTokenSymbol)
	require.Equal(t, "2950000", out.QuotedAmount)
	require.Equal(t, "0.000059", out.QuoteRate)
	require.Equal(t, "IDRX->USDC", out.Route)
	require.Equal(t, "uniswap-v4-base-usdc-idrx", out.PriceSource)
	require.NotNil(t, quoteRepo.created)
	require.Equal(t, domainentities.PaymentQuoteStatusActive, quoteRepo.created.Status)
}

func TestPartnerQuoteUsecase_CreateQuote_UnsupportedPair(t *testing.T) {
	chainID := uuid.New()
	quoteRepo := &partnerQuoteRepoStub{}
	tokenRepo := &partnerQuoteTokenRepoStub{
		byAddress: map[string]*domainentities.Token{
			"0xusdc": {ChainUUID: chainID, ContractAddress: "0xusdc", Symbol: "USDC", Decimals: 6, IsActive: true},
			"0xidrx": {ChainUUID: chainID, ContractAddress: "0xidrx", Symbol: "IDRX", Decimals: 2, IsActive: true},
		},
		bySymbol: map[string]*domainentities.Token{
			"USDC": {ChainUUID: chainID, ContractAddress: "0xusdc", Symbol: "USDC", Decimals: 6, IsActive: true},
			"IDRX": {ChainUUID: chainID, ContractAddress: "0xidrx", Symbol: "IDRX", Decimals: 2, IsActive: true},
		},
	}
	chainRepo := &partnerQuoteChainRepoStub{
		chain: &domainentities.Chain{ID: chainID, ChainID: "8453", Name: "Base", Type: domainentities.ChainTypeEVM},
	}

	uc := NewPartnerQuoteUsecase(quoteRepo, tokenRepo, chainRepo, nil)
	uc.routeSupportFn = func(ctx context.Context, chainID uuid.UUID, tokenIn string, tokenOut string) (*TokenRouteSupportStatus, error) {
		return &TokenRouteSupportStatus{Exists: false, Executable: false}, nil
	}
	uc.swapQuoteFn = func(ctx context.Context, chainID uuid.UUID, tokenIn string, tokenOut string, amountIn *big.Int) (*big.Int, error) {
		return nil, errors.New("should not be called")
	}

	out, err := uc.CreateQuote(context.Background(), &CreatePartnerQuoteInput{
		MerchantID:      uuid.New(),
		InvoiceCurrency: "IDRX",
		InvoiceAmount:   "5000000",
		SelectedChain:   "eip155:8453",
		SelectedToken:   "0xusdc",
		DestWallet:      "0xmerchant",
	})
	require.Nil(t, out)
	require.Error(t, err)
	var appErr *domainerrors.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, 400, appErr.Status)
}
