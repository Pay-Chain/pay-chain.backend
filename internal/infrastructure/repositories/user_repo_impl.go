package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

// UserRepository implements user data operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *entities.User) error {
	query := `
		INSERT INTO users (id, email, name, password_hash, role, kyc_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.PasswordHash,
		user.Role,
		user.KYCStatus,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

// GetByID gets a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	query := `
		SELECT id, email, name, password_hash, role, kyc_status, kyc_verified_at, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &entities.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Role,
		&user.KYCStatus,
		&user.KYCVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetByEmail gets a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	query := `
		SELECT id, email, name, password_hash, role, kyc_status, kyc_verified_at, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &entities.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Role,
		&user.KYCStatus,
		&user.KYCVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *entities.User) error {
	query := `
		UPDATE users
		SET name = $2, role = $3, kyc_status = $4, kyc_verified_at = $5, updated_at = $6
		WHERE id = $1 AND deleted_at IS NULL
	`

	user.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Name,
		user.Role,
		user.KYCStatus,
		user.KYCVerifiedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}

// SoftDelete soft deletes a user
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}

// EmailVerificationRepository implements email verification operations
type EmailVerificationRepository struct {
	db *sql.DB
}

// NewEmailVerificationRepository creates a new email verification repository
func NewEmailVerificationRepository(db *sql.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

// Create creates a new email verification
func (r *EmailVerificationRepository) Create(ctx context.Context, userID uuid.UUID, token string) error {
	query := `
		INSERT INTO email_verifications (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		uuid.New(),
		userID,
		token,
		time.Now().Add(24*time.Hour), // 24 hour expiry
		time.Now(),
	)

	return err
}

// GetByToken gets user by verification token
func (r *EmailVerificationRepository) GetByToken(ctx context.Context, token string) (*entities.User, error) {
	query := `
		SELECT u.id, u.email, u.name, u.password_hash, u.role, u.kyc_status, u.kyc_verified_at, u.created_at, u.updated_at
		FROM email_verifications ev
		JOIN users u ON ev.user_id = u.id
		WHERE ev.token = $1 
		AND ev.expires_at > NOW() 
		AND ev.verified_at IS NULL
		AND ev.deleted_at IS NULL
		AND u.deleted_at IS NULL
	`

	user := &entities.User{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Role,
		&user.KYCStatus,
		&user.KYCVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// MarkVerified marks an email verification as verified
func (r *EmailVerificationRepository) MarkVerified(ctx context.Context, token string) error {
	query := `
		UPDATE email_verifications
		SET verified_at = $2
		WHERE token = $1 AND verified_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, token, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}
