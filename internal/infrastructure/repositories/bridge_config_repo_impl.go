package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	domainrepos "pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/models"
	"pay-chain.backend/pkg/utils"
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

	return toBridgeConfigEntity(&m, bridge), nil
}

func (r *bridgeConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BridgeConfig, error) {
	var m models.BridgeConfig
	err := r.db.WithContext(ctx).Preload("Bridge").Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}

	var bridge *entities.PaymentBridge
	if m.Bridge.ID != uuid.Nil {
		bridge = &entities.PaymentBridge{ID: m.Bridge.ID, Name: m.Bridge.Name}
	}
	return toBridgeConfigEntity(&m, bridge), nil
}

func (r *bridgeConfigRepo) List(ctx context.Context, sourceChainID, destChainID, bridgeID *uuid.UUID, pagination utils.PaginationParams) ([]*entities.BridgeConfig, int64, error) {
	var rows []models.BridgeConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&models.BridgeConfig{})
	if sourceChainID != nil {
		query = query.Where("source_chain_id = ?", *sourceChainID)
	}
	if destChainID != nil {
		query = query.Where("dest_chain_id = ?", *destChainID)
	}
	if bridgeID != nil {
		query = query.Where("bridge_id = ?", *bridgeID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Preload("Bridge")
	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*entities.BridgeConfig, 0, len(rows))
	for _, row := range rows {
		var bridge *entities.PaymentBridge
		if row.Bridge.ID != uuid.Nil {
			bridge = &entities.PaymentBridge{ID: row.Bridge.ID, Name: row.Bridge.Name}
		}
		items = append(items, toBridgeConfigEntity(&row, bridge))
	}
	return items, total, nil
}

func (r *bridgeConfigRepo) Create(ctx context.Context, config *entities.BridgeConfig) error {
	if config.ID == uuid.Nil {
		config.ID = utils.GenerateUUIDv7()
	}
	m := &models.BridgeConfig{
		ID:            config.ID,
		BridgeID:      config.BridgeID,
		SourceChainID: config.SourceChainID,
		DestChainID:   config.DestChainID,
		RouterAddress: config.RouterAddress,
		FeePercentage: config.FeePercentage,
		Config:        config.Config,
		IsActive:      config.IsActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *bridgeConfigRepo) Update(ctx context.Context, config *entities.BridgeConfig) error {
	result := r.db.WithContext(ctx).Model(&models.BridgeConfig{}).
		Where("id = ?", config.ID).
		Updates(map[string]interface{}{
			"bridge_id":       config.BridgeID,
			"source_chain_id": config.SourceChainID,
			"dest_chain_id":   config.DestChainID,
			"router_address":  config.RouterAddress,
			"fee_percentage":  config.FeePercentage,
			"config":          config.Config,
			"is_active":       config.IsActive,
			"updated_at":      time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *bridgeConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.BridgeConfig{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func toBridgeConfigEntity(m *models.BridgeConfig, bridge *entities.PaymentBridge) *entities.BridgeConfig {
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
	}
}
