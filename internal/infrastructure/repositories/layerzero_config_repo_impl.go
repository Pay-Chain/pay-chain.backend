package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	domainrepos "pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/models"
	"pay-chain.backend/pkg/utils"
)

type layerZeroConfigRepo struct {
	db *gorm.DB
}

func NewLayerZeroConfigRepository(db *gorm.DB) domainrepos.LayerZeroConfigRepository {
	return &layerZeroConfigRepo{db: db}
}

func (r *layerZeroConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.LayerZeroConfig, error) {
	var row models.LayerZeroConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return toLayerZeroConfigEntity(&row), nil
}

func (r *layerZeroConfigRepo) GetByRoute(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.LayerZeroConfig, error) {
	var row models.LayerZeroConfig
	err := r.db.WithContext(ctx).
		Where("source_chain_id = ? AND dest_chain_id = ?", sourceChainID, destChainID).
		Order("updated_at DESC").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return toLayerZeroConfigEntity(&row), nil
}

func (r *layerZeroConfigRepo) List(ctx context.Context, sourceChainID, destChainID *uuid.UUID, activeOnly *bool, pagination utils.PaginationParams) ([]*entities.LayerZeroConfig, int64, error) {
	var rows []models.LayerZeroConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&models.LayerZeroConfig{})
	if sourceChainID != nil {
		query = query.Where("source_chain_id = ?", *sourceChainID)
	}
	if destChainID != nil {
		query = query.Where("dest_chain_id = ?", *destChainID)
	}
	if activeOnly != nil {
		query = query.Where("is_active = ?", *activeOnly)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*entities.LayerZeroConfig, 0, len(rows))
	for i := range rows {
		items = append(items, toLayerZeroConfigEntity(&rows[i]))
	}
	return items, total, nil
}

func (r *layerZeroConfigRepo) Create(ctx context.Context, config *entities.LayerZeroConfig) error {
	if config.ID == uuid.Nil {
		config.ID = utils.GenerateUUIDv7()
	}

	optionsHex := strings.TrimSpace(config.OptionsHex)
	if optionsHex == "" {
		optionsHex = "0x"
	}

	row := &models.LayerZeroConfig{
		ID:            config.ID,
		SourceChainID: config.SourceChainID,
		DestChainID:   config.DestChainID,
		DstEID:        config.DstEID,
		PeerHex:       strings.TrimSpace(config.PeerHex),
		OptionsHex:    optionsHex,
		IsActive:      config.IsActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *layerZeroConfigRepo) Update(ctx context.Context, config *entities.LayerZeroConfig) error {
	optionsHex := strings.TrimSpace(config.OptionsHex)
	if optionsHex == "" {
		optionsHex = "0x"
	}

	result := r.db.WithContext(ctx).Model(&models.LayerZeroConfig{}).
		Where("id = ?", config.ID).
		Updates(map[string]interface{}{
			"source_chain_id": config.SourceChainID,
			"dest_chain_id":   config.DestChainID,
			"dst_eid":         config.DstEID,
			"peer_hex":        strings.TrimSpace(config.PeerHex),
			"options_hex":     optionsHex,
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

func (r *layerZeroConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.LayerZeroConfig{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func toLayerZeroConfigEntity(m *models.LayerZeroConfig) *entities.LayerZeroConfig {
	return &entities.LayerZeroConfig{
		ID:            m.ID,
		SourceChainID: m.SourceChainID,
		DestChainID:   m.DestChainID,
		DstEID:        m.DstEID,
		PeerHex:       m.PeerHex,
		OptionsHex:    m.OptionsHex,
		IsActive:      m.IsActive,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
