package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	domainrepos "payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/infrastructure/models"
	"payment-kita.backend/pkg/utils"
)

type stargateConfigRepo struct {
	db *gorm.DB
}

func NewStargateConfigRepository(db *gorm.DB) domainrepos.StargateConfigRepository {
	return &stargateConfigRepo{db: db}
}

func (r *stargateConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.StargateConfig, error) {
	var row models.StargateConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return toStargateConfigEntity(&row), nil
}

func (r *stargateConfigRepo) GetByRoute(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.StargateConfig, error) {
	var row models.StargateConfig
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
	return toStargateConfigEntity(&row), nil
}

func (r *stargateConfigRepo) List(ctx context.Context, sourceChainID, destChainID *uuid.UUID, activeOnly *bool, pagination utils.PaginationParams) ([]*entities.StargateConfig, int64, error) {
	var rows []models.StargateConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&models.StargateConfig{})
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

	items := make([]*entities.StargateConfig, 0, len(rows))
	for i := range rows {
		items = append(items, toStargateConfigEntity(&rows[i]))
	}
	return items, total, nil
}

func (r *stargateConfigRepo) Create(ctx context.Context, config *entities.StargateConfig) error {
	if config.ID == uuid.Nil {
		config.ID = utils.GenerateUUIDv7()
	}

	optionsHex := strings.TrimSpace(config.OptionsHex)
	if optionsHex == "" {
		optionsHex = "0x"
	}

	row := &models.StargateConfig{
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

func (r *stargateConfigRepo) Update(ctx context.Context, config *entities.StargateConfig) error {
	optionsHex := strings.TrimSpace(config.OptionsHex)
	if optionsHex == "" {
		optionsHex = "0x"
	}

	result := r.db.WithContext(ctx).Model(&models.StargateConfig{}).
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

func (r *stargateConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.StargateConfig{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func toStargateConfigEntity(m *models.StargateConfig) *entities.StargateConfig {
	return &entities.StargateConfig{
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
