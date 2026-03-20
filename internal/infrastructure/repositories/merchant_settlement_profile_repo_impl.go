package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	domainentities "payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/infrastructure/models"
)

type MerchantSettlementProfileRepositoryImpl struct {
	db *gorm.DB
}

func NewMerchantSettlementProfileRepository(db *gorm.DB) *MerchantSettlementProfileRepositoryImpl {
	return &MerchantSettlementProfileRepositoryImpl{db: db}
}

func (r *MerchantSettlementProfileRepositoryImpl) GetByMerchantID(ctx context.Context, merchantID uuid.UUID) (*domainentities.MerchantSettlementProfile, error) {
	var m models.MerchantSettlementProfile
	if err := GetDB(ctx, r.db).Where("merchant_id = ?", merchantID).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

func (r *MerchantSettlementProfileRepositoryImpl) ListMissingMerchantIDs(ctx context.Context) ([]uuid.UUID, error) {
	type result struct {
		ID uuid.UUID
	}
	var rows []result
	err := GetDB(ctx, r.db).
		Raw(`
			SELECT m.id
			FROM merchants m
			LEFT JOIN merchant_settlement_profiles msp
				ON msp.merchant_id = m.id
				AND msp.deleted_at IS NULL
			WHERE m.deleted_at IS NULL
			  AND msp.id IS NULL
			ORDER BY m.created_at DESC
		`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ID)
	}
	return out, nil
}

func (r *MerchantSettlementProfileRepositoryImpl) HasProfilesByMerchantIDs(ctx context.Context, merchantIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := make(map[uuid.UUID]bool, len(merchantIDs))
	if len(merchantIDs) == 0 {
		return out, nil
	}
	type row struct {
		MerchantID uuid.UUID
	}
	var rows []row
	if err := GetDB(ctx, r.db).
		Model(&models.MerchantSettlementProfile{}).
		Select("merchant_id").
		Where("merchant_id IN ?", merchantIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, id := range merchantIDs {
		out[id] = false
	}
	for _, row := range rows {
		out[row.MerchantID] = true
	}
	return out, nil
}

func (r *MerchantSettlementProfileRepositoryImpl) Upsert(ctx context.Context, profile *domainentities.MerchantSettlementProfile) error {
	if profile == nil {
		return domainerrors.BadRequest("profile is required")
	}
	now := time.Now().UTC()
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	m := &models.MerchantSettlementProfile{
		ID:                profile.ID,
		MerchantID:        profile.MerchantID,
		InvoiceCurrency:   profile.InvoiceCurrency,
		DestChain:         profile.DestChain,
		DestToken:         profile.DestToken,
		DestWallet:        profile.DestWallet,
		BridgeTokenSymbol: profile.BridgeTokenSymbol,
		CreatedAt:         profile.CreatedAt,
		UpdatedAt:         profile.UpdatedAt,
	}

	return GetDB(ctx, r.db).Clauses(clauseOnConflictMerchantSettlementProfile()).Create(m).Error
}

func clauseOnConflictMerchantSettlementProfile() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{{Name: "merchant_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"invoice_currency":    gorm.Expr("EXCLUDED.invoice_currency"),
			"dest_chain":          gorm.Expr("EXCLUDED.dest_chain"),
			"dest_token":          gorm.Expr("EXCLUDED.dest_token"),
			"dest_wallet":         gorm.Expr("EXCLUDED.dest_wallet"),
			"bridge_token_symbol": gorm.Expr("EXCLUDED.bridge_token_symbol"),
			"updated_at":          gorm.Expr("EXCLUDED.updated_at"),
		}),
	}
}

func (r *MerchantSettlementProfileRepositoryImpl) toEntity(m *models.MerchantSettlementProfile) *domainentities.MerchantSettlementProfile {
	return &domainentities.MerchantSettlementProfile{
		ID:                m.ID,
		MerchantID:        m.MerchantID,
		InvoiceCurrency:   m.InvoiceCurrency,
		DestChain:         m.DestChain,
		DestToken:         m.DestToken,
		DestWallet:        m.DestWallet,
		BridgeTokenSymbol: m.BridgeTokenSymbol,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}
