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
	}, nil
}
