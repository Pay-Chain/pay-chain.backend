package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	domainentities "payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/infrastructure/models"
)

type PaymentQuoteRepositoryImpl struct {
	db *gorm.DB
}

func NewPaymentQuoteRepository(db *gorm.DB) *PaymentQuoteRepositoryImpl {
	return &PaymentQuoteRepositoryImpl{db: db}
}

func (r *PaymentQuoteRepositoryImpl) Create(ctx context.Context, quote *domainentities.PaymentQuote) error {
	m := &models.PaymentQuote{
		ID:                    quote.ID,
		MerchantID:            quote.MerchantID,
		InvoiceCurrency:       quote.InvoiceCurrency,
		InvoiceAmount:         quote.InvoiceAmount,
		SelectedChainID:       quote.SelectedChainID,
		SelectedTokenAddress:  quote.SelectedTokenAddress,
		SelectedTokenSymbol:   quote.SelectedTokenSymbol,
		SelectedTokenDecimals: quote.SelectedTokenDecimals,
		QuotedAmount:          quote.QuotedAmount,
		QuoteRate:             quote.QuoteRate,
		PriceSource:           quote.PriceSource,
		Route:                 quote.Route,
		SlippageBps:           quote.SlippageBps,
		RateTimestamp:         quote.RateTimestamp,
		ExpiresAt:             quote.ExpiresAt,
		Status:                string(quote.Status),
		UsedAt:                quote.UsedAt,
		CreatedAt:             quote.CreatedAt,
		UpdatedAt:             quote.UpdatedAt,
	}
	return GetDB(ctx, r.db).Create(m).Error
}

