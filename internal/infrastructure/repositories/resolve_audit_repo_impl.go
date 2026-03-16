package repositories

import (
	"context"
	"payment-kita.backend/internal/domain/entities"
	domainRepos "payment-kita.backend/internal/domain/repositories"
	"gorm.io/gorm"
)

type postgresResolveAuditRepository struct {
	db *gorm.DB
}

func NewResolveAuditRepository(db *gorm.DB) domainRepos.AuditRepository {
	return &postgresResolveAuditRepository{db: db}
}

func (r *postgresResolveAuditRepository) LogResolveAttempt(ctx context.Context, audit *entities.ResolveAudit) error {
	return r.db.WithContext(ctx).Table("pk_resolve_audit").Create(audit).Error
}
