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
	"pay-chain.backend/pkg/utils"
)

func TestPaymentConfigHandler_UpdatePaymentBridge_GetByIDAndBindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bridgeID := uuid.New()

	t.Run("get by id failed", func(t *testing.T) {
		h := NewPaymentConfigHandler(
			&paymentBridgeRepoErrStub{
				getByIDFn: func(context.Context, uuid.UUID) (*entities.PaymentBridge, error) {
					return nil, errors.New("get failed")
				},
			},
			nil, nil, nil, nil,
		)

		r := gin.New()
		r.PUT("/payment-bridges/:id", h.UpdatePaymentBridge)
		req := httptest.NewRequest(http.MethodPut, "/payment-bridges/"+bridgeID.String(), bytes.NewBufferString(`{"name":"Bridge X"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("bind json failed", func(t *testing.T) {
		h := NewPaymentConfigHandler(
			&paymentBridgeRepoErrStub{
				getByIDFn: func(context.Context, uuid.UUID) (*entities.PaymentBridge, error) {
					return &entities.PaymentBridge{ID: bridgeID, Name: "Old"}, nil
				},
			},
			nil, nil, nil, nil,
		)

		r := gin.New()
		r.PUT("/payment-bridges/:id", h.UpdatePaymentBridge)
		req := httptest.NewRequest(http.MethodPut, "/payment-bridges/"+bridgeID.String(), bytes.NewBufferString(`{"name":`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPaymentConfigHandler_ListAndCreateFeeConfig_InvalidTokenIDBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chainUUID := uuid.New()

	h := NewPaymentConfigHandler(
		nil,
		nil,
		&feeConfigRepoErrStub{
			listFn: func(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.FeeConfig, int64, error) {
				return []*entities.FeeConfig{}, 0, nil
			},
		},
		&crosschainChainRepoStub{
			getByChainID: func(_ context.Context, chainIDInput string) (*entities.Chain, error) {
				if chainIDInput == "eip155:8453" {
					return &entities.Chain{ID: chainUUID}, nil
				}
				return nil, domainerrors.ErrNotFound
			},
			getByCAIP2: func(context.Context, string) (*entities.Chain, error) { return nil, domainerrors.ErrNotFound },
		},
		tokenRepoExistsStub{existing: map[uuid.UUID]*entities.Token{}},
	)

	r := gin.New()
	r.GET("/fee-configs", h.ListFeeConfigs)
	r.POST("/fee-configs", h.CreateFeeConfig)

	// invalid tokenId query branch in list
	req := httptest.NewRequest(http.MethodGet, "/fee-configs?chainId=eip155:8453&tokenId=bad-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// invalid tokenId body branch in create
	req = httptest.NewRequest(http.MethodPost, "/fee-configs", bytes.NewBufferString(`{"chainId":"eip155:8453","tokenId":"bad-uuid"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
