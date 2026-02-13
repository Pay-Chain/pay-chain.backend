package entities

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Role         string     `json:"role"`
	Bio          string     `json:"bio"`
	ImageURL     string     `json:"imageUrl"`
	GithubURL    string     `json:"githubUrl,omitempty"`
	TwitterURL   string     `json:"twitterUrl,omitempty"`
	LinkedInURL  string     `json:"linkedinUrl,omitempty"`
	DisplayOrder int        `json:"displayOrder"`
	IsActive     bool       `json:"isActive"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}
