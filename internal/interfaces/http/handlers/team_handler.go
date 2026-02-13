package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/pkg/utils"
)

type TeamHandler struct {
	repo repositories.TeamRepository
}

func NewTeamHandler(repo repositories.TeamRepository) *TeamHandler {
	return &TeamHandler{repo: repo}
}

// ListPublicTeams returns active team members for public pages.
// GET /api/v1/teams
func (h *TeamHandler) ListPublicTeams(c *gin.Context) {
	items, err := h.repo.ListPublic(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"items": items})
}

// ListAdminTeams returns all team members for admin management.
// GET /api/v1/admin/teams
func (h *TeamHandler) ListAdminTeams(c *gin.Context) {
	search := c.Query("search")
	items, err := h.repo.ListAdmin(c.Request.Context(), search)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"items": items})
}

// CreateTeam creates team member.
// POST /api/v1/admin/teams
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	var input struct {
		Name         string `json:"name" binding:"required"`
		Role         string `json:"role" binding:"required"`
		Bio          string `json:"bio" binding:"required"`
		ImageURL     string `json:"imageUrl" binding:"required"`
		GithubURL    string `json:"githubUrl"`
		TwitterURL   string `json:"twitterUrl"`
		LinkedInURL  string `json:"linkedinUrl"`
		DisplayOrder int    `json:"displayOrder"`
		IsActive     *bool  `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Role) == "" || strings.TrimSpace(input.Bio) == "" || strings.TrimSpace(input.ImageURL) == "" {
		response.Error(c, domainerrors.BadRequest("name, role, bio, and imageUrl are required"))
		return
	}

	team := &entities.Team{
		ID:           utils.GenerateUUIDv7(),
		Name:         strings.TrimSpace(input.Name),
		Role:         strings.TrimSpace(input.Role),
		Bio:          strings.TrimSpace(input.Bio),
		ImageURL:     strings.TrimSpace(input.ImageURL),
		GithubURL:    strings.TrimSpace(input.GithubURL),
		TwitterURL:   strings.TrimSpace(input.TwitterURL),
		LinkedInURL:  strings.TrimSpace(input.LinkedInURL),
		DisplayOrder: input.DisplayOrder,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if input.IsActive != nil {
		team.IsActive = *input.IsActive
	}

	if err := h.repo.Create(c.Request.Context(), team); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, gin.H{
		"message": "Team member created",
		"team":    team,
	})
}

// UpdateTeam updates team member.
// PUT /api/v1/admin/teams/:id
func (h *TeamHandler) UpdateTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid team ID"))
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("team member not found"))
			return
		}
		response.Error(c, err)
		return
	}

	var input struct {
		Name         string `json:"name" binding:"required"`
		Role         string `json:"role" binding:"required"`
		Bio          string `json:"bio" binding:"required"`
		ImageURL     string `json:"imageUrl" binding:"required"`
		GithubURL    string `json:"githubUrl"`
		TwitterURL   string `json:"twitterUrl"`
		LinkedInURL  string `json:"linkedinUrl"`
		DisplayOrder int    `json:"displayOrder"`
		IsActive     *bool  `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	existing.Name = strings.TrimSpace(input.Name)
	existing.Role = strings.TrimSpace(input.Role)
	existing.Bio = strings.TrimSpace(input.Bio)
	existing.ImageURL = strings.TrimSpace(input.ImageURL)
	existing.GithubURL = strings.TrimSpace(input.GithubURL)
	existing.TwitterURL = strings.TrimSpace(input.TwitterURL)
	existing.LinkedInURL = strings.TrimSpace(input.LinkedInURL)
	existing.DisplayOrder = input.DisplayOrder
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}
	existing.UpdatedAt = time.Now()

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("team member not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Team member updated",
		"team":    existing,
	})
}

// DeleteTeam soft deletes team member.
// DELETE /api/v1/admin/teams/:id
func (h *TeamHandler) DeleteTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid team ID"))
		return
	}
	if err := h.repo.SoftDelete(c.Request.Context(), id); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("team member not found"))
			return
		}
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"message": "Team member deleted"})
}
