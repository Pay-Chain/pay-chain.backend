package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

type createPaymentTokenRepoStub struct {
	byAddress map[string]*entities.Token
	native    *entities.Token
}

func (s *createPaymentTokenRepoStub) key(chainID uuid.UUID, addr string) string {
	return chainID.String() + "|" + addr
}

func (s *createPaymentTokenRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *createPaymentTokenRepoStub) GetBySymbol(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *createPaymentTokenRepoStub) GetByAddress(_ context.Context, address string, chainID uuid.UUID) (*entities.Token, error) {
	if s.byAddress == nil {
		return nil, domainerrors.ErrNotFound
	}
	if tok, ok := s.byAddress[s.key(chainID, address)]; ok {
		return tok, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *createPaymentTokenRepoStub) GetAll(context.Context) ([]*entities.Token, error) { return nil, nil }
func (s *createPaymentTokenRepoStub) GetStablecoins(context.Context) ([]*entities.Token, error) {
	return nil, nil
}
func (s *createPaymentTokenRepoStub) GetNative(context.Context, uuid.UUID) (*entities.Token, error) {
	if s.native != nil {
		return s.native, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *createPaymentTokenRepoStub) GetTokensByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (s *createPaymentTokenRepoStub) GetAllTokens(context.Context, *uuid.UUID, *string, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (s *createPaymentTokenRepoStub) Create(context.Context, *entities.Token) error { return nil }
func (s *createPaymentTokenRepoStub) Update(context.Context, *entities.Token) error { return nil }
func (s *createPaymentTokenRepoStub) SoftDelete(context.Context, uuid.UUID) error   { return nil }

type createPaymentRepoStub struct {
	createErr error
	created   *entities.Payment
}

func (s *createPaymentRepoStub) Create(_ context.Context, payment *entities.Payment) error {
	s.created = payment
	return s.createErr
}
func (s *createPaymentRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Payment, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *createPaymentRepoStub) GetByUserID(context.Context, uuid.UUID, int, int) ([]*entities.Payment, int, error) {
	return nil, 0, nil
}
func (s *createPaymentRepoStub) GetByMerchantID(context.Context, uuid.UUID, int, int) ([]*entities.Payment, int, error) {
	return nil, 0, nil
}
func (s *createPaymentRepoStub) UpdateStatus(context.Context, uuid.UUID, entities.PaymentStatus) error {
	return nil
}
func (s *createPaymentRepoStub) UpdateDestTxHash(context.Context, uuid.UUID, string) error { return nil }
func (s *createPaymentRepoStub) MarkRefunded(context.Context, uuid.UUID) error              { return nil }

type createPaymentEventRepoStub struct {
	createErr error
	created   *entities.PaymentEvent
}

func (s *createPaymentEventRepoStub) Create(_ context.Context, event *entities.PaymentEvent) error {
	s.created = event
	return s.createErr
}
func (s *createPaymentEventRepoStub) GetByPaymentID(context.Context, uuid.UUID) ([]*entities.PaymentEvent, error) {
	return nil, nil
}
func (s *createPaymentEventRepoStub) GetLatestByPaymentID(context.Context, uuid.UUID) (*entities.PaymentEvent, error) {
	return nil, domainerrors.ErrNotFound
}

type createPaymentUOWStub struct {
	doErr error
}

func (s *createPaymentUOWStub) Do(ctx context.Context, fn func(context.Context) error) error {
	if s.doErr != nil {
		return s.doErr
	}
	return fn(ctx)
}
func (s *createPaymentUOWStub) WithLock(ctx context.Context) context.Context { return ctx }

func TestPaymentUsecase_CreatePayment_ValidationAndResolutionErrors(t *testing.T) {
	u := &PaymentUsecase{}

	_, err := u.CreatePayment(context.Background(), uuid.New(), &entities.CreatePaymentInput{})
	require.ErrorIs(t, err, domainerrors.ErrBadRequest)

	_, err = u.CreatePayment(context.Background(), uuid.New(), &entities.CreatePaymentInput{
		SourceChainID: "eip155:8453",
		DestChainID:   "eip155:42161",
	})
	require.ErrorIs(t, err, domainerrors.ErrBadRequest)

	repo := &quoteChainRepoStub{}
	u = &PaymentUsecase{chainRepo: repo, chainResolver: NewChainResolver(repo)}
	_, err = u.CreatePayment(context.Background(), uuid.New(), &entities.CreatePaymentInput{
		SourceChainID:      "eip155:8453",
		DestChainID:        "eip155:42161",
		ReceiverAddress:    "0xabc",
		SourceTokenAddress: "0x1",
		DestTokenAddress:   "0x2",
		Amount:             "1",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid source chain")
}

func TestPaymentUsecase_CreatePayment_TokenAndAmountErrors(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM}
	chainRepo := &quoteChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  source,
			"eip155:42161": dest,
		},
	}

	u := &PaymentUsecase{
		chainRepo:     chainRepo,
		chainResolver: NewChainResolver(chainRepo),
		tokenRepo:     &createPaymentTokenRepoStub{},
		contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
			return nil, domainerrors.ErrNotFound
		}},
	}
	_, err := u.CreatePayment(context.Background(), uuid.New(), &entities.CreatePaymentInput{
		SourceChainID:      "eip155:8453",
		DestChainID:        "eip155:42161",
		SourceTokenAddress: "0xsource",
		DestTokenAddress:   "0xdest",
		ReceiverAddress:    "0xreceiver",
		Amount:             "1",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "source token not found")

	srcTok := &entities.Token{ID: uuid.New(), Decimals: 6, ContractAddress: "0xsource", ChainUUID: sourceID}
	tokenRepo := &createPaymentTokenRepoStub{
		byAddress: map[string]*entities.Token{
			sourceID.String() + "|0xsource": srcTok,
		},
	}
	u.tokenRepo = tokenRepo
	_, err = u.CreatePayment(context.Background(), uuid.New(), &entities.CreatePaymentInput{
		SourceChainID:      "eip155:8453",
		DestChainID:        "eip155:42161",
		SourceTokenAddress: "0xsource",
		DestTokenAddress:   "0xdest",
		ReceiverAddress:    "0xreceiver",
		Amount:             "1",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "dest token not found")

	dstTok := &entities.Token{ID: uuid.New(), Decimals: 6, ContractAddress: "0xdest", ChainUUID: destID}
	tokenRepo.byAddress[destID.String()+"|0xdest"] = dstTok

	_, err = u.CreatePayment(context.Background(), uuid.New(), &entities.CreatePaymentInput{
		SourceChainID:      "eip155:8453",
		DestChainID:        "eip155:42161",
		SourceTokenAddress: "0xsource",
		DestTokenAddress:   "0xdest",
		ReceiverAddress:    "0xreceiver",
		Amount:             "1",
		Decimals:           18,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "source token decimals mismatch")

	_, err = u.CreatePayment(context.Background(), uuid.New(), &entities.CreatePaymentInput{
		SourceChainID:      "eip155:8453",
		DestChainID:        "eip155:42161",
		SourceTokenAddress: "0xsource",
		DestTokenAddress:   "0xdest",
		ReceiverAddress:    "0xreceiver",
		Amount:             "invalid-number",
		Decimals:           6,
	})
	require.ErrorIs(t, err, domainerrors.ErrBadRequest)
}

func TestPaymentUsecase_CreatePayment_UOWAndEventBranches(t *testing.T) {
	sourceID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
	chainRepo := &quoteChainRepoStub{
		byID: map[uuid.UUID]*entities.Chain{sourceID: source},
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453": source,
		},
	}
	srcTok := &entities.Token{ID: uuid.New(), Decimals: 6, ContractAddress: "0xsource", ChainUUID: sourceID}
	dstTok := &entities.Token{ID: uuid.New(), Decimals: 6, ContractAddress: "0xdest", ChainUUID: sourceID}
	tokenRepo := &createPaymentTokenRepoStub{
		byAddress: map[string]*entities.Token{
			sourceID.String() + "|0xsource": srcTok,
			sourceID.String() + "|0xdest":   dstTok,
		},
	}

	paymentRepo := &createPaymentRepoStub{}
	eventRepo := &createPaymentEventRepoStub{createErr: errors.New("event insert failed")}
	uow := &createPaymentUOWStub{doErr: errors.New("tx failed")}
	u := &PaymentUsecase{
		paymentRepo:      paymentRepo,
		paymentEventRepo: eventRepo,
		chainRepo:        chainRepo,
		chainResolver:    NewChainResolver(chainRepo),
		tokenRepo:        tokenRepo,
		contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
			return nil, domainerrors.ErrNotFound
		}},
		uow: uow,
	}

	req := &entities.CreatePaymentInput{
		SourceChainID:      "eip155:8453",
		DestChainID:        "eip155:8453",
		SourceTokenAddress: "0xsource",
		DestTokenAddress:   "0xdest",
		ReceiverAddress:    "0xreceiver",
		Amount:             "1",
		Decimals:           6,
	}

	_, err := u.CreatePayment(context.Background(), uuid.New(), req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tx failed")

	uow.doErr = nil
	resp, err := u.CreatePayment(context.Background(), uuid.New(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, paymentRepo.created)
	require.NotNil(t, eventRepo.created)
}
