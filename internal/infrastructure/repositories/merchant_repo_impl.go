package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

// MerchantRepository implements merchant data operations
type MerchantRepository struct {
	db *sql.DB
}

// NewMerchantRepository creates a new merchant repository
func NewMerchantRepository(db *sql.DB) *MerchantRepository {
	return &MerchantRepository{db: db}
}

// Create creates a new merchant
func (r *MerchantRepository) Create(ctx context.Context, merchant *entities.Merchant) error {
	query := `
		INSERT INTO merchants (
			id, user_id, business_name, business_email, merchant_type, 
			status, tax_id, business_address, documents, fee_discount_percent,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	merchant.ID = uuid.New()
	merchant.Status = entities.MerchantStatusPending
	merchant.CreatedAt = time.Now()
	merchant.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		merchant.ID,
		merchant.UserID,
		merchant.BusinessName,
		merchant.BusinessEmail,
		merchant.MerchantType,
		merchant.Status,
		merchant.TaxID,
		merchant.BusinessAddress,
		merchant.Documents,
		merchant.FeeDiscountPercent,
		merchant.CreatedAt,
		merchant.UpdatedAt,
	)

	return err
}

// GetByID gets a merchant by ID
func (r *MerchantRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Merchant, error) {
	query := `
		SELECT id, user_id, business_name, business_email, merchant_type, 
		       status, tax_id, business_address, documents, fee_discount_percent,
		       verified_at, created_at, updated_at
		FROM merchants
		WHERE id = $1 AND deleted_at IS NULL
	`

	merchant := &entities.Merchant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&merchant.ID,
		&merchant.UserID,
		&merchant.BusinessName,
		&merchant.BusinessEmail,
		&merchant.MerchantType,
		&merchant.Status,
		&merchant.TaxID,
		&merchant.BusinessAddress,
		&merchant.Documents,
		&merchant.FeeDiscountPercent,
		&merchant.VerifiedAt,
		&merchant.CreatedAt,
		&merchant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return merchant, nil
}

// GetByUserID gets a merchant by user ID
func (r *MerchantRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.Merchant, error) {
	query := `
		SELECT id, user_id, business_name, business_email, merchant_type, 
		       status, tax_id, business_address, documents, fee_discount_percent,
		       verified_at, created_at, updated_at
		FROM merchants
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	merchant := &entities.Merchant{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&merchant.ID,
		&merchant.UserID,
		&merchant.BusinessName,
		&merchant.BusinessEmail,
		&merchant.MerchantType,
		&merchant.Status,
		&merchant.TaxID,
		&merchant.BusinessAddress,
		&merchant.Documents,
		&merchant.FeeDiscountPercent,
		&merchant.VerifiedAt,
		&merchant.CreatedAt,
		&merchant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return merchant, nil
}

// Update updates a merchant
func (r *MerchantRepository) Update(ctx context.Context, merchant *entities.Merchant) error {
	query := `
		UPDATE merchants
		SET business_name = $2, business_email = $3, merchant_type = $4,
		    status = $5, tax_id = $6, business_address = $7, 
		    documents = $8, fee_discount_percent = $9, updated_at = $10
		WHERE id = $1 AND deleted_at IS NULL
	`

	merchant.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		merchant.ID,
		merchant.BusinessName,
		merchant.BusinessEmail,
		merchant.MerchantType,
		merchant.Status,
		merchant.TaxID,
		merchant.BusinessAddress,
		merchant.Documents,
		merchant.FeeDiscountPercent,
		merchant.UpdatedAt,
	)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}

// UpdateStatus updates merchant status
func (r *MerchantRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.MerchantStatus) error {
	query := `
		UPDATE merchants
		SET status = $2, updated_at = $3, verified_at = CASE WHEN $2 = 'active' THEN NOW() ELSE verified_at END
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, status, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}

// SoftDelete soft deletes a merchant
func (r *MerchantRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE merchants SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}
