package usecases

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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
	factory.RegisterEVMClient(source.RPCURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, _ string, data []byte) ([]byte, error) {
		if len(data) >= 4 {
			sel := hex.EncodeToString(data[:4])
			switch sel {
			case "fbfa77cf": // vault()
				return commonAddressWord("0x00000000000000000000000000000000000000aa"), nil
			case "374f353f": // getAdapter(string,uint8)
				return commonAddressWord("0x00000000000000000000000000000000000000bb"), nil
			case "de50c31d": // authorizedSpenders(address)
				return commonBoolWord(true), nil
			}
		}
		// generic non-empty uint256-compatible response for quote paths
		return commonBoolWord(true), nil
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
			contractKey(sourceID, entities.ContractTypeGateway): &entities.SmartContract{
				ID:              uuid.New(),
				ChainUUID:       sourceID,
				Type:            entities.ContractTypeGateway,
				ContractAddress: "0x00000000000000000000000000000000000000b1",
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

	sourceBadRPC := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, RPCURL: "://bad-rpc"}
	require.False(t, u.checkFeeQuoteHealth(context.Background(), sourceBadRPC, dest, 0))

	factoryEmpty := blockchain.NewClientFactory()
	factoryEmpty.RegisterEVMClient(source.RPCURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
		return []byte{}, nil
	}))
	uEmpty := &CrosschainConfigUsecase{
		contractRepo:  contractRepo,
		tokenRepo:     tokenRepo,
		clientFactory: factoryEmpty,
	}
	require.False(t, uEmpty.checkFeeQuoteHealth(context.Background(), source, dest, 0))
}

func commonAddressWord(addr string) []byte {
	out := make([]byte, 32)
	copy(out[12:], common.HexToAddress(addr).Bytes())
	return out
}

func commonBoolWord(v bool) []byte {
	out := make([]byte, 32)
	if v {
		out[31] = 1
	}
	return out
}
