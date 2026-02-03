package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/models"
)

// PaymentRequestRepositoryImpl implements PaymentRequestRepository
type PaymentRequestRepositoryImpl struct {
	db *gorm.DB
}

func NewPaymentRequestRepository(db *gorm.DB) *PaymentRequestRepositoryImpl {
	return &PaymentRequestRepositoryImpl{db: db}
}

func (r *PaymentRequestRepositoryImpl) Create(ctx context.Context, req *entities.PaymentRequest) error {
	m := &models.PaymentRequest{
		ID:           req.ID,
		MerchantID:   req.MerchantID,
		WalletID:     req.WalletID,
		ChainID:      req.ChainID,
		TokenAddress: req.TokenAddress,
		Amount:       req.Amount,
		Decimals:     req.Decimals,
		Description:  req.Description,
		Status:       string(req.Status),
		ExpiresAt:    req.ExpiresAt,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *PaymentRequestRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*entities.PaymentRequest, error) {
	var m models.PaymentRequest
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return r.toEntity(&m), nil
}

func (r *PaymentRequestRepositoryImpl) GetByMerchantID(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.PaymentRequest{}).
		Where("merchant_id = ?", merchantID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ms []models.PaymentRequest
	if err := r.db.WithContext(ctx).
		Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var requests []*entities.PaymentRequest
	for _, m := range ms {
		model := m
		requests = append(requests, r.toEntity(&model))
	}
	return requests, int(total), nil
}

func (r *PaymentRequestRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PaymentRequestStatus) error {
	return r.db.WithContext(ctx).Model(&models.PaymentRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (r *PaymentRequestRepositoryImpl) UpdateTxHash(ctx context.Context, id uuid.UUID, txHash, payerAddress string) error {
	return r.db.WithContext(ctx).Model(&models.PaymentRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"tx_hash":       txHash,
			"payer_address": payerAddress,
			"updated_at":    time.Now(),
		}).Error
}

func (r *PaymentRequestRepositoryImpl) MarkCompleted(ctx context.Context, id uuid.UUID, txHash string) error {
	now := time.Now()
	// entities.PaymentRequestStatusCompleted? import entities
	return r.db.WithContext(ctx).Model(&models.PaymentRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "completed",
			"tx_hash":      txHash,
			"completed_at": now,
			"updated_at":   now,
		}).Error
}

func (r *PaymentRequestRepositoryImpl) GetExpiredPending(ctx context.Context, limit int) ([]*entities.PaymentRequest, error) {
	var ms []models.PaymentRequest
	if err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at < ?", "pending", time.Now()).
		Limit(limit).
		Find(&ms).Error; err != nil {
		return nil, err
	}

	var requests []*entities.PaymentRequest
	for _, m := range ms {
		model := m
		requests = append(requests, r.toEntity(&model))
	}
	return requests, nil
}

func (r *PaymentRequestRepositoryImpl) ExpireRequests(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&models.PaymentRequest{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     "expired",
			"updated_at": time.Now(),
		}).Error
}

