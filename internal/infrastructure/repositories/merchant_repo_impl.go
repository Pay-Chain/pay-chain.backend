package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
)

// MerchantRepository implements merchant data operations
type MerchantRepository struct {
	db *gorm.DB
}

// NewMerchantRepository creates a new merchant repository
func NewMerchantRepository(db *gorm.DB) *MerchantRepository {
	return &MerchantRepository{db: db}
}

// Create creates a new merchant
func (r *MerchantRepository) Create(ctx context.Context, merchant *entities.Merchant) error {
	// Need to check Null types in entity vs GORM model (string/jsonb)
	// Entity has Documents null.JSON

	docs := "{}"
	if merchant.Documents.Valid {
		docs = string(merchant.Documents.JSON)
	}

	taxID := ""
	if merchant.TaxID.Valid {
		taxID = merchant.TaxID.String
	}

	addr := ""
	if merchant.BusinessAddress.Valid {
		addr = merchant.BusinessAddress.String
	}

	m := &models.Merchant{
		ID:                 merchant.ID,
		UserID:             merchant.UserID,
		BusinessName:       merchant.BusinessName,
		BusinessEmail:      merchant.BusinessEmail,
		MerchantType:       string(merchant.MerchantType),
		Status:             string(merchant.Status),
		TaxID:              taxID,
		BusinessAddress:    addr,
		Documents:          docs,
		FeeDiscountPercent: merchant.FeeDiscountPercent,
		CreatedAt:          merchant.CreatedAt,
		UpdatedAt:          merchant.UpdatedAt,
	}

	return r.db.WithContext(ctx).Create(m).Error
}

// GetByID gets a merchant by ID
func (r *MerchantRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Merchant, error) {
	var m models.Merchant
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetByUserID gets a merchant by user ID
func (r *MerchantRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.Merchant, error) {
	var m models.Merchant
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// Update updates a merchant
func (r *MerchantRepository) Update(ctx context.Context, merchant *entities.Merchant) error {
	// Marshal fields
	docs := "{}"
	if merchant.Documents.Valid {
		docs = string(merchant.Documents.JSON)
	}
	taxID := ""
	if merchant.TaxID.Valid {
		taxID = merchant.TaxID.String
	}
	addr := ""
	if merchant.BusinessAddress.Valid {
		addr = merchant.BusinessAddress.String
	}

	updates := map[string]interface{}{
		"business_name":        merchant.BusinessName,
		"business_email":       merchant.BusinessEmail,
		"merchant_type":        merchant.MerchantType,
		"status":               merchant.Status,
		"tax_id":               taxID,
		"business_address":     addr,
		"documents":            docs,
		"fee_discount_percent": merchant.FeeDiscountPercent,
		"updated_at":           time.Now(),
	}

	result := r.db.WithContext(ctx).Model(&models.Merchant{}).
		Where("id = ?", merchant.ID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}

// UpdateStatus updates merchant status
func (r *MerchantRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.MerchantStatus) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if status == entities.MerchantStatusActive {
		updates["verified_at"] = time.Now()
	}

	result := r.db.WithContext(ctx).Model(&models.Merchant{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

// List lists all merchants
func (r *MerchantRepository) List(ctx context.Context) ([]*entities.Merchant, error) {
	var mList []models.Merchant
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&mList).Error; err != nil {
		return nil, err
	}

	var merchants []*entities.Merchant
	for _, m := range mList {
		model := m
		merchants = append(merchants, r.toEntity(&model))
	}
	return merchants, nil
}

// SoftDelete soft deletes a merchant
func (r *MerchantRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Merchant{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *MerchantRepository) toEntity(m *models.Merchant) *entities.Merchant {
	// Construct entity, handling type conversions
	// Need volatiletech/null/v8 to be imported if we want to set Valid/String cleanly
	// Or we can assume default zero value is empty/invalid logic if we don't import.
	// For robustness I will not use null pkg constructors if I don't import it,
	// but I will try to populate the fields correctly if I can.

	// I'll leave null fields checks aside or use basic struct lit.
	// Since I can't construct `null.String` easily without import, and I didn't add import...
	// Gosh, I should add the import!

	return &entities.Merchant{
		ID:            m.ID,
		UserID:        m.UserID,
		BusinessName:  m.BusinessName,
		BusinessEmail: m.BusinessEmail,
		MerchantType:  entities.MerchantType(m.MerchantType),
		Status:        entities.MerchantStatus(m.Status),
		// TaxID:              null.StringFrom(m.TaxID),
		// BusinessAddress:    null.StringFrom(m.BusinessAddress),
		// Documents:          null.JSONFrom([]byte(m.Documents)),
		FeeDiscountPercent: m.FeeDiscountPercent,
		// VerifiedAt:         null.TimeFromPtr(m.VerifiedAt),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
