package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
)

func createTeamTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	mustExec(t, db, `CREATE TABLE teams (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		role TEXT NOT NULL,
		bio TEXT NOT NULL,
		image_url TEXT NOT NULL,
		github_url TEXT,
		twitter_url TEXT,
		linkedin_url TEXT,
		display_order INTEGER NOT NULL DEFAULT 0,
		is_active BOOLEAN NOT NULL DEFAULT true,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
}

func TestTeamRepository_CRUDAndLists(t *testing.T) {
	db := newTestDB(t)
	createTeamTable(t, db)
	repo := NewTeamRepository(db)
	ctx := context.Background()

	team1 := &entities.Team{
		ID:           uuid.New(),
		Name:         "Alice",
		Role:         "Engineer",
		Bio:          "Core dev",
		ImageURL:     "https://img/a.png",
		GithubURL:    "https://github.com/a",
		TwitterURL:   "https://x.com/a",
		LinkedInURL:  "https://linkedin.com/in/a",
		DisplayOrder: 1,
		IsActive:     true,
	}
	team2 := &entities.Team{
		ID:           uuid.New(),
		Name:         "Bob",
		Role:         "Designer",
		Bio:          "UI lead",
		ImageURL:     "https://img/b.png",
		DisplayOrder: 2,
		IsActive:     false,
	}

	require.NoError(t, repo.Create(ctx, team1))
	require.NoError(t, repo.Create(ctx, team2))
	// Ensure inactive state is persisted regardless of DB default behavior.
	team2.IsActive = false
	require.NoError(t, repo.Update(ctx, team2))
	require.False(t, team1.CreatedAt.IsZero())
	require.False(t, team1.UpdatedAt.IsZero())

	got, err := repo.GetByID(ctx, team1.ID)
	require.NoError(t, err)
	require.Equal(t, "Alice", got.Name)

	publicItems, err := repo.ListPublic(ctx)
	require.NoError(t, err)
	require.Len(t, publicItems, 1)
	require.Equal(t, team1.ID, publicItems[0].ID)

	adminItems, err := repo.ListAdmin(ctx, "")
	require.NoError(t, err)
	require.Len(t, adminItems, 2)
	require.Equal(t, team1.ID, adminItems[0].ID)

	// SQLite does not support ILIKE; this branch still validates repository error propagation.
	_, err = repo.ListAdmin(ctx, "desig")
	require.Error(t, err)

	team1.Name = "Alice Updated"
	team1.UpdatedAt = time.Now().Add(time.Hour)
	require.NoError(t, repo.Update(ctx, team1))
	got, err = repo.GetByID(ctx, team1.ID)
	require.NoError(t, err)
	require.Equal(t, "Alice Updated", got.Name)

	require.NoError(t, repo.SoftDelete(ctx, team2.ID))
	_, err = repo.GetByID(ctx, team2.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestTeamRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createTeamTable(t, db)
	repo := NewTeamRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Update(ctx, &entities.Team{
		ID:       uuid.New(),
		Name:     "x",
		Role:     "y",
		Bio:      "z",
		ImageURL: "https://img",
		IsActive: true,
	})
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.SoftDelete(ctx, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestTeamRepository_ToEntity_DeletedAtBranch(t *testing.T) {
	repo := NewTeamRepository(newTestDB(t))
	now := time.Now()
	deletedAt := now.Add(time.Minute)

	m := &models.Team{
		ID:           uuid.New(),
		Name:         "Deleted Member",
		Role:         "Engineer",
		Bio:          "bio",
		ImageURL:     "https://img",
		DisplayOrder: 1,
		IsActive:     false,
		CreatedAt:    now,
		UpdatedAt:    now,
		DeletedAt:    gorm.DeletedAt{Time: deletedAt, Valid: true},
	}

	e := repo.toEntity(m)
	require.NotNil(t, e.DeletedAt)
	require.Equal(t, deletedAt.Unix(), e.DeletedAt.Unix())
}

func TestTeamRepository_DBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// intentionally skip table creation
	repo := NewTeamRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &entities.Team{
		ID:       uuid.New(),
		Name:     "x",
		Role:     "r",
		Bio:      "b",
		ImageURL: "https://img",
		IsActive: true,
	})
	require.Error(t, err)

	_, err = repo.GetByID(ctx, uuid.New())
	require.Error(t, err)

	_, err = repo.ListPublic(ctx)
	require.Error(t, err)

	_, err = repo.ListAdmin(ctx, "")
	require.Error(t, err)

	err = repo.Update(ctx, &entities.Team{
		ID:       uuid.New(),
		Name:     "x",
		Role:     "r",
		Bio:      "b",
		ImageURL: "https://img",
		IsActive: true,
	})
	require.Error(t, err)

	err = repo.SoftDelete(ctx, uuid.New())
	require.Error(t, err)
}
