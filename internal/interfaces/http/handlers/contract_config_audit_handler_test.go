package handlers

import (
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
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/utils"
)

type contractAuditChainRepoStub struct {
	chainsByID map[uuid.UUID]*entities.Chain
	chainByID  map[string]*entities.Chain
	allErr     error
}

func (s *contractAuditChainRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.Chain, error) {
	if ch, ok := s.chainsByID[id]; ok {
		return ch, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *contractAuditChainRepoStub) GetByChainID(_ context.Context, chainID string) (*entities.Chain, error) {
	if ch, ok := s.chainByID[chainID]; ok {
		return ch, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *contractAuditChainRepoStub) GetByCAIP2(_ context.Context, caip2 string) (*entities.Chain, error) {
	if ch, ok := s.chainByID[caip2]; ok {
		return ch, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *contractAuditChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) {
	if s.allErr != nil {
		return nil, s.allErr
	}
	items := make([]*entities.Chain, 0, len(s.chainsByID))
	for _, ch := range s.chainsByID {
		items = append(items, ch)
	}
	return items, nil
}
func (s *contractAuditChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}
func (s *contractAuditChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (s *contractAuditChainRepoStub) Create(context.Context, *entities.Chain) error       { return nil }
func (s *contractAuditChainRepoStub) Update(context.Context, *entities.Chain) error       { return nil }
func (s *contractAuditChainRepoStub) Delete(context.Context, uuid.UUID) error             { return nil }
func (s *contractAuditChainRepoStub) CreateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *contractAuditChainRepoStub) UpdateRPC(context.Context, *entities.ChainRPC) error { return nil }
func (s *contractAuditChainRepoStub) DeleteRPC(context.Context, uuid.UUID) error          { return nil }
func (s *contractAuditChainRepoStub) GetRPCByID(context.Context, uuid.UUID) (*entities.ChainRPC, error) {
	return nil, domainerrors.ErrNotFound
}

type contractAuditContractRepoStub struct {
	contractByID map[uuid.UUID]*entities.SmartContract
	getByIDErr   error
}

func (s *contractAuditContractRepoStub) Create(context.Context, *entities.SmartContract) error {
	return nil
}
func (s *contractAuditContractRepoStub) GetByID(_ context.Context, id uuid.UUID) (*entities.SmartContract, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	if c, ok := s.contractByID[id]; ok {
		return c, nil
	}
	return nil, domainerrors.ErrNotFound
}
func (s *contractAuditContractRepoStub) GetByChainAndAddress(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *contractAuditContractRepoStub) GetActiveContract(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *contractAuditContractRepoStub) GetFiltered(context.Context, *uuid.UUID, entities.SmartContractType, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return []*entities.SmartContract{}, 0, nil
}
func (s *contractAuditContractRepoStub) GetByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *contractAuditContractRepoStub) GetAll(context.Context, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (s *contractAuditContractRepoStub) Update(context.Context, *entities.SmartContract) error {
	return nil
}
func (s *contractAuditContractRepoStub) SoftDelete(context.Context, uuid.UUID) error { return nil }

func TestContractConfigAuditHandler_CheckAndByContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sourceID := uuid.New()
	contractID := uuid.New()
	sourceChain := &entities.Chain{ID: sourceID, ChainID: "8453", Name: "Base", Type: entities.ChainTypeSVM, IsActive: true}

	okUsecase := usecases.NewContractConfigAuditUsecase(
		&contractAuditChainRepoStub{
			chainsByID: map[uuid.UUID]*entities.Chain{sourceID: sourceChain},
			chainByID: map[string]*entities.Chain{
				"eip155:8453": sourceChain,
				"8453":        sourceChain,
			},
		},
		&contractAuditContractRepoStub{contractByID: map[uuid.UUID]*entities.SmartContract{
			contractID: {ID: contractID, ChainUUID: sourceID, Type: entities.ContractTypeGateway, ContractAddress: "0x1", IsActive: true},
		}},
		nil,
	)

	errUsecase := usecases.NewContractConfigAuditUsecase(
		&contractAuditChainRepoStub{chainByID: map[string]*entities.Chain{}},
		&contractAuditContractRepoStub{getByIDErr: errors.New("db down")},
		nil,
	)

	r := gin.New()
	r.GET("/ok", NewContractConfigAuditHandler(okUsecase).Check)
	r.GET("/err", NewContractConfigAuditHandler(errUsecase).Check)
	r.GET("/by/:id", NewContractConfigAuditHandler(okUsecase).CheckByContract)
	r.GET("/byerr/:id", NewContractConfigAuditHandler(errUsecase).CheckByContract)

	req := httptest.NewRequest(http.MethodGet, "/ok?sourceChainId=eip155:8453", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"result\"")

	req = httptest.NewRequest(http.MethodGet, "/ok", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/err?sourceChainId=eip155:8453", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/by/not-a-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/by/"+contractID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"result\"")

	req = httptest.NewRequest(http.MethodGet, "/byerr/"+contractID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
