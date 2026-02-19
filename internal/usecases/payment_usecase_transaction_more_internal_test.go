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
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestPaymentUsecase_BuildTransactionData_MoreBranches(t *testing.T) {
	contract := &entities.SmartContract{ContractAddress: "0x9999999999999999999999999999999999999999"}

	t.Run("cross-chain fee quote success sets tx value", func(t *testing.T) {
		sourceID := uuid.New()
		destID := uuid.New()
		source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, RPCURL: "mock://base"}
		dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM}
		repo := &quoteChainRepoStub{
			byID:    map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
		}
		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(source.RPCURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return []byte{0x64}, nil // 100 wei
		}))
		u := &PaymentUsecase{
			chainRepo:     repo,
			chainResolver: NewChainResolver(repo),
			contractRepo: &quoteContractRepoStub{
				router: &entities.SmartContract{ContractAddress: "0x1111111111111111111111111111111111111111"},
			},
			clientFactory: factory,
		}

		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        destID,
			SourceTokenAddress: "native",
			DestTokenAddress:   "0x2222222222222222222222222222222222222222",
			ReceiverAddress:    "0x3333333333333333333333333333333333333333",
			SourceAmount:       "1000",
			SourceChain:        source,
			DestChain:          dest,
		}

		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)
		m, ok := out.(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "0x78", m["value"])
	})

	t.Run("cross-chain fee quote zero returns invalid quote error", func(t *testing.T) {
		sourceID := uuid.New()
		destID := uuid.New()
		source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, RPCURL: "mock://zero-fee"}
		dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM}
		repo := &quoteChainRepoStub{
			byID:    map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
		}
		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(source.RPCURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
			return common.FromHex(mustPackOutputs(t, []string{"uint256"}, big.NewInt(0))), nil
		}))
		u := &PaymentUsecase{
			chainRepo:     repo,
			chainResolver: NewChainResolver(repo),
			contractRepo: &quoteContractRepoStub{
				router: &entities.SmartContract{ContractAddress: "0x1111111111111111111111111111111111111111"},
			},
			clientFactory: factory,
		}

		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        destID,
			SourceTokenAddress: "native",
			DestTokenAddress:   "0x2222222222222222222222222222222222222222",
			ReceiverAddress:    "0x3333333333333333333333333333333333333333",
			SourceAmount:       "1000",
			SourceChain:        source,
			DestChain:          dest,
		}

		out, err := u.buildTransactionData(payment, contract)
		require.Nil(t, out)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to resolve bridge fee quote")
	})

	t.Run("source and dest chain resolved from repo when relation missing", func(t *testing.T) {
		sourceID := uuid.New()
		source := &entities.Chain{ID: sourceID, ChainID: "8453", Type: entities.ChainTypeEVM}
		repo := &quoteChainRepoStub{byID: map[uuid.UUID]*entities.Chain{sourceID: source}}
		u := &PaymentUsecase{chainRepo: repo}

		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        sourceID,
			SourceTokenAddress: "native",
			DestTokenAddress:   "native",
			ReceiverAddress:    "0x3333333333333333333333333333333333333333",
			SourceAmount:       "1000",
		}

		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)
		require.NotNil(t, out)
	})

	t.Run("source chain unresolved returns nil payload", func(t *testing.T) {
		u := &PaymentUsecase{chainRepo: &quoteChainRepoStub{}}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      uuid.New(),
			DestChainID:        uuid.New(),
			SourceTokenAddress: "native",
			DestTokenAddress:   "native",
			ReceiverAddress:    "0x3333333333333333333333333333333333333333",
			SourceAmount:       "1000",
		}
		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)
		require.Nil(t, out)
	})

	t.Run("approval path propagates calculateOnchainApprovalAmount error", func(t *testing.T) {
		sourceID := uuid.New()
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
				return &entities.SmartContract{ContractAddress: "0x4444444444444444444444444444444444444444"}, nil
			}},
		}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        sourceID,
			SourceTokenAddress: "0x1111111111111111111111111111111111111111",
			DestTokenAddress:   "0x2222222222222222222222222222222222222222",
			ReceiverAddress:    "0x3333333333333333333333333333333333333333",
			SourceAmount:       "not-number",
			SourceChain:        &entities.Chain{ChainID: "8453", Type: entities.ChainTypeEVM},
			DestChain:          &entities.Chain{ChainID: "8453", Type: entities.ChainTypeEVM},
		}
		out, err := u.buildTransactionData(payment, contract)
		require.Nil(t, out)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid source amount")
	})

	t.Run("cross-chain with approval success includes tx value and approval", func(t *testing.T) {
		sourceID := uuid.New()
		destID := uuid.New()
		source := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeEVM, RPCURL: "mock://full-success"}
		dest := &entities.Chain{ID: destID, ChainID: "42161", Name: "Arbitrum", Type: entities.ChainTypeEVM}
		repo := &quoteChainRepoStub{
			byID:    map[uuid.UUID]*entities.Chain{sourceID: source, destID: dest},
			byCAIP2: map[string]*entities.Chain{"eip155:8453": source, "eip155:42161": dest},
		}
		routerAddr := "0x1111111111111111111111111111111111111111"
		vaultAddr := "0x4444444444444444444444444444444444444444"
		factory := blockchain.NewClientFactory()
		factory.RegisterEVMClient(source.RPCURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, to string, _ []byte) ([]byte, error) {
			if common.HexToAddress(to) == common.HexToAddress(routerAddr) {
				return []byte{0x64}, nil // bridge fee 100 wei
			}
			// gateway quoteTotalAmount => totalAmount=1053, platformFee=53
			return common.FromHex(mustPackOutputs(t, []string{"uint256", "uint256"}, big.NewInt(1053), big.NewInt(53))), nil
		}))
		u := &PaymentUsecase{
			chainRepo:     repo,
			chainResolver: NewChainResolver(repo),
			contractRepo: &scRepoStub{getActiveFn: func(_ context.Context, _ uuid.UUID, typ entities.SmartContractType) (*entities.SmartContract, error) {
				if typ == entities.ContractTypeVault {
					return &entities.SmartContract{ContractAddress: vaultAddr}, nil
				}
				if typ == entities.ContractTypeRouter {
					return &entities.SmartContract{ContractAddress: routerAddr}, nil
				}
				return nil, errors.New("not found")
			}},
			clientFactory: factory,
			ABIResolverMixin: NewABIResolverMixin(&scRepoStub{getActiveFn: func(_ context.Context, _ uuid.UUID, typ entities.SmartContractType) (*entities.SmartContract, error) {
				if typ == entities.ContractTypeVault {
					return &entities.SmartContract{ContractAddress: vaultAddr}, nil
				}
				if typ == entities.ContractTypeRouter {
					return &entities.SmartContract{ContractAddress: routerAddr}, nil
				}
				return nil, errors.New("not found")
			}}),
		}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        destID,
			SourceTokenAddress: "0x2222222222222222222222222222222222222222",
			DestTokenAddress:   "0x3333333333333333333333333333333333333333",
			ReceiverAddress:    "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			SourceAmount:       "1000",
			TotalCharged:       "1053",
			SourceChain:        source,
			DestChain:          dest,
		}

		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)
		m, ok := out.(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "0x78", m["value"])
		approval, ok := m["approval"].(map[string]string)
		require.True(t, ok)
		require.Equal(t, "1053", approval["amount"])
		require.Equal(t, vaultAddr, approval["spender"])
		txs, ok := m["transactions"].([]map[string]string)
		require.True(t, ok)
		require.Len(t, txs, 2)
		require.Equal(t, "approve", txs[0]["kind"])
		require.Equal(t, "createPayment", txs[1]["kind"])
	})
}

