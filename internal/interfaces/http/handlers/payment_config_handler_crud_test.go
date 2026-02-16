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

type bridgeConfigRepoStub struct {
	items map[uuid.UUID]*entities.BridgeConfig
}

func newBridgeConfigRepoStub() *bridgeConfigRepoStub {
	return &bridgeConfigRepoStub{items: map[uuid.UUID]*entities.BridgeConfig{}}
}

func (s *bridgeConfigRepoStub) GetActive(context.Context, uuid.UUID, uuid.UUID) (*entities.BridgeConfig, error) {
	return nil, domainerrors.ErrNotFound
}

func (s *bridgeConfigRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.BridgeConfig, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	cpy := *item
	return &cpy, nil
}

func (s *bridgeConfigRepoStub) List(_ context.Context, sourceChainID, destChainID, bridgeID *uuid.UUID, _ utils.PaginationParams) ([]*entities.BridgeConfig, int64, error) {
	out := make([]*entities.BridgeConfig, 0, len(s.items))
	for _, v := range s.items {
		if sourceChainID != nil && v.SourceChainID != *sourceChainID {
			continue
		}
		if destChainID != nil && v.DestChainID != *destChainID {
			continue
		}
		if bridgeID != nil && v.BridgeID != *bridgeID {
			continue
		}
		cpy := *v
		out = append(out, &cpy)
	}
	return out, int64(len(out)), nil
}

func (s *bridgeConfigRepoStub) Create(_ context.Context, config *entities.BridgeConfig) error {
	cpy := *config
	s.items[config.ID] = &cpy
	return nil
}

func (s *bridgeConfigRepoStub) Update(_ context.Context, config *entities.BridgeConfig) error {
	if _, ok := s.items[config.ID]; !ok {
		return domainerrors.ErrNotFound
	}
	cpy := *config
	s.items[config.ID] = &cpy
	return nil
}

