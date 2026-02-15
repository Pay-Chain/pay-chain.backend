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

type feeConfigRepo struct {
	db *gorm.DB
}

func NewFeeConfigRepository(db *gorm.DB) domainrepos.FeeConfigRepository {
	return &feeConfigRepo{db: db}
}

func (r *feeConfigRepo) GetByChainAndToken(ctx context.Context, chainID, tokenID uuid.UUID) (*entities.FeeConfig, error) {
	var m models.FeeConfig
	err := r.db.WithContext(ctx).
		Where("chain_id = ? AND token_id = ?", chainID, tokenID).
		Order("updated_at DESC").
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}

	return toFeeConfigEntity(&m), nil
}

func (r *feeConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.FeeConfig, error) {
	var m models.FeeConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return toFeeConfigEntity(&m), nil
}

func (r *feeConfigRepo) List(ctx context.Context, chainID, tokenID *uuid.UUID, pagination utils.PaginationParams) ([]*entities.FeeConfig, int64, error) {
	var rows []models.FeeConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&models.FeeConfig{})
	if chainID != nil {
		query = query.Where("chain_id = ?", *chainID)
	}
	if tokenID != nil {
		query = query.Where("token_id = ?", *tokenID)
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

	items := make([]*entities.FeeConfig, 0, len(rows))
	for _, row := range rows {
		items = append(items, toFeeConfigEntity(&row))
	}
	return items, total, nil
}

func (r *feeConfigRepo) Create(ctx context.Context, config *entities.FeeConfig) error {
	if config.ID == uuid.Nil {
		config.ID = utils.GenerateUUIDv7()
	}

	m := &models.FeeConfig{
		ID:                 config.ID,
		ChainID:            config.ChainID,
		TokenID:            config.TokenID,
		PlatformFeePercent: config.PlatformFeePercent,
		FixedBaseFee:       config.FixedBaseFee,
		MinFee:             config.MinFee,
		MaxFee:             config.MaxFee,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *feeConfigRepo) Update(ctx context.Context, config *entities.FeeConfig) error {
	result := r.db.WithContext(ctx).Model(&models.FeeConfig{}).
		Where("id = ?", config.ID).
		Updates(map[string]interface{}{
			"chain_id":             config.ChainID,
			"token_id":             config.TokenID,
			"platform_fee_percent": config.PlatformFeePercent,
			"fixed_base_fee":       config.FixedBaseFee,
			"min_fee":              config.MinFee,
			"max_fee":              config.MaxFee,
			"updated_at":           time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *feeConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.FeeConfig{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func toFeeConfigEntity(m *models.FeeConfig) *entities.FeeConfig {
	return &entities.FeeConfig{
		ID:                 m.ID,
		ChainID:            m.ChainID,
		TokenID:            m.TokenID,
		PlatformFeePercent: m.PlatformFeePercent,
		FixedBaseFee:       m.FixedBaseFee,
		MinFee:             m.MinFee,
		MaxFee:             m.MaxFee,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}
