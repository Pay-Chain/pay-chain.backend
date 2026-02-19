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
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/utils"
)

type cfgChainRepoStub struct {
	getAllFn func(ctx context.Context) ([]*entities.Chain, error)
}

func (s *cfgChainRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *cfgChainRepoStub) GetByChainID(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *cfgChainRepoStub) GetByCAIP2(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *cfgChainRepoStub) GetAll(ctx context.Context) ([]*entities.Chain, error) {
	if s.getAllFn != nil {
		return s.getAllFn(ctx)
	}
	return []*entities.Chain{}, nil
}
func (s *cfgChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *cfgChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *cfgChainRepoStub) Create(context.Context, *entities.Chain) error       { return nil }
func (s *cfgChainRepoStub) Update(context.Context, *entities.Chain) error       { return nil }
func (s *cfgChainRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }
func (s *cfgChainRepoStub) CreateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *cfgChainRepoStub) UpdateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *cfgChainRepoStub) DeleteRPC(context.Context, uuid.UUID) error          { return nil }
func (s *cfgChainRepoStub) GetRPCByID(context.Context, uuid.UUID) (*entities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}

type cfgTokenRepoStub struct{}

func (cfgTokenRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (cfgTokenRepoStub) GetBySymbol(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (cfgTokenRepoStub) GetByAddress(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (cfgTokenRepoStub) GetAll(context.Context) ([]*entities.Token, error)         { return nil, nil }
func (cfgTokenRepoStub) GetStablecoins(context.Context) ([]*entities.Token, error) { return nil, nil }
func (cfgTokenRepoStub) GetNative(context.Context, uuid.UUID) (*entities.Token, error) {
	return nil, domainerrors.ErrNotFound
}
func (cfgTokenRepoStub) GetTokensByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (cfgTokenRepoStub) GetAllTokens(context.Context, *uuid.UUID, *string, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (cfgTokenRepoStub) Create(context.Context, *entities.Token) error { return nil }
func (cfgTokenRepoStub) Update(context.Context, *entities.Token) error { return nil }
func (cfgTokenRepoStub) SoftDelete(context.Context, uuid.UUID) error   { return nil }

type cfgContractRepoStub struct{}

func (cfgContractRepoStub) Create(context.Context, *entities.SmartContract) error { return nil }
func (cfgContractRepoStub) GetByID(context.Context, uuid.UUID) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (cfgContractRepoStub) GetByChainAndAddress(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (cfgContractRepoStub) GetActiveContract(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (cfgContractRepoStub) GetFiltered(context.Context, *uuid.UUID, entities.SmartContractType, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (cfgContractRepoStub) GetByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (cfgContractRepoStub) GetAll(context.Context, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (cfgContractRepoStub) Update(context.Context, *entities.SmartContract) error { return nil }
func (cfgContractRepoStub) SoftDelete(context.Context, uuid.UUID) error           { return nil }

func TestCrosschainConfigHandler_OverviewAndValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	okUsecase := usecases.NewCrosschainConfigUsecase(
		&cfgChainRepoStub{
			getAllFn: func(context.Context) ([]*entities.Chain, error) { return []*entities.Chain{}, nil },
		},
		cfgTokenRepoStub{},
		cfgContractRepoStub{},
		nil,
		&usecases.OnchainAdapterUsecase{},
	)
	errorUsecase := usecases.NewCrosschainConfigUsecase(
		&cfgChainRepoStub{
			getAllFn: func(context.Context) ([]*entities.Chain, error) { return nil, errors.New("failed-all") },
		},
		cfgTokenRepoStub{},
		cfgContractRepoStub{},
		nil,
		&usecases.OnchainAdapterUsecase{},
	)

	okHandler := NewCrosschainConfigHandler(okUsecase)
	errHandler := NewCrosschainConfigHandler(errorUsecase)

	r := gin.New()
	r.GET("/ok", okHandler.Overview)
	r.GET("/err", errHandler.Overview)
	r.GET("/preflight", okHandler.Preflight)
	r.POST("/recheck", okHandler.Recheck)
	r.POST("/autofix", okHandler.AutoFix)
	r.POST("/recheck-bulk", okHandler.RecheckBulk)
	r.POST("/autofix-bulk", okHandler.AutoFixBulk)

	req := httptest.NewRequest(http.MethodGet, "/ok?page=1&limit=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"items\"")

	req = httptest.NewRequest(http.MethodGet, "/err", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/preflight", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/recheck", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/autofix", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/recheck-bulk", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/autofix-bulk", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