func TestPaymentUsecase_ResolveVaultAddressForApproval_UsesActiveRPCList(t *testing.T) {
	vault := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	padded := common.LeftPadBytes(vault.Bytes(), 32)
	srv := newPaymentRPCServer(t, func(_ int, _ string) string {
		return "0x" + common.Bytes2Hex(padded)
	})
	defer srv.Close()

	chainID := uuid.New()
	u := &PaymentUsecase{
		contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
			return nil, errors.New("not found")
		}},
		chainRepo: &approvalChainRepoStub{chain: &entities.Chain{
			ID:     chainID,
			RPCURL: "",
			RPCs:   []entities.ChainRPC{{URL: srv.URL, IsActive: true}},
		}},
		clientFactory: blockchain.NewClientFactory(),
	}

	got := u.resolveVaultAddressForApproval(chainID, "0x1111111111111111111111111111111111111111")
	require.Equal(t, vault.Hex(), got)
}

func TestPaymentUsecase_CalculateOnchainApprovalAmount_ClientCreateError(t *testing.T) {
	chainID := uuid.New()
	u := &PaymentUsecase{
		chainRepo: &approvalChainRepoStub{chain: &entities.Chain{
			ID:     chainID,
			RPCURL: "://bad-rpc-url",
		}},
		clientFactory: blockchain.NewClientFactory(),
		contractRepo: &scRepoStub{getActiveFn: func(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
			return nil, errors.New("not found")
		}},
		ABIResolverMixin: NewABIResolverMixin(&scRepoStub{}),
	}
	_, err := u.calculateOnchainApprovalAmount(&entities.Payment{
		SourceChainID: chainID,
		SourceAmount:  "1000",
		TotalCharged:  "1000",
	}, "0x1111111111111111111111111111111111111111")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create evm client for approval quote")
}

func TestPaymentUsecase_QuoteBridgeFeeByType_CallError(t *testing.T) {
	u := &PaymentUsecase{}
	client := blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(context.Context, string, []byte) ([]byte, error) {
		return nil, errors.New("rpc call failed")
	})
	_, err := u.quoteBridgeFeeByType(
		context.Background(),
		client,
		"0x1111111111111111111111111111111111111111",
		"eip155:42161",
		0,
		"0x2222222222222222222222222222222222222222",
		"0x3333333333333333333333333333333333333333",
		big.NewInt(1000),
		big.NewInt(0),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "contract call failed")
}