func (r *PaymentQuoteRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domainentities.PaymentQuote, error) {
	var m models.PaymentQuote
	if err := GetDB(ctx, r.db).Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

func (r *PaymentQuoteRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status domainentities.PaymentQuoteStatus) error {
	res := GetDB(ctx, r.db).Model(&models.PaymentQuote{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     string(status),
		"updated_at": time.Now(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *PaymentQuoteRepositoryImpl) MarkUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	res := GetDB(ctx, r.db).Model(&models.PaymentQuote{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     string(domainentities.PaymentQuoteStatusUsed),
		"used_at":    now,
		"updated_at": now,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *PaymentQuoteRepositoryImpl) GetExpiredActive(ctx context.Context, limit int) ([]*domainentities.PaymentQuote, error) {
	var ms []models.PaymentQuote
	if err := GetDB(ctx, r.db).
		Where("status = ? AND expires_at < ?", string(domainentities.PaymentQuoteStatusActive), time.Now()).
		Limit(limit).
		Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]*domainentities.PaymentQuote, 0, len(ms))
	for i := range ms {
		out = append(out, r.toEntity(&ms[i]))
	}
	return out, nil
}

func (r *PaymentQuoteRepositoryImpl) ExpireQuotes(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return GetDB(ctx, r.db).Model(&models.PaymentQuote{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     string(domainentities.PaymentQuoteStatusExpired),
			"updated_at": time.Now(),
		}).Error
}

func (r *PaymentQuoteRepositoryImpl) toEntity(m *models.PaymentQuote) *domainentities.PaymentQuote {
	return &domainentities.PaymentQuote{
		ID:                    m.ID,
		MerchantID:            m.MerchantID,
		InvoiceCurrency:       m.InvoiceCurrency,
		InvoiceAmount:         m.InvoiceAmount,
		SelectedChainID:       m.SelectedChainID,
		SelectedTokenAddress:  m.SelectedTokenAddress,
		SelectedTokenSymbol:   m.SelectedTokenSymbol,
		SelectedTokenDecimals: m.SelectedTokenDecimals,
		QuotedAmount:          m.QuotedAmount,
		QuoteRate:             m.QuoteRate,
		PriceSource:           m.PriceSource,
		Route:                 m.Route,
		SlippageBps:           m.SlippageBps,
		RateTimestamp:         m.RateTimestamp,
		ExpiresAt:             m.ExpiresAt,
		Status:                domainentities.PaymentQuoteStatus(m.Status),
		UsedAt:                m.UsedAt,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}

type PartnerPaymentSessionRepositoryImpl struct {
	db *gorm.DB
}

func NewPartnerPaymentSessionRepository(db *gorm.DB) *PartnerPaymentSessionRepositoryImpl {
	return &PartnerPaymentSessionRepositoryImpl{db: db}
}

func (r *PartnerPaymentSessionRepositoryImpl) Create(ctx context.Context, session *domainentities.PartnerPaymentSession) error {
	m := &models.PartnerPaymentSession{
		ID:                    session.ID,
		MerchantID:            session.MerchantID,
		QuoteID:               session.QuoteID,
		PaymentRequestID:      session.PaymentRequestID,
		InvoiceCurrency:       session.InvoiceCurrency,
		InvoiceAmount:         session.InvoiceAmount,
		SelectedChainID:       session.SelectedChainID,
		SelectedTokenAddress:  session.SelectedTokenAddress,
		SelectedTokenSymbol:   session.SelectedTokenSymbol,
		SelectedTokenDecimals: session.SelectedTokenDecimals,
		DestChain:             session.DestChain,
		DestToken:             session.DestToken,
		DestWallet:            session.DestWallet,
		PaymentAmount:         session.PaymentAmount,
		PaymentAmountDecimals: session.PaymentAmountDecimals,
		Status:                string(session.Status),
		ChannelUsed:           session.ChannelUsed,
		PaymentCode:           session.PaymentCode,
		PaymentURL:            session.PaymentURL,
		InstructionTo:         session.InstructionTo,
		InstructionValue:      session.InstructionValue,
		InstructionDataHex:    session.InstructionDataHex,
		InstructionDataBase58: session.InstructionDataBase58,
		InstructionDataBase64: session.InstructionDataBase64,
		InstructionApprovalTo: session.InstructionApprovalTo,
		InstructionApprovalDataHex: session.InstructionApprovalDataHex,
		QuoteRate:             session.QuoteRate,
		QuoteSource:           session.QuoteSource,
		QuoteRoute:            session.QuoteRoute,
		QuoteExpiresAt:        session.QuoteExpiresAt,
		QuoteSnapshotJSON:     session.QuoteSnapshotJSON,
		ExpiresAt:             session.ExpiresAt,
		PaidTxHash:            session.PaidTxHash,
		PaidChainID:           session.PaidChainID,
		PaidTokenAddress:      session.PaidTokenAddress,
		PaidAmount:            session.PaidAmount,
		PaidSenderAddress:     session.PaidSenderAddress,
		CompletedAt:           session.CompletedAt,
		CreatedAt:             session.CreatedAt,
		UpdatedAt:             session.UpdatedAt,
	}
	return GetDB(ctx, r.db).Create(m).Error
}

func (r *PartnerPaymentSessionRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domainentities.PartnerPaymentSession, error) {
	var m models.PartnerPaymentSession
	if err := GetDB(ctx, r.db).Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

func (r *PartnerPaymentSessionRepositoryImpl) GetByPaymentRequestID(ctx context.Context, paymentRequestID uuid.UUID) (*domainentities.PartnerPaymentSession, error) {
	var m models.PartnerPaymentSession
	if err := GetDB(ctx, r.db).Where("payment_request_id = ?", paymentRequestID).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

func (r *PartnerPaymentSessionRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status domainentities.PartnerPaymentSessionStatus) error {
	res := GetDB(ctx, r.db).Model(&models.PartnerPaymentSession{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     string(status),
		"updated_at": time.Now(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *PartnerPaymentSessionRepositoryImpl) MarkCompleted(ctx context.Context, id uuid.UUID, paidTxHash string) error {
	now := time.Now()
	res := GetDB(ctx, r.db).Model(&models.PartnerPaymentSession{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       string(domainentities.PartnerPaymentSessionStatusCompleted),
		"paid_tx_hash": paidTxHash,
		"completed_at": now,
		"updated_at":   now,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *PartnerPaymentSessionRepositoryImpl) GetExpiredPending(ctx context.Context, limit int) ([]*domainentities.PartnerPaymentSession, error) {
	var ms []models.PartnerPaymentSession
	if err := GetDB(ctx, r.db).
		Where("status = ? AND expires_at < ?", string(domainentities.PartnerPaymentSessionStatusPending), time.Now()).
		Limit(limit).
		Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]*domainentities.PartnerPaymentSession, 0, len(ms))
	for i := range ms {
		out = append(out, r.toEntity(&ms[i]))
	}
	return out, nil
}

func (r *PartnerPaymentSessionRepositoryImpl) ExpireSessions(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return GetDB(ctx, r.db).Model(&models.PartnerPaymentSession{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     string(domainentities.PartnerPaymentSessionStatusExpired),
			"updated_at": time.Now(),
		}).Error
}

func (r *PartnerPaymentSessionRepositoryImpl) toEntity(m *models.PartnerPaymentSession) *domainentities.PartnerPaymentSession {
	return &domainentities.PartnerPaymentSession{
		ID:                    m.ID,
		MerchantID:            m.MerchantID,
		QuoteID:               m.QuoteID,
		PaymentRequestID:      m.PaymentRequestID,
		InvoiceCurrency:       m.InvoiceCurrency,
		InvoiceAmount:         m.InvoiceAmount,
		SelectedChainID:       m.SelectedChainID,
		SelectedTokenAddress:  m.SelectedTokenAddress,
		SelectedTokenSymbol:   m.SelectedTokenSymbol,
		SelectedTokenDecimals: m.SelectedTokenDecimals,
		DestChain:             m.DestChain,
		DestToken:             m.DestToken,
		DestWallet:            m.DestWallet,
		PaymentAmount:         m.PaymentAmount,
		PaymentAmountDecimals: m.PaymentAmountDecimals,
		Status:                domainentities.PartnerPaymentSessionStatus(m.Status),
		ChannelUsed:           m.ChannelUsed,
		PaymentCode:           m.PaymentCode,
		PaymentURL:            m.PaymentURL,
		InstructionTo:         m.InstructionTo,
		InstructionValue:      m.InstructionValue,
		InstructionDataHex:    m.InstructionDataHex,
		InstructionDataBase58: m.InstructionDataBase58,
		InstructionDataBase64: m.InstructionDataBase64,
		InstructionApprovalTo: m.InstructionApprovalTo,
		InstructionApprovalDataHex: m.InstructionApprovalDataHex,
		QuoteRate:             m.QuoteRate,
		QuoteSource:           m.QuoteSource,
		QuoteRoute:            m.QuoteRoute,
		QuoteExpiresAt:        m.QuoteExpiresAt,
		QuoteSnapshotJSON:     m.QuoteSnapshotJSON,
		ExpiresAt:             m.ExpiresAt,
		PaidTxHash:            m.PaidTxHash,
		PaidChainID:           m.PaidChainID,
		PaidTokenAddress:      m.PaidTokenAddress,
		PaidAmount:            m.PaidAmount,
		PaidSenderAddress:     m.PaidSenderAddress,
		CompletedAt:           m.CompletedAt,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}
