package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
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

type routePolicyRepo struct {
	db *gorm.DB
}

func NewRoutePolicyRepository(db *gorm.DB) domainrepos.RoutePolicyRepository {
	return &routePolicyRepo{db: db}
}

func (r *routePolicyRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.RoutePolicy, error) {
	var row models.RoutePolicy
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return toRoutePolicyEntity(&row), nil
}

func (r *routePolicyRepo) GetByRoute(ctx context.Context, sourceChainID, destChainID uuid.UUID) (*entities.RoutePolicy, error) {
	var row models.RoutePolicy
	tx := r.db.WithContext(ctx).
		Where("source_chain_id = ? AND dest_chain_id = ?", sourceChainID, destChainID).
		Order("updated_at DESC").
		Limit(1).
		Find(&row)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, domainerrors.ErrNotFound
	}
	return toRoutePolicyEntity(&row), nil
}

func (r *routePolicyRepo) List(ctx context.Context, sourceChainID, destChainID *uuid.UUID, pagination utils.PaginationParams) ([]*entities.RoutePolicy, int64, error) {
	var rows []models.RoutePolicy
	var total int64

	query := r.db.WithContext(ctx).Model(&models.RoutePolicy{})
	if sourceChainID != nil {
		query = query.Where("source_chain_id = ?", *sourceChainID)
	}
	if destChainID != nil {
		query = query.Where("dest_chain_id = ?", *destChainID)
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

	items := make([]*entities.RoutePolicy, 0, len(rows))
	for i := range rows {
		items = append(items, toRoutePolicyEntity(&rows[i]))
	}
	return items, total, nil
}

func (r *routePolicyRepo) Create(ctx context.Context, policy *entities.RoutePolicy) error {
	if policy.ID == uuid.Nil {
		policy.ID = utils.GenerateUUIDv7()
	}

	fallbackOrder := marshalFallbackOrder(policy.FallbackOrder)
	mode := string(policy.FallbackMode)
	if mode == "" {
		mode = string(entities.BridgeFallbackModeStrict)
	}

	row := &models.RoutePolicy{
		ID:                policy.ID,
		SourceChainID:     policy.SourceChainID,
		DestChainID:       policy.DestChainID,
		DefaultBridgeType: int16(policy.DefaultBridgeType),
		FallbackMode:      mode,
		FallbackOrder:     fallbackOrder,
		PerByteRate:       nullableNumeric(policy.PerByteRate),
		OverheadBytes:     nullableNumeric(policy.OverheadBytes),
		MinFee:            nullableNumeric(policy.MinFee),
		MaxFee:            nullableNumeric(policy.MaxFee),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *routePolicyRepo) Update(ctx context.Context, policy *entities.RoutePolicy) error {
	fallbackOrder := marshalFallbackOrder(policy.FallbackOrder)
	mode := string(policy.FallbackMode)
	if mode == "" {
		mode = string(entities.BridgeFallbackModeStrict)
	}

	result := r.db.WithContext(ctx).Model(&models.RoutePolicy{}).
		Where("id = ?", policy.ID).
		Updates(map[string]interface{}{
			"source_chain_id":     policy.SourceChainID,
			"dest_chain_id":       policy.DestChainID,
			"default_bridge_type": int16(policy.DefaultBridgeType),
			"fallback_mode":       mode,
			"fallback_order":      fallbackOrder,
			"per_byte_rate":       nullableNumeric(policy.PerByteRate),
			"overhead_bytes":      nullableNumeric(policy.OverheadBytes),
			"min_fee":             nullableNumeric(policy.MinFee),
			"max_fee":             nullableNumeric(policy.MaxFee),
			"updated_at":          time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *routePolicyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.RoutePolicy{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func toRoutePolicyEntity(m *models.RoutePolicy) *entities.RoutePolicy {
	return &entities.RoutePolicy{
		ID:                m.ID,
		SourceChainID:     m.SourceChainID,
		DestChainID:       m.DestChainID,
		DefaultBridgeType: uint8(m.DefaultBridgeType),
		FallbackMode:      entities.BridgeFallbackMode(m.FallbackMode),
		FallbackOrder:     parseFallbackOrder(m.FallbackOrder),
		PerByteRate:       derefString(m.PerByteRate),
		OverheadBytes:     derefString(m.OverheadBytes),
		MinFee:            derefString(m.MinFee),
		MaxFee:            derefString(m.MaxFee),
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func nullableNumeric(v string) *string {
	s := strings.TrimSpace(v)
	if s == "" {
		return nil
	}
	return &s
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func marshalFallbackOrder(order []uint8) string {
	if len(order) == 0 {
		return "[0]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, item := range order {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(int(item)))
	}
	b.WriteByte(']')
	return b.String()
}

func parseFallbackOrder(raw string) []uint8 {
	if raw == "" {
		return []uint8{0}
	}
	var order []uint8
	if err := json.Unmarshal([]byte(raw), &order); err != nil || len(order) == 0 {
		return []uint8{0}
	}
	return order
}
