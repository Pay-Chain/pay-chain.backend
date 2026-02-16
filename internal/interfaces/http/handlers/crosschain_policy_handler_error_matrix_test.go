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
	"pay-chain.backend/pkg/utils"
)

type routePolicyRepoErrMatrixStub struct {
	item      *entities.RoutePolicy
	getByIDFn func(context.Context, uuid.UUID) (*entities.RoutePolicy, error)
	listFn    func(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.RoutePolicy, int64, error)
	updateFn  func(context.Context, *entities.RoutePolicy) error
	deleteFn  func(context.Context, uuid.UUID) error
}

func (s *routePolicyRepoErrMatrixStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.RoutePolicy, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	if s.item == nil {
		s.item = &entities.RoutePolicy{ID: id}
	}
	return s.item, nil
}
func (s *routePolicyRepoErrMatrixStub) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *routePolicyRepoErrMatrixStub) List(ctx context.Context, sourceChainID, destChainID *uuid.UUID, p utils.PaginationParams) ([]*entities.RoutePolicy, int64, error) {
	if s.listFn != nil {
		return s.listFn(ctx, sourceChainID, destChainID, p)
	}
	return []*entities.RoutePolicy{}, 0, nil
}
func (s *routePolicyRepoErrMatrixStub) Create(context.Context, *entities.RoutePolicy) error {
	return nil
}
func (s *routePolicyRepoErrMatrixStub) Update(ctx context.Context, policy *entities.RoutePolicy) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, policy)
	}
	s.item = policy
	return nil
}
func (s *routePolicyRepoErrMatrixStub) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

type layerZeroRepoErrMatrixStub struct {
	item      *entities.LayerZeroConfig
	getByIDFn func(context.Context, uuid.UUID) (*entities.LayerZeroConfig, error)
	listFn    func(context.Context, *uuid.UUID, *uuid.UUID, *bool, utils.PaginationParams) ([]*entities.LayerZeroConfig, int64, error)
	createFn  func(context.Context, *entities.LayerZeroConfig) error
	updateFn  func(context.Context, *entities.LayerZeroConfig) error
	deleteFn  func(context.Context, uuid.UUID) error
}

func (s *layerZeroRepoErrMatrixStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.LayerZeroConfig, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	if s.item == nil {
		s.item = &entities.LayerZeroConfig{ID: id}
	}
	return s.item, nil
}
func (s *layerZeroRepoErrMatrixStub) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.LayerZeroConfig, error) {
	return nil, domainerrors.ErrNotFound
}
func (s *layerZeroRepoErrMatrixStub) List(ctx context.Context, sourceChainID, destChainID *uuid.UUID, activeOnly *bool, p utils.PaginationParams) ([]*entities.LayerZeroConfig, int64, error) {
	if s.listFn != nil {
		return s.listFn(ctx, sourceChainID, destChainID, activeOnly, p)
	}
	return []*entities.LayerZeroConfig{}, 0, nil
}
func (s *layerZeroRepoErrMatrixStub) Create(ctx context.Context, config *entities.LayerZeroConfig) error {
	if s.createFn != nil {
		return s.createFn(ctx, config)
	}
	s.item = config
	return nil
}
func (s *layerZeroRepoErrMatrixStub) Update(ctx context.Context, config *entities.LayerZeroConfig) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, config)
	}
	s.item = config
	return nil
}
func (s *layerZeroRepoErrMatrixStub) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

