package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	domainrepos "pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/models"
)

type bridgeConfigRepo struct {
	db *gorm.DB
}

func NewBridgeConfigRepository(db *gorm.DB) domainrepos.BridgeConfigRepository {
	return &bridgeConfigRepo{db: db}
}

func (r *bridgeConfigRepo) GetActive(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.BridgeConfig, error) {
	var m models.BridgeConfig
	err := r.db.WithContext(ctx).
		Preload("Bridge").
		Where("source_chain_id = ? AND dest_chain_id = ? AND is_active = ?", sourceChainID, destChainID, true).
		Order("updated_at DESC").
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}

	var bridge *entities.PaymentBridge
	if m.Bridge.ID != uuid.Nil {
		bridge = &entities.PaymentBridge{
			ID:   m.Bridge.ID,
			Name: m.Bridge.Name,
		}
	}

	return &entities.BridgeConfig{
		ID:            m.ID,
		BridgeID:      m.BridgeID,
		SourceChainID: m.SourceChainID,
		DestChainID:   m.DestChainID,
		RouterAddress: m.RouterAddress,
		FeePercentage: m.FeePercentage,
		Config:        m.Config,
		IsActive:      m.IsActive,
		Bridge:        bridge,
	}, nil
}
