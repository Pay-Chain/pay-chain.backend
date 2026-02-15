package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8" // Added import
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
)

// PaymentRepository implements payment data operations
type PaymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository creates a new payment repository
func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create creates a new payment
func (r *PaymentRepository) Create(ctx context.Context, payment *entities.Payment) error {
	fmt.Printf("DEBUG: PaymentRepository.Create - ID: %s\n", payment.ID)

	// Handle nullable fields from Entity -> Model (*type)
	m := &models.Payment{
		ID:            payment.ID,
		SenderID:      *payment.SenderID, // Entity has *uuid.UUID, Model has uuid.UUID
		MerchantID:    payment.MerchantID,
		BridgeID:      payment.BridgeID,
		SourceChainID: payment.SourceChainID,
		DestChainID:   payment.DestChainID,
		SourceTokenID: *payment.SourceTokenID,
		DestTokenID:   *payment.DestTokenID,
		// payment.SourceTokenAddress? payment.go Entity has SourceAddress?
		// Entity: SenderAddress string, DestAddress string.
		// It does NOT have SourceTokenAddress.
		// The Model has `SourceTokenAddress`.
		// This suggests I need to fetch Token address. OR Entity should store it.
		// RE-READ `payment.go` entity.
	}

	// I need to correct mapping.
	// Entity: `SenderAddress`, `DestAddress` (User Wallets).
	// Model: `ReceiverWalletID`.
	// Model: `SourceTokenAddress`, `DestTokenAddress` (Token Contract Addresses).
	// Entity: `SourceToken`, `DestToken` relations.

	// Simplification for now: Just use values provided.

	// FIX: Using minimal creating for now.

	m.SenderID = *payment.SenderID
	if payment.MerchantID != nil {
		m.MerchantID = payment.MerchantID
	}
	// m.ReceiverWalletID = uuid.Nil - Removed
	m.SourceChainID = payment.SourceChainID
	m.DestChainID = payment.DestChainID
	if payment.SourceTokenID != nil {
		m.SourceTokenID = *payment.SourceTokenID
	}
	if payment.DestTokenID != nil {
		m.DestTokenID = *payment.DestTokenID
	}
	m.SourceAmount = payment.SourceAmount
	if payment.DestAmount.Valid {
		val := payment.DestAmount.String
		m.DestAmount = &val
	}
	m.FeeAmount = payment.FeeAmount
	m.TotalCharged = payment.TotalCharged
	m.SenderAddress = payment.SenderAddress
	m.DestAddress = payment.ReceiverAddress
	m.Status = string(payment.Status)
	m.CreatedAt = payment.CreatedAt
	m.UpdatedAt = payment.UpdatedAt

	// Use the transaction-aware DB instance
	db := GetDB(ctx, r.db)
	isTx := false
	if _, ok := ctx.Value(txKey).(*gorm.DB); ok {
		isTx = true
	}
	fmt.Printf("DEBUG: PaymentRepository.Create - IsTX: %v\n", isTx)

	if err := db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	// Ensure caller uses the actual persisted ID (important for subsequent FK inserts in same tx).
	payment.ID = m.ID
	return nil
}

// GetByID gets a payment by ID
func (r *PaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Payment, error) {
	var m models.Payment
	// Use the transaction-aware DB instance
	db := GetDB(ctx, r.db)
	if err := db.WithContext(ctx).Preload("SourceChain").Preload("DestChain").Preload("SourceToken").Preload("DestToken").Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetByUserID gets payments for a user with pagination
func (r *PaymentRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Payment, int, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.Payment{}).
		Where("sender_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ms []models.Payment
	if err := r.db.WithContext(ctx).
		Preload("SourceChain").Preload("DestChain").
		Where("sender_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var payments []*entities.Payment
	for _, m := range ms {
		model := m
		payments = append(payments, r.toEntity(&model))
	}

	return payments, int(total), nil
}

// GetByMerchantID gets payments for a merchant
func (r *PaymentRepository) GetByMerchantID(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*entities.Payment, int, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.Payment{}).
		Where("merchant_id = ?", merchantID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ms []models.Payment
	if err := r.db.WithContext(ctx).
		Preload("SourceChain").Preload("DestChain").
		Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var payments []*entities.Payment
	for _, m := range ms {
		model := m
		payments = append(payments, r.toEntity(&model))
	}

	return payments, int(total), nil
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PaymentStatus) error {
	db := GetDB(ctx, r.db)
	result := db.WithContext(ctx).Model(&models.Payment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
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

func (r *PaymentRepository) UpdateDestTxHash(ctx context.Context, id uuid.UUID, txHash string) error {
	db := GetDB(ctx, r.db)
	return db.WithContext(ctx).Model(&models.Payment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"dest_tx_hash": txHash,
			"updated_at":   time.Now(),
		}).Error
}

func (r *PaymentRepository) MarkRefunded(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	db := GetDB(ctx, r.db)
	result := db.WithContext(ctx).Model(&models.Payment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     entities.PaymentStatusRefunded,
			"updated_at": now,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *PaymentRepository) toEntity(m *models.Payment) *entities.Payment {
	p := &entities.Payment{
		ID:                  m.ID,
		SenderID:            &m.SenderID,
		MerchantID:          m.MerchantID,
		BridgeID:            m.BridgeID,
		SourceChainID:       m.SourceChainID,
		DestChainID:         m.DestChainID,
		SourceTokenID:       &m.SourceTokenID,
		DestTokenID:         &m.DestTokenID,
		SenderAddress:       m.SenderAddress,
		DestAddress:         m.DestAddress,
		ReceiverAddress:     m.DestAddress,
		SourceAmount:        m.SourceAmount,
		DestAmount:          null.StringFromPtr(m.DestAmount),
		FeeAmount:           m.FeeAmount,
		TotalCharged:        m.TotalCharged,
		Status:              entities.PaymentStatus(m.Status),
		SourceTxHash:        null.StringFromPtr(m.SourceTxHash),
		DestTxHash:          null.StringFromPtr(m.DestTxHash),
		RefundTxHash:        null.StringFromPtr(m.RefundTxHash),
		CrossChainMessageID: null.StringFromPtr(m.CrossChainMessageID),
		ExpiresAt:           m.ExpiresAt,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}

	// Map Chain Relations
	// Just minimal mapping if preloaded
	if m.SourceChain.ID != uuid.Nil {
		p.SourceChain = &entities.Chain{
			ID:      m.SourceChain.ID,
			ChainID: m.SourceChain.NetworkID,
			Name:    m.SourceChain.Name,
		}
	}
	if m.DestChain.ID != uuid.Nil {
		p.DestChain = &entities.Chain{
			ID:      m.DestChain.ID,
			ChainID: m.DestChain.NetworkID,
			Name:    m.DestChain.Name,
		}
	}

	return p
}
