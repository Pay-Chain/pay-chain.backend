package usecases

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
)

func TestPaymentRequestUsecase_BuildSvmTransactionBase58(t *testing.T) {
	uc := &PaymentRequestUsecase{}
	req := &entities.PaymentRequest{
		ID:        uuid.New(),
		NetworkID: "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp",
		Amount:    "1000",
		Decimals:  6,
	}

	encoded := uc.buildSvmTransactionBase58(req)
	assert.NotEmpty(t, encoded)
}
