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
	"pay-chain.backend/pkg/utils"
)

type paymentBridgeRepoErrStub struct {
	getByIDFn func(context.Context, uuid.UUID) (*entities.PaymentBridge, error)
	listFn    func(context.Context, utils.PaginationParams) ([]*entities.PaymentBridge, int64, error)
	createFn  func(context.Context, *entities.PaymentBridge) error
	updateFn  func(context.Context, *entities.PaymentBridge) error
	deleteFn  func(context.Context, uuid.UUID) error
}

func (s *paymentBridgeRepoErrStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.PaymentBridge, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (s *paymentBridgeRepoErrStub) GetByName(context.Context, string) (*entities.PaymentBridge, error) { return nil, nil }
func (s *paymentBridgeRepoErrStub) List(ctx context.Context, p utils.PaginationParams) ([]*entities.PaymentBridge, int64, error) {
	if s.listFn != nil {
		return s.listFn(ctx, p)
	}
	return nil, 0, nil
}
func (s *paymentBridgeRepoErrStub) Create(ctx context.Context, bridge *entities.PaymentBridge) error {
	if s.createFn != nil {
		return s.createFn(ctx, bridge)
	}
	return nil
}
func (s *paymentBridgeRepoErrStub) Update(ctx context.Context, bridge *entities.PaymentBridge) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, bridge)
	}
	return nil
}
func (s *paymentBridgeRepoErrStub) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

type bridgeConfigRepoErrStub struct {
	getByIDFn func(context.Context, uuid.UUID) (*entities.BridgeConfig, error)
	listFn    func(context.Context, *uuid.UUID, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.BridgeConfig, int64, error)
	createFn  func(context.Context, *entities.BridgeConfig) error
	updateFn  func(context.Context, *entities.BridgeConfig) error
	deleteFn  func(context.Context, uuid.UUID) error
}

func (s *bridgeConfigRepoErrStub) GetActive(context.Context, uuid.UUID, uuid.UUID) (*entities.BridgeConfig, error) {
	return nil, nil
}
func (s *bridgeConfigRepoErrStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.BridgeConfig, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (s *bridgeConfigRepoErrStub) List(ctx context.Context, sourceChainID, destChainID, bridgeID *uuid.UUID, p utils.PaginationParams) ([]*entities.BridgeConfig, int64, error) {
	if s.listFn != nil {
		return s.listFn(ctx, sourceChainID, destChainID, bridgeID, p)
	}
	return nil, 0, nil
}
func (s *bridgeConfigRepoErrStub) Create(ctx context.Context, config *entities.BridgeConfig) error {
	if s.createFn != nil {
		return s.createFn(ctx, config)
	}
	return nil
}
func (s *bridgeConfigRepoErrStub) Update(ctx context.Context, config *entities.BridgeConfig) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, config)
	}
	return nil
}
func (s *bridgeConfigRepoErrStub) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

type feeConfigRepoErrStub struct {
	getByIDFn func(context.Context, uuid.UUID) (*entities.FeeConfig, error)
	listFn    func(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.FeeConfig, int64, error)
	createFn  func(context.Context, *entities.FeeConfig) error
	updateFn  func(context.Context, *entities.FeeConfig) error
	deleteFn  func(context.Context, uuid.UUID) error
}

func (s *feeConfigRepoErrStub) GetByChainAndToken(context.Context, uuid.UUID, uuid.UUID) (*entities.FeeConfig, error) {
	return nil, nil
}
func (s *feeConfigRepoErrStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.FeeConfig, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (s *feeConfigRepoErrStub) List(ctx context.Context, chainID, tokenID *uuid.UUID, p utils.PaginationParams) ([]*entities.FeeConfig, int64, error) {
	if s.listFn != nil {
		return s.listFn(ctx, chainID, tokenID, p)
	}
	return nil, 0, nil
}
func (s *feeConfigRepoErrStub) Create(ctx context.Context, config *entities.FeeConfig) error {
	if s.createFn != nil {
		return s.createFn(ctx, config)
	}
	return nil
}
func (s *feeConfigRepoErrStub) Update(ctx context.Context, config *entities.FeeConfig) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, config)
	}
	return nil
}
func (s *feeConfigRepoErrStub) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

type tokenRepoAlwaysFoundStub struct {
	token *entities.Token
}

func (s tokenRepoAlwaysFoundStub) GetByID(context.Context, uuid.UUID) (*entities.Token, error) {
	return s.token, nil
}
func (s tokenRepoAlwaysFoundStub) GetBySymbol(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, nil
}
func (s tokenRepoAlwaysFoundStub) GetByAddress(context.Context, string, uuid.UUID) (*entities.Token, error) {
	return nil, nil
}
func (s tokenRepoAlwaysFoundStub) GetAll(context.Context) ([]*entities.Token, error) { return nil, nil }
func (s tokenRepoAlwaysFoundStub) GetStablecoins(context.Context) ([]*entities.Token, error) {
	return nil, nil
}
func (s tokenRepoAlwaysFoundStub) GetNative(context.Context, uuid.UUID) (*entities.Token, error) { return nil, nil }
func (s tokenRepoAlwaysFoundStub) GetTokensByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (s tokenRepoAlwaysFoundStub) GetAllTokens(context.Context, *uuid.UUID, *string, utils.PaginationParams) ([]*entities.Token, int64, error) {
	return nil, 0, nil
}
func (s tokenRepoAlwaysFoundStub) Create(context.Context, *entities.Token) error { return nil }
func (s tokenRepoAlwaysFoundStub) Update(context.Context, *entities.Token) error { return nil }
func (s tokenRepoAlwaysFoundStub) SoftDelete(context.Context, uuid.UUID) error   { return nil }

