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
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/utils"
)

type walletRepoStub struct {
	items       map[uuid.UUID]*entities.Wallet
	addressToID map[string]uuid.UUID
}

func newWalletRepoStub() *walletRepoStub {
	return &walletRepoStub{
		items:       map[uuid.UUID]*entities.Wallet{},
		addressToID: map[string]uuid.UUID{},
	}
}

func walletKey(chainID uuid.UUID, address string) string {
	return chainID.String() + ":" + address
}

func (s *walletRepoStub) Create(_ context.Context, wallet *entities.Wallet) error {
	id := wallet.ID
	if id == uuid.Nil {
		id = utils.GenerateUUIDv7()
		wallet.ID = id
	}
	cpy := *wallet
	s.items[id] = &cpy
	s.addressToID[walletKey(wallet.ChainID, wallet.Address)] = id
	return nil
}

func (s *walletRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Wallet, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	cpy := *item
	return &cpy, nil
}

func (s *walletRepoStub) GetByUserID(_ context.Context, userID uuid.UUID) ([]*entities.Wallet, error) {
	out := []*entities.Wallet{}
	for _, item := range s.items {
		if item.UserID != nil && *item.UserID == userID {
			cpy := *item
			out = append(out, &cpy)
		}
	}
	return out, nil
}

func (s *walletRepoStub) GetByAddress(_ context.Context, chainID uuid.UUID, address string) (*entities.Wallet, error) {
	id, ok := s.addressToID[walletKey(chainID, address)]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	cpy := *s.items[id]
	return &cpy, nil
}

func (s *walletRepoStub) SetPrimary(_ context.Context, userID, walletID uuid.UUID) error {
	item, ok := s.items[walletID]
	if !ok {
		return domainerrors.ErrNotFound
	}
	if item.UserID == nil || *item.UserID != userID {
		return domainerrors.ErrForbidden
	}
	for _, w := range s.items {
		if w.UserID != nil && *w.UserID == userID {
			w.IsPrimary = false
		}
	}
	item.IsPrimary = true
	return nil
}

