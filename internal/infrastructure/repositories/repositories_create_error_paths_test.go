package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestRepositoryCreate_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("bridge config create error", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewBridgeConfigRepository(db)
		err := repo.Create(ctx, &entities.BridgeConfig{
			ID:            uuid.New(),
			BridgeID:      uuid.New(),
			SourceChainID: uuid.New(),
			DestChainID:   uuid.New(),
			RouterAddress: "0x1",
			FeePercentage: "0.1",
			Config:        "{}",
			IsActive:      true,
		})
		require.Error(t, err)
	})

	t.Run("chain create error", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewChainRepository(db)
		err := repo.Create(ctx, &entities.Chain{
			ID:             uuid.New(),
			ChainID:        "8453",
			Name:           "Base",
			Type:           entities.ChainTypeEVM,
			RPCURL:         "https://rpc.example",
			ExplorerURL:    "https://scan.example",
			CurrencySymbol: "ETH",
			IsActive:       true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		})
		require.Error(t, err)
	})

	t.Run("fee config create error", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewFeeConfigRepository(db)
		err := repo.Create(ctx, &entities.FeeConfig{
			ID:                 uuid.New(),
			ChainID:            uuid.New(),
			TokenID:            uuid.New(),
			PlatformFeePercent: "0.01",
			FixedBaseFee:       "1",
			MinFee:             "0.1",
		})
		require.Error(t, err)
	})

	t.Run("payment bridge create error", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewPaymentBridgeRepository(db)
		err := repo.Create(ctx, &entities.PaymentBridge{
			ID:   uuid.New(),
			Name: "Hyperbridge",
		})
		require.Error(t, err)
	})

	t.Run("payment event create error", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewPaymentEventRepository(db)
		err := repo.Create(ctx, &entities.PaymentEvent{
			ID:        uuid.New(),
			PaymentID: uuid.New(),
			EventType: entities.PaymentEventTypeCreated,
			Metadata:  "{}",
			CreatedAt: time.Now(),
		})
		require.Error(t, err)
	})

	t.Run("user create error", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewUserRepository(db)
		err := repo.Create(ctx, &entities.User{
			ID:           uuid.New(),
			Email:        "a@b.c",
			Name:         "A",
			PasswordHash: "x",
			Role:         entities.UserRoleUser,
			KYCStatus:    entities.KYCNotStarted,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		})
		require.Error(t, err)
	})
}
