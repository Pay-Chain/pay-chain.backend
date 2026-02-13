package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Team struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
	Name         string    `gorm:"type:varchar(120);not null"`
	Role         string    `gorm:"type:varchar(120);not null"`
	Bio          string    `gorm:"type:text;not null"`
	ImageURL     string    `gorm:"type:text;not null"`
	GithubURL    string    `gorm:"type:text"`
	TwitterURL   string    `gorm:"type:text"`
	LinkedInURL  string    `gorm:"column:linkedin_url;type:text"`
	DisplayOrder int       `gorm:"not null;default:0"`
	IsActive     bool      `gorm:"not null;default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
