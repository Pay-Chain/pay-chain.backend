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

type paymentBridgeRepo struct {
	db *gorm.DB
}

func NewPaymentBridgeRepository(db *gorm.DB) domainrepos.PaymentBridgeRepository {
	return &paymentBridgeRepo{db: db}
}

func (r *paymentBridgeRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.PaymentBridge, error) {
	var m models.PaymentBridge
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &entities.PaymentBridge{ID: m.ID, Name: m.Name}, nil
}

func (r *paymentBridgeRepo) GetByName(ctx context.Context, name string) (*entities.PaymentBridge, error) {
	var m models.PaymentBridge
	if err := r.db.WithContext(ctx).Where("LOWER(name) = ?", strings.ToLower(strings.TrimSpace(name))).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &entities.PaymentBridge{ID: m.ID, Name: m.Name}, nil
}

func (r *paymentBridgeRepo) List(ctx context.Context, pagination utils.PaginationParams) ([]*entities.PaymentBridge, int64, error) {
	var rows []models.PaymentBridge
	var total int64

	query := r.db.WithContext(ctx).Model(&models.PaymentBridge{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}

	if err := query.Order("name ASC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*entities.PaymentBridge, 0, len(rows))
	for _, row := range rows {
		items = append(items, &entities.PaymentBridge{
			ID:   row.ID,
			Name: row.Name,
		})
	}
	return items, total, nil
}

func (r *paymentBridgeRepo) Create(ctx context.Context, bridge *entities.PaymentBridge) error {
	if bridge.ID == uuid.Nil {
		bridge.ID = utils.GenerateUUIDv7()
	}

	m := &models.PaymentBridge{
		ID:        bridge.ID,
		Name:      strings.TrimSpace(bridge.Name),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *paymentBridgeRepo) Update(ctx context.Context, bridge *entities.PaymentBridge) error {
	result := r.db.WithContext(ctx).Model(&models.PaymentBridge{}).
		Where("id = ?", bridge.ID).
		Updates(map[string]interface{}{
			"name":       strings.TrimSpace(bridge.Name),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *paymentBridgeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.PaymentBridge{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}