func (s *bridgeConfigRepoStub) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.items[id]; !ok {
		return domainerrors.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

type feeConfigRepoStub struct {
	items map[uuid.UUID]*entities.FeeConfig
}

func newFeeConfigRepoStub() *feeConfigRepoStub {
	return &feeConfigRepoStub{items: map[uuid.UUID]*entities.FeeConfig{}}
}

func (s *feeConfigRepoStub) GetByChainAndToken(context.Context, uuid.UUID, uuid.UUID) (*entities.FeeConfig, error) {
	return nil, domainerrors.ErrNotFound
}

func (s *feeConfigRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.FeeConfig, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	cpy := *item
	return &cpy, nil
}

func (s *feeConfigRepoStub) List(_ context.Context, chainID, tokenID *uuid.UUID, _ utils.PaginationParams) ([]*entities.FeeConfig, int64, error) {
	out := make([]*entities.FeeConfig, 0, len(s.items))
	for _, v := range s.items {
		if chainID != nil && v.ChainID != *chainID {
			continue
		}
		if tokenID != nil && v.TokenID != *tokenID {
			continue
		}
		cpy := *v
		out = append(out, &cpy)
	}
	return out, int64(len(out)), nil
}

func (s *feeConfigRepoStub) Create(_ context.Context, config *entities.FeeConfig) error {
	cpy := *config
	s.items[config.ID] = &cpy
	return nil
}

func (s *feeConfigRepoStub) Update(_ context.Context, config *entities.FeeConfig) error {
	if _, ok := s.items[config.ID]; !ok {
		return domainerrors.ErrNotFound
	}
	cpy := *config
	s.items[config.ID] = &cpy
	return nil
}

func (s *feeConfigRepoStub) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.items[id]; !ok {
		return domainerrors.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

type tokenRepoExistsStub struct {
	existing map[uuid.UUID]*entities.Token
}

func (s tokenRepoExistsStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Token, error) {
	item, ok := s.existing[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return item, nil
}
func (s tokenRepoExistsStub) GetBySymbol(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s tokenRepoExistsStub) GetByAddress(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (s tokenRepoExistsStub) GetAll(context.Context) ([]*entities.Token, error)                        { return nil, nil }
func (s tokenRepoExistsStub) GetStablecoins(context.Context) ([]*entities.Token, error)                 { return nil, nil }
func (s tokenRepoExistsStub) GetNative(context.Context, uuid.UUID) (*entities.Token, error)             { return nil, domainerrors.ErrNotFound }
func (s tokenRepoExistsStub) GetTokensByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (s tokenRepoExistsStub) GetAllTokens(context.Context, *uuid.UUID, *string, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (s tokenRepoExistsStub) Create(context.Context, *entities.Token) error  { return nil }
func (s tokenRepoExistsStub) Update(context.Context, *entities.Token) error  { return nil }
func (s tokenRepoExistsStub) SoftDelete(context.Context, uuid.UUID) error    { return nil }

func TestPaymentConfigHandler_BridgeConfigCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sourceChain := uuid.New()
	destChain := uuid.New()
	bridgeID := uuid.New()
	repo := newBridgeConfigRepoStub()

	h := NewPaymentConfigHandler(
		nil,
		repo,
		nil,
		&crosschainChainRepoStub{
			getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
				switch chainID {
				case "eip155:8453":
					return &entities.Chain{ID: sourceChain}, nil
				case "eip155:42161":
					return &entities.Chain{ID: destChain}, nil
				default:
					return nil, domainerrors.ErrNotFound
				}
			},
			getByCAIP2: func(context.Context, string) (*entities.Chain, error) { return nil, domainerrors.ErrNotFound },
		},
		nil,
	)

	r := gin.New()
	r.POST("/bridge-configs", h.CreateBridgeConfig)
	r.GET("/bridge-configs", h.ListBridgeConfigs)
	r.PUT("/bridge-configs/:id", h.UpdateBridgeConfig)
	r.DELETE("/bridge-configs/:id", h.DeleteBridgeConfig)

	createBody := []byte(`{"bridgeId":"` + bridgeID.String() + `","sourceChainId":"eip155:8453","destChainId":"eip155:42161","routerAddress":"0xabc","feePercentage":"0.10","config":"{\"mode\":\"fast\"}","isActive":true}`)
	req := httptest.NewRequest(http.MethodPost, "/bridge-configs", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created struct {
		Config entities.BridgeConfig `json:"config"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/bridge-configs?sourceChainId=eip155:8453&destChainId=eip155:42161&bridgeId="+bridgeID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	updateBody := []byte(`{"bridgeId":"` + bridgeID.String() + `","sourceChainId":"eip155:8453","destChainId":"eip155:42161","routerAddress":"0xdef","feePercentage":"0.20","config":"{}","isActive":false}`)
	req = httptest.NewRequest(http.MethodPut, "/bridge-configs/"+created.Config.ID.String(), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/bridge-configs/"+created.Config.ID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPaymentConfigHandler_FeeConfigCRUDAndTokenNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainID := uuid.New()
	tokenID := uuid.New()
	repo := newFeeConfigRepoStub()

	h := NewPaymentConfigHandler(
		nil,
		nil,
		repo,
		&crosschainChainRepoStub{
			getByChainID: func(_ context.Context, chainIDInput string) (*entities.Chain, error) {
				if chainIDInput == "eip155:8453" {
					return &entities.Chain{ID: chainID}, nil
				}
				return nil, domainerrors.ErrNotFound
			},
			getByCAIP2: func(context.Context, string) (*entities.Chain, error) { return nil, domainerrors.ErrNotFound },
		},
		tokenRepoExistsStub{
			existing: map[uuid.UUID]*entities.Token{
				tokenID: {ID: tokenID, Symbol: "USDC"},
			},
		},
	)

	r := gin.New()
	r.POST("/fee-configs", h.CreateFeeConfig)
	r.GET("/fee-configs", h.ListFeeConfigs)
	r.PUT("/fee-configs/:id", h.UpdateFeeConfig)
	r.DELETE("/fee-configs/:id", h.DeleteFeeConfig)

	createBody := []byte(`{"chainId":"eip155:8453","tokenId":"` + tokenID.String() + `","platformFeePercent":"0.1","fixedBaseFee":"1","minFee":"0.01"}`)
	req := httptest.NewRequest(http.MethodPost, "/fee-configs", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created struct {
		Config entities.FeeConfig `json:"config"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/fee-configs?chainId=eip155:8453&tokenId="+tokenID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	updateBody := []byte(`{"chainId":"eip155:8453","tokenId":"` + tokenID.String() + `","platformFeePercent":"0.2","fixedBaseFee":"2","minFee":"0.02"}`)
	req = httptest.NewRequest(http.MethodPut, "/fee-configs/"+created.Config.ID.String(), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/fee-configs/"+created.Config.ID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// token not found branch
	createBody = []byte(`{"chainId":"eip155:8453","tokenId":"` + uuid.NewString() + `","platformFeePercent":"0.1"}`)
	req = httptest.NewRequest(http.MethodPost, "/fee-configs", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown token, got %d body=%s", rec.Code, rec.Body.String())
	}
}