func (r *PaymentRequestRepositoryImpl) toEntity(m *models.PaymentRequest) *entities.PaymentRequest {
	return &entities.PaymentRequest{
		ID:           m.ID,
		MerchantID:   m.MerchantID,
		WalletID:     m.WalletID,
		ChainID:      m.ChainID,
		TokenAddress: m.TokenAddress,
		Amount:       m.Amount,
		Decimals:     m.Decimals,
		Description:  m.Description,
		Status:       entities.PaymentRequestStatus(m.Status),
		ExpiresAt:    m.ExpiresAt,
		TxHash:       m.TxHash,
		PayerAddress: m.PayerAddress,
		CompletedAt:  m.CompletedAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// RpcEndpointRepositoryImpl implements RpcEndpointRepository
type RpcEndpointRepositoryImpl struct {
	db *gorm.DB
}

func NewRpcEndpointRepository(db *gorm.DB) *RpcEndpointRepositoryImpl {
	return &RpcEndpointRepositoryImpl{db: db}
}

func (r *RpcEndpointRepositoryImpl) GetByChainID(ctx context.Context, chainID int) ([]*entities.RpcEndpoint, error) {
	var ms []models.RpcEndpoint
	if err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Order("priority ASC").
		Find(&ms).Error; err != nil {
		return nil, err
	}
	return r.toEntities(ms), nil
}

func (r *RpcEndpointRepositoryImpl) GetActiveByChainID(ctx context.Context, chainID int) ([]*entities.RpcEndpoint, error) {
	var ms []models.RpcEndpoint
	if err := r.db.WithContext(ctx).
		Where("chain_id = ? AND is_active = ?", chainID, true).
		Order("priority ASC, error_count ASC").
		Find(&ms).Error; err != nil {
		return nil, err
	}
	return r.toEntities(ms), nil
}

func (r *RpcEndpointRepositoryImpl) MarkError(ctx context.Context, id uuid.UUID) error {
	// gorm usage: UPDATE rpc_endpoints SET error_count = error_count + 1, ...
	return r.db.WithContext(ctx).Model(&models.RpcEndpoint{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"error_count":   gorm.Expr("error_count + ?", 1),
			"last_error_at": time.Now(),
			"updated_at":    time.Now(),
		}).Error
}

func (r *RpcEndpointRepositoryImpl) ResetErrors(ctx context.Context, chainID int) error {
	return r.db.WithContext(ctx).Model(&models.RpcEndpoint{}).
		Where("chain_id = ?", chainID).
		Updates(map[string]interface{}{
			"error_count": 0,
			"updated_at":  time.Now(),
		}).Error
}

func (r *RpcEndpointRepositoryImpl) toEntities(ms []models.RpcEndpoint) []*entities.RpcEndpoint {
	var eps []*entities.RpcEndpoint
	for _, m := range ms {
		eps = append(eps, &entities.RpcEndpoint{
			ID:          m.ID,
			ChainID:     m.ChainID,
			URL:         m.URL,
			Priority:    m.Priority,
			IsActive:    m.IsActive,
			LastErrorAt: m.LastErrorAt,
			ErrorCount:  m.ErrorCount,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}
	return eps
}

// BackgroundJobRepositoryImpl implements BackgroundJobRepository
type BackgroundJobRepositoryImpl struct {
	db *gorm.DB
}

func NewBackgroundJobRepository(db *gorm.DB) *BackgroundJobRepositoryImpl {
	return &BackgroundJobRepositoryImpl{db: db}
}

func (r *BackgroundJobRepositoryImpl) Create(ctx context.Context, job *entities.BackgroundJob) error {
	m := &models.BackgroundJob{
		ID:          job.ID,
		JobType:     job.JobType,
		Payload:     job.Payload,
		Status:      job.Status,
		MaxAttempts: job.MaxAttempts,
		ScheduledAt: job.ScheduledAt,
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *BackgroundJobRepositoryImpl) GetPending(ctx context.Context, limit int) ([]*entities.BackgroundJob, error) {
	var ms []models.BackgroundJob
	if err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at <= ? AND attempts < max_attempts", "pending", time.Now()).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&ms).Error; err != nil {
		return nil, err
	}

	var jobs []*entities.BackgroundJob
	for _, m := range ms {
		jobs = append(jobs, &entities.BackgroundJob{
			ID:           m.ID,
			JobType:      m.JobType,
			Payload:      m.Payload,
			Status:       m.Status,
			Attempts:     m.Attempts,
			MaxAttempts:  m.MaxAttempts,
			ScheduledAt:  m.ScheduledAt,
			StartedAt:    m.StartedAt,
			CompletedAt:  m.CompletedAt,
			ErrorMessage: m.ErrorMessage,
			CreatedAt:    m.CreatedAt,
			UpdatedAt:    m.UpdatedAt,
		})
	}
	return jobs, nil
}

func (r *BackgroundJobRepositoryImpl) MarkProcessing(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.BackgroundJob{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     "processing",
			"started_at": time.Now(),
			"attempts":   gorm.Expr("attempts + ?", 1),
			"updated_at": time.Now(),
		}).Error
}

func (r *BackgroundJobRepositoryImpl) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.BackgroundJob{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": now,
			"updated_at":   now,
		}).Error
}

func (r *BackgroundJobRepositoryImpl) MarkFailed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	return r.db.WithContext(ctx).Model(&models.BackgroundJob{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": errorMsg,
			"updated_at":    time.Now(),
		}).Error
}
