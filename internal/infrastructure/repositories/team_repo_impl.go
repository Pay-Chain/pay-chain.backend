package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
)

type TeamRepository struct {
	db *gorm.DB
}

func NewTeamRepository(db *gorm.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(ctx context.Context, team *entities.Team) error {
	m := r.toModel(team)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	team.ID = m.ID
	team.CreatedAt = m.CreatedAt
	team.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *TeamRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Team, error) {
	var m models.Team
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

func (r *TeamRepository) ListPublic(ctx context.Context) ([]*entities.Team, error) {
	var ms []models.Team
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("display_order ASC, created_at ASC").
		Find(&ms).Error; err != nil {
		return nil, err
	}

	items := make([]*entities.Team, 0, len(ms))
	for i := range ms {
		items = append(items, r.toEntity(&ms[i]))
	}
	return items, nil
}

func (r *TeamRepository) ListAdmin(ctx context.Context, search string) ([]*entities.Team, error) {
	var ms []models.Team
	query := r.db.WithContext(ctx).Model(&models.Team{})
	if strings.TrimSpace(search) != "" {
		term := "%" + strings.TrimSpace(search) + "%"
		query = query.Where("name ILIKE ? OR role ILIKE ?", term, term)
	}
	if err := query.Order("display_order ASC, created_at ASC").Find(&ms).Error; err != nil {
		return nil, err
	}

	items := make([]*entities.Team, 0, len(ms))
	for i := range ms {
		items = append(items, r.toEntity(&ms[i]))
	}
	return items, nil
}

func (r *TeamRepository) Update(ctx context.Context, team *entities.Team) error {
	updates := map[string]interface{}{
		"name":          team.Name,
		"role":          team.Role,
		"bio":           team.Bio,
		"image_url":     team.ImageURL,
		"github_url":    team.GithubURL,
		"twitter_url":   team.TwitterURL,
		"linkedin_url":  team.LinkedInURL,
		"display_order": team.DisplayOrder,
		"is_active":     team.IsActive,
		"updated_at":    time.Now(),
	}

	result := r.db.WithContext(ctx).
		Model(&models.Team{}).
		Where("id = ?", team.ID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *TeamRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Team{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *TeamRepository) toEntity(m *models.Team) *entities.Team {
	var deletedAt *time.Time
	if m.DeletedAt.Valid {
		deletedAt = &m.DeletedAt.Time
	}
	return &entities.Team{
		ID:           m.ID,
		Name:         m.Name,
		Role:         m.Role,
		Bio:          m.Bio,
		ImageURL:     m.ImageURL,
		GithubURL:    m.GithubURL,
		TwitterURL:   m.TwitterURL,
		LinkedInURL:  m.LinkedInURL,
		DisplayOrder: m.DisplayOrder,
		IsActive:     m.IsActive,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    deletedAt,
	}
}

func (r *TeamRepository) toModel(e *entities.Team) *models.Team {
	return &models.Team{
		ID:           e.ID,
		Name:         e.Name,
		Role:         e.Role,
		Bio:          e.Bio,
		ImageURL:     e.ImageURL,
		GithubURL:    e.GithubURL,
		TwitterURL:   e.TwitterURL,
		LinkedInURL:  e.LinkedInURL,
		DisplayOrder: e.DisplayOrder,
		IsActive:     e.IsActive,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
