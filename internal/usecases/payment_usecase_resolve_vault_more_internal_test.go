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

type approvalNilChainRepoStub struct {
	chain *entities.Chain
}

func (s *approvalNilChainRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Chain, error) {
	return s.chain, nil
}
func (s *approvalNilChainRepoStub) GetByChainID(context.Context, string) (*entities.Chain, error) {
	return nil, errors.New("not found")
}
func (s *approvalNilChainRepoStub) GetByCAIP2(context.Context, string) (*entities.Chain, error) {
	return nil, errors.New("not found")
}
func (s *approvalNilChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) {
	return nil, nil
}
func (s *approvalNilChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *approvalNilChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *approvalNilChainRepoStub) Create(context.Context, *entities.Chain) error       { return nil }
func (s *approvalNilChainRepoStub) Update(context.Context, *entities.Chain) error       { return nil }
func (s *approvalNilChainRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }
func (s *approvalNilChainRepoStub) CreateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *approvalNilChainRepoStub) UpdateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *approvalNilChainRepoStub) DeleteRPC(context.Context, uuid.UUID) error          { return nil }
func (s *approvalNilChainRepoStub) GetRPCByID(context.Context, uuid.UUID) (*entities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}

func TestPaymentUsecase_ResolveVaultAddressForApproval_MoreBranches(t *testing.T) {
	chainID := uuid.New()

	t.Run("contract repo returns nil,nil then chain nil", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return nil, nil
			}},
			chainRepo: &approvalNilChainRepoStub{chain: nil},
		}
		got := u.resolveVaultAddressForApproval(chainID, "0x1111111111111111111111111111111111111111")
		require.Equal(t, "", got)
	})

	t.Run("no active rpc but fallback any rpc url works path", func(t *testing.T) {
		srv := newPaymentRPCServer(t, func(_ int, _ string) string {
			return "0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		})
		defer srv.Close()
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return nil, errors.New("not found")
			}},
			chainRepo: &approvalNilChainRepoStub{chain: &entities.Chain{
				ID:   chainID,
				RPCs: []entities.ChainRPC{{URL: srv.URL, IsActive: false}},
			}},
			clientFactory: blockchain.NewClientFactory(),
		}
		got := u.resolveVaultAddressForApproval(chainID, "0x1111111111111111111111111111111111111111")
		require.Equal(t, "0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa", got)
	})

	t.Run("active rpc mock client returns padded vault address", func(t *testing.T) {
		const rpcURL = "mock://vault-success-line581"
		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(rpcURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0xaa, 0xaa, 0xaa, 0xaa,
				0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa,
				0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa,
			}, nil
		}))
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return nil, errors.New("not found")
			}},
			chainRepo:     &approvalNilChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: rpcURL}},
			clientFactory: factory,
		}
		got := u.resolveVaultAddressForApproval(chainID, "0x1111111111111111111111111111111111111111")
		require.Equal(t, "0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa", got)
	})

	t.Run("gateway empty short-circuit when no vault configured", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return nil, errors.New("not found")
			}},
		}
		got := u.resolveVaultAddressForApproval(chainID, "")
		require.Equal(t, "", got)
	})

	t.Run("call view error returns empty", func(t *testing.T) {
		const rpcURL = "mock://vault-call-error"
		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(rpcURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return nil, errors.New("call failed")
		}))
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return nil, errors.New("not found")
			}},
			chainRepo:     &approvalNilChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: rpcURL}},
			clientFactory: factory,
		}
		got := u.resolveVaultAddressForApproval(chainID, "0x1111111111111111111111111111111111111111")
		require.Equal(t, "", got)
	})

	t.Run("no rpc available returns empty", func(t *testing.T) {
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return nil, errors.New("not found")
			}},
			chainRepo:     &approvalNilChainRepoStub{chain: &entities.Chain{ID: chainID}},
			clientFactory: blockchain.NewClientFactory(),
		}
		got := u.resolveVaultAddressForApproval(chainID, "0x1111111111111111111111111111111111111111")
		require.Equal(t, "", got)
	})
}
