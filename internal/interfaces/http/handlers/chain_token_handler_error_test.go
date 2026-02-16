package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

type chainRepoErrStub struct {
	*chainRepoStub
	getActiveErr   error
	createErr      error
	updateErr      error
	deleteErr      error
	getByChainIDFn func(context.Context, string) (*entities.Chain, error)
}

func (s *chainRepoErrStub) GetActive(ctx context.Context, p utils.PaginationParams) ([]*entities.Chain, int64, error) {
	if s.getActiveErr != nil {
		return nil, 0, s.getActiveErr
	}
	return s.chainRepoStub.GetActive(ctx, p)
}
func (s *chainRepoErrStub) Create(ctx context.Context, chain *entities.Chain) error {
	if s.createErr != nil {
		return s.createErr
	}
	return s.chainRepoStub.Create(ctx, chain)
}
func (s *chainRepoErrStub) Update(ctx context.Context, chain *entities.Chain) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.chainRepoStub.Update(ctx, chain)
}
func (s *chainRepoErrStub) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.chainRepoStub.Delete(ctx, id)
}
func (s *chainRepoErrStub) GetByChainID(ctx context.Context, chainID string) (*entities.Chain, error) {
	if s.getByChainIDFn != nil {
		return s.getByChainIDFn(ctx, chainID)
	}
	return s.chainRepoStub.GetByChainID(ctx, chainID)
}

type tokenRepoErrStub struct {
	*tokenRepoStub
	getByChainErr error
	getAllErr     error
	stableErr     error
	createErr     error
	getByIDErr    error
	updateErr     error
	deleteErr     error
}

func (s *tokenRepoErrStub) GetTokensByChain(ctx context.Context, chainID uuid.UUID, p utils.PaginationParams) ([]*entities.Token, int64, error) {
	if s.getByChainErr != nil {
		return nil, 0, s.getByChainErr
	}
	return s.tokenRepoStub.GetTokensByChain(ctx, chainID, p)
}
func (s *tokenRepoErrStub) GetAllTokens(ctx context.Context, chainID *uuid.UUID, search *string, p utils.PaginationParams) ([]*entities.Token, int64, error) {
	if s.getAllErr != nil {
		return nil, 0, s.getAllErr
	}
	return s.tokenRepoStub.GetAllTokens(ctx, chainID, search, p)
}
func (s *tokenRepoErrStub) GetStablecoins(ctx context.Context) ([]*entities.Token, error) {
	if s.stableErr != nil {
		return nil, s.stableErr
	}
	return s.tokenRepoStub.GetStablecoins(ctx)
}
func (s *tokenRepoErrStub) Create(ctx context.Context, token *entities.Token) error {
	if s.createErr != nil {
		return s.createErr
	}
	return s.tokenRepoStub.Create(ctx, token)
}
func (s *tokenRepoErrStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.Token, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	return s.tokenRepoStub.GetByID(ctx, id)
}
func (s *tokenRepoErrStub) Update(ctx context.Context, token *entities.Token) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.tokenRepoStub.Update(ctx, token)
}
func (s *tokenRepoErrStub) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.tokenRepoStub.SoftDelete(ctx, id)
}

