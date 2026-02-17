package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestEmailVerificationRepository_CRUDLikeFlow(t *testing.T) {
	db := newTestDB(t)
	createUserTable(t, db)
	mustExec(t, db, `CREATE TABLE email_verifications (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		verified_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)

	repo := NewEmailVerificationRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	mustExec(t, db, `INSERT INTO users(id,email,name,role,kyc_status,password_hash,is_email_verified,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		userID.String(), "u@paychain.io", "User", "USER", "NOT_STARTED", "hash", false, time.Now(), time.Now(),
	)

	require.NoError(t, repo.Create(ctx, userID, "token-1"))

	user, err := repo.GetByToken(ctx, "token-1")
	require.NoError(t, err)
	require.Equal(t, userID, user.ID)

	require.NoError(t, repo.MarkVerified(ctx, "token-1"))

	_, err = repo.GetByToken(ctx, "token-1")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.MarkVerified(ctx, "token-1")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestEmailVerificationRepository_DBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// intentionally skip table creation
	repo := NewEmailVerificationRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, uuid.New(), "token")
	require.Error(t, err)

	_, err = repo.GetByToken(ctx, "token")
	require.Error(t, err)

	err = repo.MarkVerified(ctx, "token")
	require.Error(t, err)
}
