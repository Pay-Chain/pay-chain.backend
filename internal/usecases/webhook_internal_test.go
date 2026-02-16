package usecases

import (
	"testing"

	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestMapStatus_AllBranches(t *testing.T) {
	tests := []struct {
		in   string
		want entities.PaymentStatus
	}{
		{in: "pending", want: entities.PaymentStatusPending},
		{in: "processing", want: entities.PaymentStatusProcessing},
		{in: "completed", want: entities.PaymentStatusCompleted},
		{in: "failed", want: entities.PaymentStatusFailed},
		{in: "refunded", want: entities.PaymentStatusRefunded},
		{in: "unexpected", want: entities.PaymentStatusPending},
	}

	for _, tt := range tests {
		require.Equal(t, tt.want, mapStatus(tt.in))
	}
}
