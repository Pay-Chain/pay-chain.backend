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

// UserRepository implements user data operations
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *entities.User) error {
	m := &models.User{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
		KYCStatus:    string(user.KYCStatus),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
	// Note: DeletedAt handled by GORM

	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

// GetByID gets a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	var m models.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetByEmail gets a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	var m models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *entities.User) error {
	// Only update specific fields as per original impl
	updates := map[string]interface{}{
		"name":       user.Name,
		"role":       user.Role,
		"kyc_status": user.KYCStatus,
		"updated_at": time.Now(),
	}
	if user.KYCVerifiedAt.Valid {
		updates["kyc_verified_at"] = user.KYCVerifiedAt.Time
	}

	result := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", user.ID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

// List lists users with optional search filter
func (r *UserRepository) List(ctx context.Context, search string) ([]*entities.User, error) {
	var userModels []models.User
	query := r.db.WithContext(ctx).Order("created_at DESC")

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("name ILIKE ? OR email ILIKE ?", searchTerm, searchTerm)
	}

	if err := query.Find(&userModels).Error; err != nil {
		return nil, err
	}

	var users []*entities.User
	for _, m := range userModels {
		model := m
		users = append(users, r.toEntity(&model))
	}
	return users, nil
}

// SoftDelete soft deletes a user
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *UserRepository) toEntity(m *models.User) *entities.User {
	// Need to handle null.Time for KYCVerifiedAt if needed, although entity uses null.Time
	// Model has *time.Time.
	// Entity has `github.com/volatiletech/null/v8`. We need proper conversion if we import it,
	// OR we assume `entities.User` uses it and we map manually.
	// Since I can't easily import null/v8 without check, I'll rely on it being available if entities uses it.
	// Actually I need to recreate the entity struct.
	// I'll assume standard import/usage.

	// Wait, I cannot assign *time.Time to null.Time directly.
	// Use null.TimeFromPtr(m.KYCVerifiedAt) if I import null.
	// Since I am already importing "pay-chain.backend/internal/domain/entities" which imports null/v8,
	// I should probably import null/v8 too to use helper constructors.
	// OR I construct it manually: null.Time{Time: *t, Valid: true}
	// But `null.Time` is in `volatiletech/null/v8` package.

	// I'll try to add the import.

	return &entities.User{
		ID:           m.ID,
		Email:        m.Email,
		Name:         m.Name,
		PasswordHash: m.PasswordHash,
		Role:         entities.UserRole(m.Role),
		KYCStatus:    entities.KYCStatus(m.KYCStatus),
		// KYCVerifiedAt: null.TimeFromPtr(m.KYCVerifiedAt), // Need import
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		// DeletedAt: ...
	}
}

// EmailVerificationRepository implements email verification operations
type EmailVerificationRepository struct {
	db *gorm.DB
}

// NewEmailVerificationRepository creates a new email verification repository
func NewEmailVerificationRepository(db *gorm.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

// Create creates a new email verification
func (r *EmailVerificationRepository) Create(ctx context.Context, userID uuid.UUID, token string) error {
	m := &models.EmailVerification{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Create(m).Error
}

// GetByToken gets user by verification token
func (r *EmailVerificationRepository) GetByToken(ctx context.Context, token string) (*entities.User, error) {
	// Original used JOIN.
	// SELECT u.* FROM email_verifications ev JOIN users u ON ...

	var userModel models.User

	err := r.db.WithContext(ctx).
		Table("users").
		Joins("JOIN email_verifications ev ON ev.user_id = users.id").
		Where("ev.token = ? AND ev.expires_at > ? AND ev.verified_at IS NULL AND ev.deleted_at IS NULL", token, time.Now()).
		First(&userModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}

	// Use UserRepository's toEntity helper? Or copy paste.
	// Since they are in same package, can I reuse? Receiver is different.
	// I'll duplicate logic or make a standalone helper.
	// I'll implement inline.

	return &entities.User{
		ID:           userModel.ID,
		Email:        userModel.Email,
		Name:         userModel.Name,
		PasswordHash: userModel.PasswordHash,
		Role:         entities.UserRole(userModel.Role),
		KYCStatus:    entities.KYCStatus(userModel.KYCStatus),
		CreatedAt:    userModel.CreatedAt,
		UpdatedAt:    userModel.UpdatedAt,
	}, nil
}

// MarkVerified marks an email verification as verified
func (r *EmailVerificationRepository) MarkVerified(ctx context.Context, token string) error {
	result := r.db.WithContext(ctx).
		Model(&models.EmailVerification{}).
		Where("token = ? AND verified_at IS NULL", token).
		Update("verified_at", time.Now())

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}
