package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestAdminHandler_ErrorBranchesAndStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	merchantID := uuid.New()

	h := NewAdminHandler(
		&adminUserRepoStub{
			listFn: func(context.Context, string) ([]*entities.User, error) {
				return nil, errors.New("list users failed")
			},
		},
		&adminMerchantRepoStub{
			listFn: func(context.Context) ([]*entities.Merchant, error) {
				return nil, errors.New("list merchants failed")
			},
			getByID: func(_ context.Context, id uuid.UUID) (*entities.Merchant, error) {
				if id == merchantID {
					return nil, errors.New("get merchant failed")
				}
				return nil, domainerrors.ErrNotFound
			},
		},
		adminPaymentRepoStub{},
	)

	r := gin.New()
	r.GET("/users", h.ListUsers)
	r.GET("/merchants", h.ListMerchants)
	r.PUT("/merchants/:id/status", h.UpdateMerchantStatus)
	r.GET("/stats", h.GetStats)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/merchants", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/merchants/not-uuid/status", bytes.NewBufferString(`{"status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/merchants/"+merchantID.String()+"/status", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/merchants/"+merchantID.String()+"/status", bytes.NewBufferString(`{"status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/merchants/"+uuid.NewString()+"/status", bytes.NewBufferString(`{"status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/stats", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "totalUsers")
}

type teamRepoErrorStub struct {
	listPublicFn func(context.Context) ([]*entities.Team, error)
	listAdminFn  func(context.Context, string) ([]*entities.Team, error)
	createFn     func(context.Context, *entities.Team) error
	getByIDFn    func(context.Context, uuid.UUID) (*entities.Team, error)
	updateFn     func(context.Context, *entities.Team) error
	deleteFn     func(context.Context, uuid.UUID) error
}

func (s *teamRepoErrorStub) Create(ctx context.Context, team *entities.Team) error {
	if s.createFn != nil {
		return s.createFn(ctx, team)
	}
	return nil
}

func (s *teamRepoErrorStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.Team, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, domainerrors.ErrNotFound
}

func (s *teamRepoErrorStub) ListPublic(ctx context.Context) ([]*entities.Team, error) {
	if s.listPublicFn != nil {
		return s.listPublicFn(ctx)
	}
	return nil, nil
}

func (s *teamRepoErrorStub) ListAdmin(ctx context.Context, search string) ([]*entities.Team, error) {
	if s.listAdminFn != nil {
		return s.listAdminFn(ctx, search)
	}
	return nil, nil
}

func (s *teamRepoErrorStub) Update(ctx context.Context, team *entities.Team) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, team)
	}
	return nil
}

func (s *teamRepoErrorStub) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

func TestTeamHandler_ErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	teamID := uuid.New()
	errID := uuid.New()

	h := NewTeamHandler(&teamRepoErrorStub{
		listPublicFn: func(context.Context) ([]*entities.Team, error) {
			return nil, errors.New("list public failed")
		},
		listAdminFn: func(context.Context, string) ([]*entities.Team, error) {
			return nil, errors.New("list admin failed")
		},
		createFn: func(context.Context, *entities.Team) error {
			return errors.New("create failed")
		},
		getByIDFn: func(_ context.Context, id uuid.UUID) (*entities.Team, error) {
			if id == teamID {
				return &entities.Team{ID: teamID, Name: "A", Role: "R", Bio: "B", ImageURL: "I"}, nil
			}
			if id == errID {
				return nil, errors.New("get failed")
			}
			return nil, domainerrors.ErrNotFound
		},
		updateFn: func(_ context.Context, team *entities.Team) error {
			if team.Name == "missing" {
				return domainerrors.ErrNotFound
			}
			return errors.New("update failed")
		},
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			if id == teamID {
				return errors.New("delete failed")
			}
			return domainerrors.ErrNotFound
		},
	})

	r := gin.New()
	r.GET("/teams", h.ListPublicTeams)
	r.GET("/admin/teams", h.ListAdminTeams)
	r.POST("/admin/teams", h.CreateTeam)
	r.PUT("/admin/teams/:id", h.UpdateTeam)
	r.DELETE("/admin/teams/:id", h.DeleteTeam)

	req := httptest.NewRequest(http.MethodGet, "/teams", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/admin/teams?search=x", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/admin/teams", bytes.NewBufferString(`{"name":"A","role":"R","bio":"B","imageUrl":"I"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/admin/teams/not-uuid", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/admin/teams/"+uuid.NewString(), bytes.NewBufferString(`{"name":"A","role":"R","bio":"B","imageUrl":"I","isActive":true}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/admin/teams/"+errID.String(), bytes.NewBufferString(`{"name":"A","role":"R","bio":"B","imageUrl":"I","isActive":true}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/admin/teams/"+teamID.String(), bytes.NewBufferString(`{"name":"missing","role":"R","bio":"B","imageUrl":"I","isActive":true}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/admin/teams/"+teamID.String(), bytes.NewBufferString(`{"name":"other","role":"R","bio":"B","imageUrl":"I","isActive":true}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/admin/teams/"+teamID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/admin/teams/"+uuid.NewString(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/admin/teams/not-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/admin/teams", bytes.NewBufferString(`{"name":"A","role":"R","bio":"B","imageUrl":"I","githubUrl":"","twitterUrl":"","linkedinUrl":"","displayOrder":1,"isActive":true}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