func TestPaymentConfigHandler_PaymentBridgeErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bridgeID := uuid.New()
	h := NewPaymentConfigHandler(
		&paymentBridgeRepoErrStub{
			getByIDFn: func(context.Context, uuid.UUID) (*entities.PaymentBridge, error) {
				return &entities.PaymentBridge{ID: bridgeID, Name: "Old"}, nil
			},
			listFn: func(context.Context, utils.PaginationParams) ([]*entities.PaymentBridge, int64, error) {
				return nil, 0, errors.New("list failed")
			},
			createFn: func(context.Context, *entities.PaymentBridge) error { return errors.New("create failed") },
			updateFn: func(context.Context, *entities.PaymentBridge) error { return errors.New("update failed") },
			deleteFn: func(context.Context, uuid.UUID) error { return errors.New("delete failed") },
		},
		nil, nil, nil, nil,
	)

	r := gin.New()
	r.GET("/payment-bridges", h.ListPaymentBridges)
	r.POST("/payment-bridges", h.CreatePaymentBridge)
	r.PUT("/payment-bridges/:id", h.UpdatePaymentBridge)
	r.DELETE("/payment-bridges/:id", h.DeletePaymentBridge)

	req := httptest.NewRequest(http.MethodGet, "/payment-bridges", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/payment-bridges", bytes.NewBufferString(`{"name":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/payment-bridges", bytes.NewBufferString(`{"name":"CCIP"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/payment-bridges/"+bridgeID.String(), bytes.NewBufferString(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/payment-bridges/"+bridgeID.String(), bytes.NewBufferString(`{"name":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/payment-bridges/"+bridgeID.String(), bytes.NewBufferString(`{"name":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/payment-bridges/"+bridgeID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPaymentConfigHandler_BridgeAndFeeErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sourceChainID := uuid.New()
	destChainID := uuid.New()
	bridgeID := uuid.New()
	bridgeConfigID := uuid.New()
	feeConfigID := uuid.New()
	tokenID := uuid.New()

	h := NewPaymentConfigHandler(
		nil,
		&bridgeConfigRepoErrStub{
			getByIDFn: func(context.Context, uuid.UUID) (*entities.BridgeConfig, error) {
				return &entities.BridgeConfig{ID: bridgeConfigID}, nil
			},
			listFn: func(context.Context, *uuid.UUID, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.BridgeConfig, int64, error) {
				return nil, 0, errors.New("bridge list failed")
			},
			createFn: func(context.Context, *entities.BridgeConfig) error { return errors.New("bridge create failed") },
			updateFn: func(context.Context, *entities.BridgeConfig) error { return errors.New("bridge update failed") },
			deleteFn: func(context.Context, uuid.UUID) error { return errors.New("bridge delete failed") },
		},
		&feeConfigRepoErrStub{
			getByIDFn: func(context.Context, uuid.UUID) (*entities.FeeConfig, error) {
				return &entities.FeeConfig{ID: feeConfigID}, nil
			},
			listFn: func(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.FeeConfig, int64, error) {
				return nil, 0, errors.New("fee list failed")
			},
			createFn: func(context.Context, *entities.FeeConfig) error { return errors.New("fee create failed") },
			updateFn: func(context.Context, *entities.FeeConfig) error { return errors.New("fee update failed") },
			deleteFn: func(context.Context, uuid.UUID) error { return errors.New("fee delete failed") },
		},
		nil,
		tokenRepoAlwaysFoundStub{token: &entities.Token{ID: tokenID}},
	)

	r := gin.New()
	r.GET("/bridge-configs", h.ListBridgeConfigs)
	r.POST("/bridge-configs", h.CreateBridgeConfig)
	r.PUT("/bridge-configs/:id", h.UpdateBridgeConfig)
	r.DELETE("/bridge-configs/:id", h.DeleteBridgeConfig)
	r.GET("/fee-configs", h.ListFeeConfigs)
	r.POST("/fee-configs", h.CreateFeeConfig)
	r.PUT("/fee-configs/:id", h.UpdateFeeConfig)
	r.DELETE("/fee-configs/:id", h.DeleteFeeConfig)

	req := httptest.NewRequest(http.MethodGet, "/bridge-configs?sourceChainId="+sourceChainID.String()+"&destChainId="+destChainID.String()+"&bridgeId="+bridgeID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/bridge-configs", bytes.NewBufferString(`{"bridgeId":"`+bridgeID.String()+`","sourceChainId":"`+sourceChainID.String()+`","destChainId":"`+destChainID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/bridge-configs/"+bridgeConfigID.String(), bytes.NewBufferString(`{"bridgeId":"`+bridgeID.String()+`","sourceChainId":"`+sourceChainID.String()+`","destChainId":"`+destChainID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/bridge-configs/"+bridgeConfigID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/fee-configs?chainId="+sourceChainID.String()+"&tokenId="+tokenID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/fee-configs", bytes.NewBufferString(`{"chainId":"`+sourceChainID.String()+`","tokenId":"`+tokenID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/fee-configs/"+feeConfigID.String(), bytes.NewBufferString(`{"chainId":"`+sourceChainID.String()+`","tokenId":"`+tokenID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/fee-configs/"+feeConfigID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
