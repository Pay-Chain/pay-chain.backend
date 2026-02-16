package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
)

type apiKeyRepoStub struct {
	createFn     func(ctx context.Context, apiKey *entities.ApiKey) error
	findByUserFn func(ctx context.Context, userID uuid.UUID) ([]*entities.ApiKey, error)
	findByIDFn   func(ctx context.Context, id uuid.UUID) (*entities.ApiKey, error)
	deleteFn     func(ctx context.Context, id uuid.UUID) error
}

func (s *apiKeyRepoStub) Create(ctx context.Context, apiKey *entities.ApiKey) error {
	if s.createFn != nil {
		return s.createFn(ctx, apiKey)
	}
	return nil
}
func (s *apiKeyRepoStub) FindByKeyHash(context.Context, string) (*entities.ApiKey, error) { return nil, errors.New("unused") }
func (s *apiKeyRepoStub) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.ApiKey, error) {
	if s.findByUserFn != nil {
		return s.findByUserFn(ctx, userID)
	}
	return []*entities.ApiKey{}, nil
}
func (s *apiKeyRepoStub) FindByID(ctx context.Context, id uuid.UUID) (*entities.ApiKey, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id)
	}
	return nil, errors.New("not found")
}
func (s *apiKeyRepoStub) Update(context.Context, *entities.ApiKey) error { return nil }
func (s *apiKeyRepoStub) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

type apiKeyUserRepoStub struct{}

func (apiKeyUserRepoStub) Create(context.Context, *entities.User) error                         { return nil }
func (apiKeyUserRepoStub) GetByID(context.Context, uuid.UUID) (*entities.User, error)          { return nil, errors.New("unused") }
func (apiKeyUserRepoStub) GetByEmail(context.Context, string) (*entities.User, error)           { return nil, errors.New("unused") }
func (apiKeyUserRepoStub) Update(context.Context, *entities.User) error                         { return nil }
func (apiKeyUserRepoStub) UpdatePassword(context.Context, uuid.UUID, string) error              { return nil }
func (apiKeyUserRepoStub) SoftDelete(context.Context, uuid.UUID) error                          { return nil }
func (apiKeyUserRepoStub) List(context.Context, string) ([]*entities.User, error)               { return nil, nil }

func TestApiKeyHandler_CreateListRevoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	revokeID := uuid.New()

	repo := &apiKeyRepoStub{
		findByUserFn: func(_ context.Context, gotUserID uuid.UUID) ([]*entities.ApiKey, error) {
			require.Equal(t, userID, gotUserID)
			return []*entities.ApiKey{
				{ID: uuid.New(), Name: "Key A", UserID: userID},
			}, nil
		},
		findByIDFn: func(_ context.Context, id uuid.UUID) (*entities.ApiKey, error) {
			return &entities.ApiKey{ID: id, UserID: userID}, nil
		},
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			require.Equal(t, revokeID, id)
			return nil
		},
	}

	uc := usecases.NewApiKeyUsecase(
		repo,
		apiKeyUserRepoStub{},
		"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
	)
	h := NewApiKeyHandler(uc)

	r := gin.New()
	withUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
	r.POST("/api-keys", withUser, h.CreateApiKey)
	r.GET("/api-keys", withUser, h.ListApiKeys)
	r.DELETE("/api-keys/:id", withUser, h.RevokeApiKey)

	req := httptest.NewRequest(http.MethodPost, "/api-keys", strings.NewReader(`{"name":"Main","permissions":["read"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), `"name":"Main"`)
	require.Contains(t, w.Body.String(), `"apiKey"`)
	require.Contains(t, w.Body.String(), `"secretKey"`)

	req = httptest.NewRequest(http.MethodGet, "/api-keys", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Key A")

	req = httptest.NewRequest(http.MethodDelete, "/api-keys/"+revokeID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "revoked successfully")
}

func TestApiKeyHandler_ValidationPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := usecases.NewApiKeyUsecase(
		&apiKeyRepoStub{},
		apiKeyUserRepoStub{},
		"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
	)
	h := NewApiKeyHandler(uc)

	r := gin.New()
	r.POST("/api-keys", h.CreateApiKey)
	r.GET("/api-keys", h.ListApiKeys)
	r.DELETE("/api-keys/:id", h.RevokeApiKey)

	req := httptest.NewRequest(http.MethodPost, "/api-keys", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api-keys", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api-keys/not-a-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