func TestChainHandler_ErrorPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &chainRepoErrStub{chainRepoStub: newChainRepoStub()}
	h := NewChainHandler(repo)
	r := gin.New()
	r.GET("/chains", h.ListChains)
	r.POST("/admin/chains", h.CreateChain)
	r.PUT("/admin/chains/:id", h.UpdateChain)
	r.DELETE("/admin/chains/:id", h.DeleteChain)

	repo.getActiveErr = errors.New("db error")
	req := httptest.NewRequest(http.MethodGet, "/chains", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	repo.getActiveErr = nil

	body, _ := json.Marshal(map[string]any{
		"networkId": "8453", "name": "Base", "chainType": "EVM", "rpcUrl": "https://rpc", "symbol": "ETH",
	})
	repo.createErr = errors.New("insert fail")
	req = httptest.NewRequest(http.MethodPost, "/admin/chains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	repo.createErr = nil

	id := uuid.New()
	repo.items[id] = &entities.Chain{ID: id}
	updBody, _ := json.Marshal(map[string]any{
		"networkId": "8453", "name": "Base", "chainType": "EVM", "rpcUrl": "https://rpc", "symbol": "ETH", "isActive": true,
	})

	repo.updateErr = domainerrors.ErrNotFound
	req = httptest.NewRequest(http.MethodPut, "/admin/chains/"+id.String(), bytes.NewReader(updBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	repo.updateErr = errors.New("update fail")
	req = httptest.NewRequest(http.MethodPut, "/admin/chains/"+id.String(), bytes.NewReader(updBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	repo.updateErr = nil

	repo.deleteErr = domainerrors.ErrNotFound
	req = httptest.NewRequest(http.MethodDelete, "/admin/chains/"+id.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	repo.deleteErr = errors.New("delete fail")
	req = httptest.NewRequest(http.MethodDelete, "/admin/chains/"+id.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}

	// create invalid JSON validation branch
	req = httptest.NewRequest(http.MethodPost, "/admin/chains", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid create payload, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTokenHandler_ErrorPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainRepo := &chainRepoErrStub{chainRepoStub: newChainRepoStub()}
	tokenRepo := &tokenRepoErrStub{tokenRepoStub: newTokenRepoStub()}

	chainID := uuid.New()
	chainRepo.items[chainID] = &entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}
	tokenID := uuid.New()
	tokenRepo.items[tokenID] = &entities.Token{ID: tokenID, ChainUUID: chainID, Symbol: "USDC"}

	h := NewTokenHandler(tokenRepo, chainRepo)
	r := gin.New()
	r.GET("/tokens", h.ListSupportedTokens)
	r.GET("/tokens/stablecoins", h.ListStablecoins)
	r.POST("/admin/tokens", h.CreateToken)
	r.PUT("/admin/tokens/:id", h.UpdateToken)
	r.DELETE("/admin/tokens/:id", h.DeleteToken)

	chainRepo.getByChainIDFn = func(context.Context, string) (*entities.Chain, error) {
		return nil, domainerrors.ErrNotFound
	}
	req := httptest.NewRequest(http.MethodGet, "/tokens?chainId=bad-chain", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	chainRepo.getByChainIDFn = nil

	tokenRepo.getByChainErr = errors.New("chain query fail")
	req = httptest.NewRequest(http.MethodGet, "/tokens?chainId="+chainID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	tokenRepo.getByChainErr = nil

	tokenRepo.getAllErr = errors.New("all query fail")
	req = httptest.NewRequest(http.MethodGet, "/tokens", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	tokenRepo.getAllErr = nil

	tokenRepo.stableErr = errors.New("stable fail")
	req = httptest.NewRequest(http.MethodGet, "/tokens/stablecoins", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	tokenRepo.stableErr = nil

	createBody, _ := json.Marshal(map[string]any{
		"symbol": "IDRX", "name": "IDRX", "decimals": 6, "type": "ERC20", "chainId": chainID.String(),
	})
	tokenRepo.createErr = errors.New("insert fail")
	req = httptest.NewRequest(http.MethodPost, "/admin/tokens", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	tokenRepo.createErr = nil

	tokenRepo.getByIDErr = domainerrors.ErrNotFound
	req = httptest.NewRequest(http.MethodPut, "/admin/tokens/"+tokenID.String(), bytes.NewReader([]byte(`{"name":"X"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	tokenRepo.getByIDErr = nil

	// malformed update payload should fail bind
	req = httptest.NewRequest(http.MethodPut, "/admin/tokens/"+tokenID.String(), bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid update payload, got %d body=%s", rec.Code, rec.Body.String())
	}

	tokenRepo.updateErr = errors.New("update fail")
	req = httptest.NewRequest(http.MethodPut, "/admin/tokens/"+tokenID.String(), bytes.NewReader([]byte(`{"name":"X"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	tokenRepo.updateErr = nil

	tokenRepo.deleteErr = errors.New("delete fail")
	req = httptest.NewRequest(http.MethodDelete, "/admin/tokens/"+tokenID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
}
