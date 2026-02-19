package usecases

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestPaymentUsecase_ResolveVaultAddress_EmptyGatewayBranch(t *testing.T) {
	scRepo := &scRepoStub{}
	u := &PaymentUsecase{
		contractRepo:     scRepo,
		ABIResolverMixin: NewABIResolverMixin(scRepo),
	}
	got := u.resolveVaultAddressForApproval(uuid.New(), "")
	require.Equal(t, "", got)
}

func TestPaymentUsecase_CalculateOnchainApprovalAmount_MoreErrorBranches(t *testing.T) {
	t.Run("fixed fee call error", func(t *testing.T) {
		srv := newPaymentRPCServer(t, func(callIndex int, _ string) string {
			if callIndex == 1 {
				return "0x" // quoteTotalAmount decode fail -> fallback
			}
			return "0x"
		})
		defer srv.Close()

		chainID := uuid.New()
		scRepo := &scRepoStub{getActiveFn: func(ctx context.Context, chainID uuid.UUID, t entities.SmartContractType) (*entities.SmartContract, error) {
			return nil, domainerrors.ErrNotFound
		}}
		u := &PaymentUsecase{
			contractRepo:     scRepo,
			chainRepo:        &approvalChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: srv.URL}},
			clientFactory:    blockchain.NewClientFactory(),
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
	})

	t.Run("fee bps call error", func(t *testing.T) {
		srv := newPaymentRPCServer(t, func(callIndex int, _ string) string {
			switch callIndex {
			case 1:
				return "0x"
			case 2:
				return mustPackOutputs(t, []string{"uint256"}, big.NewInt(50))
			default:
				return "0x"
			}
		})
		defer srv.Close()

		chainID := uuid.New()
		scRepo := &scRepoStub{getActiveFn: func(ctx context.Context, chainID uuid.UUID, t entities.SmartContractType) (*entities.SmartContract, error) {
			return nil, domainerrors.ErrNotFound
		}}
		u := &PaymentUsecase{
			contractRepo:     scRepo,
			chainRepo:        &approvalChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: srv.URL}},
			clientFactory:    blockchain.NewClientFactory(),
			ABIResolverMixin: NewABIResolverMixin(scRepo),
		}
		_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
			SourceChainID: chainID,
			SourceAmount:  "1000",
			TotalCharged:  "1000",
		}, "0x1111111111111111111111111111111111111111")
		require.Error(t, err)
	})

}

func TestPaymentUsecase_GetBridgeFeeQuote_FirstConnectedRPCCallError(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{
		ID:      sourceID,
		ChainID: "8453",
		Type:    entities.ChainTypeEVM,
		RPCs: []entities.ChainRPC{
			{URL: "mock://fail-first", IsActive: true},
			{URL: "mock://second-ok", IsActive: true},
		},
	}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Type: entities.ChainTypeEVM}
	router := &entities.SmartContract{ContractAddress: "0x1111111111111111111111111111111111111111", Type: entities.ContractTypeRouter}

	chainRepo := &quoteChainRepoStub{
		byCAIP2: map[string]*entities.Chain{
			"eip155:8453":  source,
			"eip155:42161": dest,
		},
		byID: map[uuid.UUID]*entities.Chain{
			sourceID: source,
			destID:   dest,
		},
	}

	factory := blockchain.NewClientFactory()
	factory.RegisterEVMClient("mock://fail-first", blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
		return nil, errors.New("first endpoint down")
	}))
	factory.RegisterEVMClient("mock://second-ok", blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
		// uint256(100) abi-encoded
		return common.FromHex(mustPackOutputs(t, []string{"uint256"}, big.NewInt(100))), nil
	}))

	scRepo := &quoteContractRepoStub{router: router}
	u := &PaymentUsecase{
		chainRepo:        chainRepo,
		chainResolver:    NewChainResolver(chainRepo),
		contractRepo:     scRepo,
		clientFactory:    factory,
		routePolicyRepo:  nil,
		ABIResolverMixin: NewABIResolverMixin(scRepo),
	}

	_, err := u.getBridgeFeeQuote(context.Background(), "eip155:8453", "eip155:42161", "0x1", "0x2", big.NewInt(10), big.NewInt(0))
	require.Error(t, err)
	require.Contains(t, err.Error(), "contract call failed")
}
