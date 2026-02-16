package usecases

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
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
}
