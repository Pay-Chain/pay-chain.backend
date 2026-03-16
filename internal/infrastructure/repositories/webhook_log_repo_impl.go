package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"gorm.io/gorm"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/infrastructure/models"
)

type GormWebhookLogRepository struct {
	db *gorm.DB
}

func NewGormWebhookLogRepository(db *gorm.DB) *GormWebhookLogRepository {
	return &GormWebhookLogRepository{db: db}
}

func (r *GormWebhookLogRepository) toEntity(m *models.WebhookLog) *entities.WebhookDelivery {
	return &entities.WebhookDelivery{
		ID:             m.ID,
		MerchantID:     m.MerchantID,
		PaymentID:      m.PaymentID,
		EventType:      m.EventType,
		Payload:        null.JSONFrom([]byte(m.Payload)),
		DeliveryStatus: entities.WebhookDeliveryStatus(m.DeliveryStatus),
		HttpStatus:     m.HttpStatus,
		ResponseBody:   m.ResponseBody,
		RetryCount:     m.RetryCount,
		NextRetryAt:    m.NextRetryAt,
		LastAttemptAt:  m.LastAttemptAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func (r *GormWebhookLogRepository) toModel(e *entities.WebhookDelivery) *models.WebhookLog {
	payloadStr := "{}"
	if e.Payload.Valid {
		payloadStr = string(e.Payload.JSON)
	}

	return &models.WebhookLog{
		ID:             e.ID,
		MerchantID:     e.MerchantID,
		PaymentID:      e.PaymentID,
		EventType:      e.EventType,
		Payload:        payloadStr,
		DeliveryStatus: string(e.DeliveryStatus),
		HttpStatus:     e.HttpStatus,
		ResponseBody:   e.ResponseBody,
		RetryCount:     e.RetryCount,
		NextRetryAt:    e.NextRetryAt,
		LastAttemptAt:  e.LastAttemptAt,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func (r *GormWebhookLogRepository) Create(ctx context.Context, log *entities.WebhookDelivery) error {
	m := r.toModel(log)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	log.ID = m.ID
	return nil
}

func (r *GormWebhookLogRepository) Update(ctx context.Context, log *entities.WebhookDelivery) error {
	m := r.toModel(log)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *GormWebhookLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.WebhookDelivery, error) {
	var m models.WebhookLog
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return r.toEntity(&m), nil
}

func (r *GormWebhookLogRepository) GetPendingAttempts(ctx context.Context, limit int) ([]entities.WebhookDelivery, error) {
	var ms []models.WebhookLog
	err := r.db.WithContext(ctx).
		Where("delivery_status IN ?", []string{"pending", "retrying"}).
		Where("next_retry_at IS NULL OR next_retry_at < ?", time.Now()).
		Order("created_at ASC").
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}

	entities := make([]entities.WebhookDelivery, len(ms))
	for i, m := range ms {
		entities[i] = *r.toEntity(&m)
	}
	return entities, nil
}

func (r *GormWebhookLogRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, httpCode int, body string) error {
	updates := map[string]interface{}{
		"delivery_status": status,
		"http_status":     httpCode,
		"response_body":   body,
		"last_attempt_at": time.Now(),
		"updated_at":      time.Now(),
	}
	return r.db.WithContext(ctx).Model(&models.WebhookLog{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GormWebhookLogRepository) GetMerchantHistory(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]entities.WebhookDelivery, int64, error) {
	var ms []models.WebhookLog
	var total int64

	base := r.db.WithContext(ctx).Model(&models.WebhookLog{}).Where("merchant_id = ?", merchantID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&ms).Error
	if err != nil {
		return nil, 0, err
	}

	entities := make([]entities.WebhookDelivery, len(ms))
	for i, m := range ms {
		entities[i] = *r.toEntity(&m)
	}

	return entities, total, nil
}