func TestCrosschainPolicyHandler_ErrorMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sourceID := uuid.New()
	destID := uuid.New()
	routeID := uuid.New()
	lzID := uuid.New()
	peerHex := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	chainRepo := &crosschainChainRepoStub{
		getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
			switch chainID {
			case "8453":
				return &entities.Chain{ID: sourceID}, nil
			case "42161":
				return &entities.Chain{ID: destID}, nil
			default:
				return nil, domainerrors.ErrNotFound
			}
		},
		getByCAIP2: func(_ context.Context, caip2 string) (*entities.Chain, error) {
			switch caip2 {
			case "eip155:8453":
				return &entities.Chain{ID: sourceID}, nil
			case "eip155:42161":
				return &entities.Chain{ID: destID}, nil
			default:
				return nil, domainerrors.ErrNotFound
			}
		},
	}

	routeRepo := &routePolicyRepoErrMatrixStub{item: &entities.RoutePolicy{ID: routeID}}
	lzRepo := &layerZeroRepoErrMatrixStub{item: &entities.LayerZeroConfig{ID: lzID}}
	h := NewCrosschainPolicyHandler(routeRepo, lzRepo, chainRepo)

	r := gin.New()
	r.GET("/route", h.ListRoutePolicies)
	r.PUT("/route/:id", h.UpdateRoutePolicy)
	r.DELETE("/route/:id", h.DeleteRoutePolicy)
	r.GET("/lz", h.ListLayerZeroConfigs)
	r.POST("/lz", h.CreateLayerZeroConfig)
	r.PUT("/lz/:id", h.UpdateLayerZeroConfig)
	r.DELETE("/lz/:id", h.DeleteLayerZeroConfig)

	routeRepo.listFn = func(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.RoutePolicy, int64, error) {
		return nil, 0, errors.New("list failed")
	}
	req := httptest.NewRequest(http.MethodGet, "/route?sourceChainId=eip155:8453", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	routeRepo.listFn = nil

	updateRouteBody := `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + sourceID.String() + `","defaultBridgeType":0}`
	req = httptest.NewRequest(http.MethodPut, "/route/"+routeID.String(), strings.NewReader(updateRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	updateRouteBody = `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","defaultBridgeType":9}`
	req = httptest.NewRequest(http.MethodPut, "/route/"+routeID.String(), strings.NewReader(updateRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	updateRouteBody = `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","defaultBridgeType":1,"fallbackMode":"bad-mode"}`
	req = httptest.NewRequest(http.MethodPut, "/route/"+routeID.String(), strings.NewReader(updateRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	updateRouteBody = `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","defaultBridgeType":1,"fallbackOrder":[1,1]}`
	req = httptest.NewRequest(http.MethodPut, "/route/"+routeID.String(), strings.NewReader(updateRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	routeRepo.getByIDFn = func(context.Context, uuid.UUID) (*entities.RoutePolicy, error) { return nil, errors.New("get failed") }
	updateRouteBody = `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","defaultBridgeType":1}`
	req = httptest.NewRequest(http.MethodPut, "/route/"+routeID.String(), strings.NewReader(updateRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	routeRepo.getByIDFn = nil

	routeRepo.updateFn = func(context.Context, *entities.RoutePolicy) error { return errors.New("update failed") }
	req = httptest.NewRequest(http.MethodPut, "/route/"+routeID.String(), strings.NewReader(updateRouteBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	routeRepo.updateFn = nil

	routeRepo.deleteFn = func(context.Context, uuid.UUID) error { return errors.New("delete failed") }
	req = httptest.NewRequest(http.MethodDelete, "/route/"+routeID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	routeRepo.deleteFn = nil

	lzRepo.listFn = func(context.Context, *uuid.UUID, *uuid.UUID, *bool, utils.PaginationParams) ([]*entities.LayerZeroConfig, int64, error) {
		return nil, 0, errors.New("lz list failed")
	}
	req = httptest.NewRequest(http.MethodGet, "/lz?sourceChainId=8453&destChainId=eip155:42161&activeOnly=no", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	lzRepo.listFn = nil

	createLZBody := `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + sourceID.String() + `","dstEid":1,"peerHex":"` + peerHex + `"}`
	req = httptest.NewRequest(http.MethodPost, "/lz", strings.NewReader(createLZBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	createLZBody = `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","dstEid":1,"peerHex":"0x1234"}`
	req = httptest.NewRequest(http.MethodPost, "/lz", strings.NewReader(createLZBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	lzRepo.createFn = func(context.Context, *entities.LayerZeroConfig) error { return errors.New("create failed") }
	createLZBody = `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","dstEid":1,"peerHex":"` + peerHex + `","optionsHex":"01"}`
	req = httptest.NewRequest(http.MethodPost, "/lz", strings.NewReader(createLZBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	lzRepo.createFn = nil

	lzRepo.getByIDFn = func(context.Context, uuid.UUID) (*entities.LayerZeroConfig, error) {
		return nil, errors.New("get failed")
	}
	updateLZBody := `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","dstEid":1,"peerHex":"` + peerHex + `"}`
	req = httptest.NewRequest(http.MethodPut, "/lz/"+lzID.String(), strings.NewReader(updateLZBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	lzRepo.getByIDFn = nil

	updateLZBody = `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","dstEid":1,"peerHex":"0x12"}`
	req = httptest.NewRequest(http.MethodPut, "/lz/"+lzID.String(), strings.NewReader(updateLZBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	lzRepo.updateFn = func(context.Context, *entities.LayerZeroConfig) error { return errors.New("update failed") }
	updateLZBody = `{"sourceChainId":"` + sourceID.String() + `","destChainId":"` + destID.String() + `","dstEid":1,"peerHex":"` + peerHex + `"}`
	req = httptest.NewRequest(http.MethodPut, "/lz/"+lzID.String(), strings.NewReader(updateLZBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	lzRepo.updateFn = nil

	lzRepo.deleteFn = func(context.Context, uuid.UUID) error { return errors.New("delete failed") }
	req = httptest.NewRequest(http.MethodDelete, "/lz/"+lzID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
