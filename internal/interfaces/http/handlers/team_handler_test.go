package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

type teamRepoStub struct {
	items map[uuid.UUID]*entities.Team
}

func newTeamRepoStub() *teamRepoStub {
	return &teamRepoStub{items: map[uuid.UUID]*entities.Team{}}
}

func (s *teamRepoStub) Create(_ context.Context, team *entities.Team) error {
	s.items[team.ID] = team
	return nil
}

func (s *teamRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Team, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return item, nil
}

func (s *teamRepoStub) ListPublic(_ context.Context) ([]*entities.Team, error) {
	out := make([]*entities.Team, 0)
	for _, item := range s.items {
		if item.IsActive {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *teamRepoStub) ListAdmin(_ context.Context, search string) ([]*entities.Team, error) {
	if strings.TrimSpace(search) == "" {
		out := make([]*entities.Team, 0, len(s.items))
		for _, item := range s.items {
			out = append(out, item)
		}
		return out, nil
	}
	out := make([]*entities.Team, 0)
	q := strings.ToLower(strings.TrimSpace(search))
	for _, item := range s.items {
		if strings.Contains(strings.ToLower(item.Name), q) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *teamRepoStub) Update(_ context.Context, team *entities.Team) error {
	if _, ok := s.items[team.ID]; !ok {
		return domainerrors.ErrNotFound
	}
	s.items[team.ID] = team
	return nil
}

func (s *teamRepoStub) SoftDelete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.items[id]; !ok {
		return domainerrors.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func TestTeamHandler_FullFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTeamRepoStub()
	h := NewTeamHandler(repo)

	r := gin.New()
	r.GET("/teams", h.ListPublicTeams)
	r.GET("/admin/teams", h.ListAdminTeams)
	r.POST("/admin/teams", h.CreateTeam)
	r.PUT("/admin/teams/:id", h.UpdateTeam)
	r.DELETE("/admin/teams/:id", h.DeleteTeam)

	createPayload := map[string]any{
		"name":         "Alice",
		"role":         "Engineer",
		"bio":          "Builder",
		"imageUrl":     "https://img",
		"displayOrder": 1,
		"isActive":     true,
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/admin/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created struct {
		Team entities.Team `json:"team"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.Team.Name != "Alice" {
		t.Fatalf("unexpected team name: %s", created.Team.Name)
	}

	// Public list
	req = httptest.NewRequest(http.MethodGet, "/teams", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Admin list
	req = httptest.NewRequest(http.MethodGet, "/admin/teams?search=ali", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Update
	updatePayload := map[string]any{
		"name":         "Alice Updated",
		"role":         "Lead Engineer",
		"bio":          "Builder++",
		"imageUrl":     "https://img2",
		"displayOrder": 2,
		"isActive":     true,
	}
	body, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest(http.MethodPut, "/admin/teams/"+created.Team.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Update without isActive should keep previous value (covers nil pointer branch).
	updatePayloadNoActive := map[string]any{
		"name":         "Alice Updated 2",
		"role":         "Principal Engineer",
		"bio":          "Builder+++",
		"imageUrl":     "https://img3",
		"displayOrder": 3,
	}
	body, _ = json.Marshal(updatePayloadNoActive)
	req = httptest.NewRequest(http.MethodPut, "/admin/teams/"+created.Team.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Malformed JSON for bind error branch with existing ID.
	req = httptest.NewRequest(http.MethodPut, "/admin/teams/"+created.Team.ID.String(), bytes.NewReader([]byte(`{"name":`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/admin/teams/"+created.Team.ID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTeamHandler_ValidationAndNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTeamRepoStub()
	h := NewTeamHandler(repo)

	r := gin.New()
	r.POST("/admin/teams", h.CreateTeam)
	r.PUT("/admin/teams/:id", h.UpdateTeam)
	r.DELETE("/admin/teams/:id", h.DeleteTeam)

	// Bad create payload
	req := httptest.NewRequest(http.MethodPost, "/admin/teams", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Invalid UUID
	req = httptest.NewRequest(http.MethodPut, "/admin/teams/not-uuid", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Not found on delete
	req = httptest.NewRequest(http.MethodDelete, "/admin/teams/"+uuid.NewString(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Required fields after trim should fail.
	req = httptest.NewRequest(http.MethodPost, "/admin/teams", bytes.NewReader([]byte(`{
		"name":"   ",
		"role":"Engineer",
		"bio":"Builder",
		"imageUrl":"https://img"
	}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
