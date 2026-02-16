package usecases

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestCrosschainConfigUsecase_CheckFeeQuoteHealth_Matrix(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, RPCURL: "mock://fee-health"}
	dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM}

	tokenRepo := &ccTokenRepoStub{
		byChain: map[uuid.UUID][]*entities.Token{
			sourceID: {&entities.Token{ID: uuid.New(), ChainUUID: sourceID, ContractAddress: "0x1111111111111111111111111111111111111111"}},
			destID:   {&entities.Token{ID: uuid.New(), ChainUUID: destID, ContractAddress: "0x2222222222222222222222222222222222222222"}},
		},
	}

	factory := blockchain.NewClientFactory()
	factory.RegisterEVMClient(source.RPCURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
		return []byte{0x01}, nil
	}))

	contractRepo := &ccContractRepoStub{
		active: map[string]*entities.SmartContract{
			contractKey(sourceID, entities.ContractTypeRouter): &entities.SmartContract{
				ID:              uuid.New(),
				ChainUUID:       sourceID,
				Type:            entities.ContractTypeRouter,
				ContractAddress: "0x00000000000000000000000000000000000000b2",
				IsActive:        true,
			},
		},
	}
	u := &CrosschainConfigUsecase{
		contractRepo:  contractRepo,
		tokenRepo:     tokenRepo,
		clientFactory: factory,
	}

	require.False(t, u.checkFeeQuoteHealth(context.Background(), nil, dest, 0))
	require.False(t, u.checkFeeQuoteHealth(context.Background(), source, nil, 0))
	require.True(t, u.checkFeeQuoteHealth(context.Background(), source, dest, 0))

	contractRepoNoRouter := &ccContractRepoStub{active: map[string]*entities.SmartContract{}}
	uNoRouter := &CrosschainConfigUsecase{
		contractRepo:  contractRepoNoRouter,
		tokenRepo:     tokenRepo,
		clientFactory: factory,
	}
	require.False(t, uNoRouter.checkFeeQuoteHealth(context.Background(), source, dest, 0))

	sourceNoRPC := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM}
	require.False(t, u.checkFeeQuoteHealth(context.Background(), sourceNoRPC, dest, 0))

	factoryErr := blockchain.NewClientFactory()
	factoryErr.RegisterEVMClient(source.RPCURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
		return nil, errors.New("call failed")
	}))
	uErr := &CrosschainConfigUsecase{
		contractRepo:  contractRepo,
		tokenRepo:     tokenRepo,
		clientFactory: factoryErr,
	}
	require.False(t, uErr.checkFeeQuoteHealth(context.Background(), source, dest, 0))
}