func (s *walletRepoStub) SoftDelete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.items[id]; !ok {
		return domainerrors.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

type walletUserRepoStub struct {
	user *entities.User
}

func (s walletUserRepoStub) Create(context.Context, *entities.User) error { return nil }
func (s walletUserRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.User, error) {
	if s.user != nil && s.user.ID == id {
		return s.user, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s walletUserRepoStub) GetByEmail(context.Context, string) (*entities.User, error) {
	return nil, domainerrors.ErrNotFound
}
func (s walletUserRepoStub) Update(context.Context, *entities.User) error            { return nil }
func (s walletUserRepoStub) UpdatePassword(context.Context, uuid.UUID, string) error { return nil }
func (s walletUserRepoStub) SoftDelete(context.Context, uuid.UUID) error             { return nil }
func (s walletUserRepoStub) List(context.Context, string) ([]*entities.User, error)  { return nil, nil }

type walletChainRepoStub struct {
	chain *entities.Chain
}

func (s walletChainRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Chain, error) {
	if s.chain != nil && s.chain.ID == id {
		return s.chain, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s walletChainRepoStub) GetByChainID(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s walletChainRepoStub) GetByCAIP2(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s walletChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) { return nil, nil }
func (s walletChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s walletChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s walletChainRepoStub) Create(context.Context, *entities.Chain) error       { return nil }
func (s walletChainRepoStub) Update(context.Context, *entities.Chain) error       { return nil }
func (s walletChainRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }
func (s walletChainRepoStub) CreateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s walletChainRepoStub) UpdateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s walletChainRepoStub) DeleteRPC(context.Context, uuid.UUID) error          { return nil }
func (s walletChainRepoStub) GetRPCByID(context.Context, uuid.UUID) (*entities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}

func TestWalletHandler_SuccessFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	chainID := uuid.New()
	repo := newWalletRepoStub()

	uc := usecases.NewWalletUsecase(
		repo,
		walletUserRepoStub{user: &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}},
		walletChainRepoStub{chain: &entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}},
	)
	h := NewWalletHandler(uc)

	r := gin.New()
	withUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
	r.POST("/wallets/connect", withUser, h.ConnectWallet)
	r.GET("/wallets", withUser, h.ListWallets)
	r.PUT("/wallets/:id/primary", withUser, h.SetPrimaryWallet)
	r.DELETE("/wallets/:id", withUser, h.DisconnectWallet)

	connectBody := []byte(`{"chainId":"` + chainID.String() + `","address":"0xabc","signature":"sig","message":"msg"}`)
	req := httptest.NewRequest(http.MethodPost, "/wallets/connect", bytes.NewReader(connectBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var created struct {
		Wallet entities.Wallet `json:"wallet"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal connect response: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/wallets", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/wallets/"+created.Wallet.ID.String()+"/primary", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/wallets/"+created.Wallet.ID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestWalletHandler_ErrorMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	otherUserID := uuid.New()
	chainID := uuid.New()
	repo := newWalletRepoStub()

	// seed one wallet owned by other user
	seedID := utils.GenerateUUIDv7()
	repo.items[seedID] = &entities.Wallet{
		ID:        seedID,
		UserID:    &otherUserID,
		ChainID:   chainID,
		Address:   "0xseed",
		IsPrimary: true,
	}

	uc := usecases.NewWalletUsecase(
		repo,
		walletUserRepoStub{user: &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}},
		walletChainRepoStub{chain: &entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}},
	)
	h := NewWalletHandler(uc)

	r := gin.New()
	withUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
	r.PUT("/wallets/:id/primary", withUser, h.SetPrimaryWallet)
	r.DELETE("/wallets/:id", withUser, h.DisconnectWallet)

	req := httptest.NewRequest(http.MethodPut, "/wallets/"+uuid.NewString()+"/primary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for set primary not found, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/wallets/"+seedID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disconnect forbidden, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestWalletHandler_ConnectWallet_ErrorPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	otherUserID := uuid.New()
	chainID := uuid.New()

	repo := newWalletRepoStub()
	seedID := utils.GenerateUUIDv7()
	repo.items[seedID] = &entities.Wallet{
		ID:      seedID,
		UserID:  &otherUserID,
		ChainID: chainID,
		Address: "0xdup",
	}
	repo.addressToID[walletKey(chainID, "0xdup")] = seedID

	uc := usecases.NewWalletUsecase(
		repo,
		walletUserRepoStub{user: &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}},
		walletChainRepoStub{chain: &entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}},
	)
	h := NewWalletHandler(uc)

	r := gin.New()
	withUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
	r.POST("/wallets/connect", withUser, h.ConnectWallet)

	req := httptest.NewRequest(http.MethodPost, "/wallets/connect", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad json, got %d body=%s", w.Code, w.Body.String())
	}

	noAuthRouter := gin.New()
	noAuthRouter.POST("/wallets/connect", h.ConnectWallet)
	connectBody := []byte(`{"chainId":"` + chainID.String() + `","address":"0xabc","signature":"sig","message":"msg"}`)
	req = httptest.NewRequest(http.MethodPost, "/wallets/connect", bytes.NewReader(connectBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	noAuthRouter.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing auth, got %d body=%s", w.Code, w.Body.String())
	}

	connectDupBody := []byte(`{"chainId":"` + chainID.String() + `","address":"0xdup","signature":"sig","message":"msg"}`)
	req = httptest.NewRequest(http.MethodPost, "/wallets/connect", bytes.NewReader(connectDupBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for wallet already exists, got %d body=%s", w.Code, w.Body.String())
	}

	// Generic error branch: user lookup fails (not mapped by special handler cases).
	req = httptest.NewRequest(http.MethodPost, "/wallets/connect", bytes.NewReader(connectBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	genericErrRouter := gin.New()
	ucGeneric := usecases.NewWalletUsecase(
		repo,
		walletUserRepoStub{user: nil},
		walletChainRepoStub{chain: &entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}},
	)
	genericErrRouter.POST("/wallets/connect", withUser, NewWalletHandler(ucGeneric).ConnectWallet)
	genericErrRouter.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for generic connect error branch, got %d body=%s", w.Code, w.Body.String())
	}
}

type walletRepoErrorPathStub struct {
	*walletRepoStub
	getByIDFn     func(context.Context, uuid.UUID) (*entities.Wallet, error)
	getByUserIDFn func(context.Context, uuid.UUID) ([]*entities.Wallet, error)
	softDeleteFn  func(context.Context, uuid.UUID) error
}

func (s *walletRepoErrorPathStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.Wallet, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return s.walletRepoStub.GetByID(ctx, id)
}

func (s *walletRepoErrorPathStub) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error) {
	if s.getByUserIDFn != nil {
		return s.getByUserIDFn(ctx, userID)
	}
	return s.walletRepoStub.GetByUserID(ctx, userID)
}

func (s *walletRepoErrorPathStub) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if s.softDeleteFn != nil {
		return s.softDeleteFn(ctx, id)
	}
	return s.walletRepoStub.SoftDelete(ctx, id)
}

func TestWalletHandler_ListAndDisconnect_ErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	chainID := uuid.New()
	walletID := utils.GenerateUUIDv7()

	baseRepo := newWalletRepoStub()
	baseRepo.items[walletID] = &entities.Wallet{ID: walletID, UserID: &userID, ChainID: chainID, Address: "0xabc"}

	repo := &walletRepoErrorPathStub{walletRepoStub: baseRepo}
	uc := usecases.NewWalletUsecase(
		repo,
		walletUserRepoStub{user: &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}},
		walletChainRepoStub{chain: &entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}},
	)
	h := NewWalletHandler(uc)

	withUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
	r := gin.New()
	r.GET("/wallets", withUser, h.ListWallets)
	r.DELETE("/wallets/:id", withUser, h.DisconnectWallet)

	repo.getByUserIDFn = func(context.Context, uuid.UUID) ([]*entities.Wallet, error) {
		return nil, errors.New("list fail")
	}
	req := httptest.NewRequest(http.MethodGet, "/wallets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for list error, got %d body=%s", w.Code, w.Body.String())
	}

	repo.getByUserIDFn = func(context.Context, uuid.UUID) ([]*entities.Wallet, error) {
		return nil, nil
	}
	req = httptest.NewRequest(http.MethodGet, "/wallets", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for nil wallet list, got %d body=%s", w.Code, w.Body.String())
	}

	repo.getByIDFn = func(context.Context, uuid.UUID) (*entities.Wallet, error) { return nil, domainerrors.ErrNotFound }
	req = httptest.NewRequest(http.MethodDelete, "/wallets/"+walletID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wallet not found, got %d body=%s", w.Code, w.Body.String())
	}

	repo.getByIDFn = nil
	repo.softDeleteFn = func(context.Context, uuid.UUID) error { return errors.New("delete fail") }
	req = httptest.NewRequest(http.MethodDelete, "/wallets/"+walletID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for wallet delete error, got %d body=%s", w.Code, w.Body.String())
	}

	noAuth := gin.New()
	noAuth.DELETE("/wallets/:id", h.DisconnectWallet)
	req = httptest.NewRequest(http.MethodDelete, "/wallets/"+walletID.String(), nil)
	w = httptest.NewRecorder()
	noAuth.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing user on disconnect, got %d body=%s", w.Code, w.Body.String())
	}
}
