package usecases

import (
	"context"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

func TestPaymentUsecase_BuildTransactionData_Branches(t *testing.T) {
	sourceID := uuid.New()
	destID := uuid.New()
	contract := &entities.SmartContract{ContractAddress: "0x9999999999999999999999999999999999999999"}

	t.Run("contract nil", func(t *testing.T) {
		u := &PaymentUsecase{}
		out, err := u.buildTransactionData(&entities.Payment{}, nil)
		require.NoError(t, err)
		require.Nil(t, out)
	})

	t.Run("evm same chain no approval", func(t *testing.T) {
		u := &PaymentUsecase{}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        sourceID,
			SourceTokenAddress: "native",
			DestTokenAddress:   "0x1111111111111111111111111111111111111111",
			ReceiverAddress:    "0x2222222222222222222222222222222222222222",
			SourceAmount:       "1000",
			SourceChain:        &entities.Chain{ChainID: "8453", Type: entities.ChainTypeEVM},
			DestChain:          &entities.Chain{ChainID: "8453", Type: entities.ChainTypeEVM},
		}
		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)
		m, ok := out.(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, contract.ContractAddress, m["to"])
		require.Equal(t, "0x0", m["value"])
		txs, ok := m["transactions"].([]map[string]string)
		require.True(t, ok)
		require.Len(t, txs, 1)
		require.Equal(t, "createPayment", txs[0]["kind"])
	})

	t.Run("evm approval required but vault missing", func(t *testing.T) {
		chainRepo := &approvalChainRepoStub{chain: nil}
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{},
			chainRepo:    chainRepo,
		}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        sourceID,
			SourceTokenAddress: "0x3333333333333333333333333333333333333333",
			DestTokenAddress:   "0x1111111111111111111111111111111111111111",
			ReceiverAddress:    "0x2222222222222222222222222222222222222222",
			SourceAmount:       "1000",
			SourceChain:        &entities.Chain{ChainID: "8453", Type: entities.ChainTypeEVM},
			DestChain:          &entities.Chain{ChainID: "8453", Type: entities.ChainTypeEVM},
		}
		out, err := u.buildTransactionData(payment, contract)
		require.Nil(t, out)
		require.Error(t, err)
		require.Contains(t, err.Error(), "vault contract address is not configured")
	})

	t.Run("solana chain", func(t *testing.T) {
		u := &PaymentUsecase{}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        destID,
			SourceTokenAddress: "So11111111111111111111111111111111111111112",
			DestTokenAddress:   "So11111111111111111111111111111111111111112",
			ReceiverAddress:    "11111111111111111111111111111111",
			SourceAmount:       "1",
			SourceChain:        &entities.Chain{ChainID: "mainnet", Type: entities.ChainTypeSVM},
			DestChain:          &entities.Chain{ChainID: "mainnet", Type: entities.ChainTypeSVM},
		}
		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)
		m, ok := out.(map[string]string)
		require.True(t, ok)
		require.Equal(t, contract.ContractAddress, m["programId"])
		require.NotEmpty(t, m["data"])
	})

	t.Run("evm cross chain fee quote failed", func(t *testing.T) {
		chainRepo := &approvalChainRepoStub{chain: nil}
		u := &PaymentUsecase{
			chainRepo:     chainRepo,
			chainResolver: NewChainResolver(chainRepo),
		}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        destID,
			SourceTokenAddress: "native",
			DestTokenAddress:   "0x1111111111111111111111111111111111111111",
			ReceiverAddress:    "0x2222222222222222222222222222222222222222",
			SourceAmount:       "1000",
			SourceChain:        &entities.Chain{ChainID: "8453", Type: entities.ChainTypeEVM},
			DestChain:          &entities.Chain{ChainID: "42161", Type: entities.ChainTypeEVM},
		}
		out, err := u.buildTransactionData(payment, contract)
		require.Nil(t, out)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to resolve bridge fee quote")
	})

	t.Run("evm cross chain invalid source amount", func(t *testing.T) {
		u := &PaymentUsecase{}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        destID,
			SourceTokenAddress: "native",
			DestTokenAddress:   "0x1111111111111111111111111111111111111111",
			ReceiverAddress:    "0x2222222222222222222222222222222222222222",
			SourceAmount:       "not-a-number",
			SourceChain:        &entities.Chain{ChainID: "8453", Type: entities.ChainTypeEVM},
			DestChain:          &entities.Chain{ChainID: "42161", Type: entities.ChainTypeEVM},
		}
		out, err := u.buildTransactionData(payment, contract)
		require.Nil(t, out)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid source amount for bridge fee quote")
	})

	t.Run("evm chain with unknown network still returns payload", func(t *testing.T) {
		u := &PaymentUsecase{}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        destID,
			SourceTokenAddress: "native",
			DestTokenAddress:   "native",
			ReceiverAddress:    "receiver",
			SourceAmount:       "1",
			SourceChain:        &entities.Chain{ChainID: "unknown", Type: entities.ChainTypeEVM},
			DestChain:          &entities.Chain{ChainID: "unknown", Type: entities.ChainTypeEVM},
		}
		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)
		m, ok := out.(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "0x0", m["value"])
	})

	t.Run("unknown chain type returns nil payload", func(t *testing.T) {
		u := &PaymentUsecase{}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      sourceID,
			DestChainID:        destID,
			SourceTokenAddress: "native",
			DestTokenAddress:   "native",
			ReceiverAddress:    "receiver",
			SourceAmount:       "1",
			SourceChain:        &entities.Chain{ChainID: "cosmos:osmosis-1", Type: entities.ChainTypeSubstrate},
			DestChain:          &entities.Chain{ChainID: "cosmos:osmosis-1", Type: entities.ChainTypeSubstrate},
		}
		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)
		require.Nil(t, out)
	})

	t.Run("evm same-chain with source/dest resolved from repo", func(t *testing.T) {
		srcID := uuid.New()
		dstID := uuid.New()
		src := &entities.Chain{ID: srcID, ChainID: "8453", Type: entities.ChainTypeEVM}
		dst := &entities.Chain{ID: dstID, ChainID: "8453", Type: entities.ChainTypeEVM}

		chainRepo := &quoteChainRepoStub{
			byID: map[uuid.UUID]*entities.Chain{
				srcID: src,
				dstID: dst,
			},
		}
		u := &PaymentUsecase{chainRepo: chainRepo}
		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      srcID,
			DestChainID:        dstID,
			SourceTokenAddress: "native",
			DestTokenAddress:   "0x1111111111111111111111111111111111111111",
			ReceiverAddress:    "0x2222222222222222222222222222222222222222",
			SourceAmount:       "1000",
		}
		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)
		m, ok := out.(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "0x0", m["value"])
	})

	t.Run("evm approval path includes approve + createPayment txs", func(t *testing.T) {
		srv := newPaymentRPCServer(t, func(callIndex int, _ string) string {
			if callIndex == 1 {
				return mustPackOutputs(t, []string{"uint256", "uint256"}, big.NewInt(1053), big.NewInt(53))
			}
			return "0x"
		})
		defer srv.Close()

		chainID := uuid.New()
		u := &PaymentUsecase{
			contractRepo: &scRepoStub{getActiveFn: func(_ context.Context, _ uuid.UUID, t entities.SmartContractType) (*entities.SmartContract, error) {
				if t == entities.ContractTypeVault {
					return &entities.SmartContract{ContractAddress: "0x4444444444444444444444444444444444444444"}, nil
				}
				return nil, domainerrors.ErrNotFound
			}},
			chainRepo:     &approvalChainRepoStub{chain: &entities.Chain{ID: chainID, RPCURL: srv.URL}},
			clientFactory: blockchain.NewClientFactory(),
		}

		payment := &entities.Payment{
			ID:                 uuid.New(),
			SourceChainID:      chainID,
			DestChainID:        chainID,
			SourceTokenAddress: "0x3333333333333333333333333333333333333333",
			DestTokenAddress:   "0x1111111111111111111111111111111111111111",
			ReceiverAddress:    "0x2222222222222222222222222222222222222222",
			SourceAmount:       "1000",
			TotalCharged:       "1053",
			SourceChain:        &entities.Chain{ChainID: "eip155:8453", Type: entities.ChainTypeEVM},
			DestChain:          &entities.Chain{ChainID: "eip155:8453", Type: entities.ChainTypeEVM},
		}
		out, err := u.buildTransactionData(payment, contract)
		require.NoError(t, err)

		m, ok := out.(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "0x0", m["value"])

		approval, ok := m["approval"].(map[string]string)
		require.True(t, ok)
		require.Equal(t, "approve", approval["kind"])
		require.Equal(t, payment.SourceTokenAddress, approval["to"])
		require.Equal(t, "1053", approval["amount"])

		txs, ok := m["transactions"].([]map[string]string)
		require.True(t, ok)
		require.Len(t, txs, 2)
		require.Equal(t, "approve", txs[0]["kind"])
		require.Equal(t, "createPayment", txs[1]["kind"])
	})
}
