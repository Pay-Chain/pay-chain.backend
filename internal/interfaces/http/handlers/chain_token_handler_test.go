package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

type chainRepoStub struct {
	items map[uuid.UUID]*entities.Chain
}

func newChainRepoStub() *chainRepoStub { return &chainRepoStub{items: map[uuid.UUID]*entities.Chain{}} }

func (s *chainRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Chain, error) {
	if c, ok := s.items[id]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *chainRepoStub) GetByChainID(_ context.Context, chainID string) (*entities.Chain, error) {
	for _, c := range s.items {
		if c.ChainID == chainID {
			return c, nil
		}
	}
	return nil, domainerrors.ErrNotFound
}
func (s *chainRepoStub) GetByCAIP2(_ context.Context, caip2 string) (*entities.Chain, error) {
	for _, c := range s.items {
		if c.GetCAIP2ID() == caip2 {
			return c, nil
		}
	}
	return nil, domainerrors.ErrNotFound
}
func (s *chainRepoStub) GetAll(_ context.Context) ([]*entities.Chain, error) { return nil, nil }
func (s *chainRepoStub) GetAllRPCs(_ context.Context, _ *uuid.UUID, _ *bool, _ *string, _ utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *chainRepoStub) GetActive(_ context.Context, _ utils.PaginationParams) ([]*entities.Chain, int64, error) {
	out := make([]*entities.Chain, 0, len(s.items))
	for _, c := range s.items {
		out = append(out, c)
	}
	return out, int64(len(out)), nil
}
func (s *chainRepoStub) Create(_ context.Context, chain *entities.Chain) error {
	s.items[chain.ID] = chain
	return nil
}
func (s *chainRepoStub) Update(_ context.Context, chain *entities.Chain) error {
	if _, ok := s.items[chain.ID]; !ok {
		return domainerrors.ErrNotFound
	}
	s.items[chain.ID] = chain
	return nil
}
func (s *chainRepoStub) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.items[id]; !ok {
		return domainerrors.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

type tokenRepoStub struct {
	items map[uuid.UUID]*entities.Token
}

func newTokenRepoStub() *tokenRepoStub { return &tokenRepoStub{items: map[uuid.UUID]*entities.Token{}} }

func (s *tokenRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Token, error) {
	if t, ok := s.items[id]; ok {
		return t, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *tokenRepoStub) GetBySymbol(_ context.Context, _ string, _ uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *tokenRepoStub) GetByAddress(_ context.Context, address string, chainID uuid.UUID) (*entities.Token, error) {
	for _, t := range s.items {
		if t.ContractAddress == address && t.ChainUUID == chainID {
			return t, nil
		}
	}
	return nil, domainerrors.ErrNotFound
}
func (s *tokenRepoStub) GetAll(_ context.Context) ([]*entities.Token, error) {
	out := make([]*entities.Token, 0, len(s.items))
	for _, t := range s.items {
		out = append(out, t)
	}
	return out, nil
}
func (s *tokenRepoStub) GetStablecoins(_ context.Context) ([]*entities.Token, error) {
	return s.GetAll(context.Background())
}
func (s *tokenRepoStub) GetNative(_ context.Context, _ uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *tokenRepoStub) GetTokensByChain(_ context.Context, chainID uuid.UUID, _ utils.PaginationParams) ([]*entities.Token, int64, error) {
	out := make([]*entities.Token, 0)
	for _, t := range s.items {
		if t.ChainUUID == chainID {
			out = append(out, t)
		}
	}
	return out, int64(len(out)), nil
}
func (s *tokenRepoStub) GetAllTokens(_ context.Context, _ *uuid.UUID, _ *string, _ utils.PaginationParams) ([]*entities.Token, int64, error) {
	out, _ := s.GetAll(context.Background())
	return out, int64(len(out)), nil
}
func (s *tokenRepoStub) Create(_ context.Context, token *entities.Token) error {
	s.items[token.ID] = token
	return nil
}
func (s *tokenRepoStub) Update(_ context.Context, token *entities.Token) error {
	if _, ok := s.items[token.ID]; !ok {
		return domainerrors.ErrNotFound
	}
	s.items[token.ID] = token
	return nil
}
func (s *tokenRepoStub) SoftDelete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.items[id]; !ok {
		return domainerrors.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func TestChainHandler_CRUDAndList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainRepo := newChainRepoStub()
	h := NewChainHandler(chainRepo)
	r := gin.New()
	r.GET("/chains", h.ListChains)
	r.POST("/admin/chains", h.CreateChain)
	r.PUT("/admin/chains/:id", h.UpdateChain)
	r.DELETE("/admin/chains/:id", h.DeleteChain)

	// Create
	createBody := map[string]any{
		"networkId": "8453", "name": "Base", "chainType": "EVM", "rpcUrl": "https://rpc", "symbol": "ETH",
	}
	b, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/admin/chains", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d body=%s", rec.Code, rec.Body.String())
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/chains?page=1&limit=10", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Items []map[string]any `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &listResp)
	if len(listResp.Items) == 0 {
		t.Fatal("expected at least one chain")
	}
	id, _ := listResp.Items[0]["id"].(string)

	// Update
	upd := map[string]any{
		"networkId": "8453", "name": "Base Updated", "chainType": "EVM", "rpcUrl": "https://rpc", "symbol": "ETH", "isActive": true,
	}
	b, _ = json.Marshal(upd)
	req = httptest.NewRequest(http.MethodPut, "/admin/chains/"+id, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rec.Code, rec.Body.String())
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/admin/chains/"+id, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestChainHandler_InvalidInputs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewChainHandler(newChainRepoStub())
	r := gin.New()
	r.PUT("/admin/chains/:id", h.UpdateChain)
	r.DELETE("/admin/chains/:id", h.DeleteChain)

	req := httptest.NewRequest(http.MethodPut, "/admin/chains/not-uuid", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/admin/chains/not-uuid", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}
}

func TestTokenHandler_CoreFlows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainRepo := newChainRepoStub()
	tokenRepo := newTokenRepoStub()

	chain := &entities.Chain{ID: uuid.New(), ChainID: "8453", Type: entities.ChainTypeEVM, Name: "Base", IsActive: true}
	chainRepo.items[chain.ID] = chain
	token := &entities.Token{ID: uuid.New(), ChainUUID: chain.ID, Symbol: "USDC", Name: "USD Coin", Decimals: 6, ContractAddress: "0xUSDC", IsActive: true}
	tokenRepo.items[token.ID] = token

	h := NewTokenHandler(tokenRepo, chainRepo)
	r := gin.New()
	r.GET("/tokens", h.ListSupportedTokens)
	r.GET("/tokens/stablecoins", h.ListStablecoins)
	r.POST("/admin/tokens", h.CreateToken)
	r.PUT("/admin/tokens/:id", h.UpdateToken)
	r.DELETE("/admin/tokens/:id", h.DeleteToken)

	// List by chainId fallback via GetByChainID
	req := httptest.NewRequest(http.MethodGet, "/tokens?chainId=8453", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rec.Code, rec.Body.String())
	}

	// List all
	req = httptest.NewRequest(http.MethodGet, "/tokens?page=1&limit=20", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rec.Code, rec.Body.String())
	}

	// Stablecoins
	req = httptest.NewRequest(http.MethodGet, "/tokens/stablecoins", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rec.Code, rec.Body.String())
	}

	// Create token
	createReq := map[string]any{
		"symbol": "IDRX", "name": "IDRX", "decimals": 6, "type": "ERC20", "chainId": chain.ID.String(), "contractAddress": "0xIDRX", "minAmount": "1",
	}
	b, _ := json.Marshal(createReq)
	req = httptest.NewRequest(http.MethodPost, "/admin/tokens", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d body=%s", rec.Code, rec.Body.String())
	}

	// Update existing token
	upd := map[string]any{"name": "USD Coin Updated", "chainId": "8453"}
	b, _ = json.Marshal(upd)
	req = httptest.NewRequest(http.MethodPut, "/admin/tokens/"+token.ID.String(), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rec.Code, rec.Body.String())
	}

	// Delete token
	req = httptest.NewRequest(http.MethodDelete, "/admin/tokens/"+token.ID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTokenHandler_InvalidInputs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTokenHandler(newTokenRepoStub(), newChainRepoStub())
	r := gin.New()
	r.POST("/admin/tokens", h.CreateToken)
	r.PUT("/admin/tokens/:id", h.UpdateToken)
	r.DELETE("/admin/tokens/:id", h.DeleteToken)

	// Invalid create (required fields missing)
	req := httptest.NewRequest(http.MethodPost, "/admin/tokens", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}

	// Invalid token id
	req = httptest.NewRequest(http.MethodPut, "/admin/tokens/not-uuid", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/admin/tokens/not-uuid", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}
}
