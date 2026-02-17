package usecases

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestPaymentRequestUsecase_BuildTransactionData_Branches(t *testing.T) {
	uc := &PaymentRequestUsecase{}

	t.Run("unknown chain type keeps hex and base58 empty", func(t *testing.T) {
		req := &entities.PaymentRequest{
			ID:        uuid.New(),
			NetworkID: "unknown:123",
			Amount:    "1000",
			Decimals:  6,
		}
		tx := uc.buildTransactionData(req, nil)
		require.NotNil(t, tx)
		require.Equal(t, req.ID.String(), tx.RequestID)
		require.Equal(t, "unknown:123", tx.ChainID)
		require.Empty(t, tx.Hex)
		require.Empty(t, tx.Base58)
		require.Empty(t, tx.ContractAddress)
	})

	t.Run("solana chain with contract fills base58 and program fields", func(t *testing.T) {
		req := &entities.PaymentRequest{
			ID:        uuid.New(),
			NetworkID: "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp",
			Amount:    "1000",
			Decimals:  6,
		}
		contract := &entities.SmartContract{ContractAddress: "So11111111111111111111111111111111111111112"}
		tx := uc.buildTransactionData(req, contract)
		require.NotNil(t, tx)
		require.Equal(t, contract.ContractAddress, tx.ContractAddress)
		require.Equal(t, contract.ContractAddress, tx.To)
		require.Equal(t, contract.ContractAddress, tx.ProgramID)
		require.NotEmpty(t, tx.Base58)
		require.Empty(t, tx.Hex)
	})
}

