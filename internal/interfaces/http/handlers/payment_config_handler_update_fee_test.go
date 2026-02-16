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
)

func TestPaymentConfigHandler_UpdateFeeConfig_ErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	feeID := uuid.New()
	chainID := uuid.New()
	tokenID := uuid.New()

	baseChainRepo := &crosschainChainRepoStub{
		getByChainID: func(_ context.Context, chainIDInput string) (*entities.Chain, error) {
			if chainIDInput == "eip155:8453" {
				return &entities.Chain{ID: chainID}, nil
			}
			return nil, domainerrors.ErrNotFound
		},
		getByCAIP2: func(context.Context, string) (*entities.Chain, error) { return nil, domainerrors.ErrNotFound },
	}

	t.Run("get existing failed", func(t *testing.T) {
		h := NewPaymentConfigHandler(
			nil,
			nil,
			&feeConfigRepoErrStub{
				getByIDFn: func(context.Context, uuid.UUID) (*entities.FeeConfig, error) { return nil, errors.New("get failed") },
			},
			baseChainRepo,
			tokenRepoExistsStub{existing: map[uuid.UUID]*entities.Token{tokenID: {ID: tokenID}}},
		)

		r := gin.New()
		r.PUT("/fee-configs/:id", h.UpdateFeeConfig)

		req := httptest.NewRequest(http.MethodPut, "/fee-configs/"+feeID.String(), bytes.NewBufferString(`{"chainId":"eip155:8453","tokenId":"`+tokenID.String()+`"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("bind json failed", func(t *testing.T) {
		h := NewPaymentConfigHandler(
			nil,
			nil,
			&feeConfigRepoErrStub{
				getByIDFn: func(context.Context, uuid.UUID) (*entities.FeeConfig, error) { return &entities.FeeConfig{ID: feeID}, nil },
			},
			baseChainRepo,
			tokenRepoExistsStub{existing: map[uuid.UUID]*entities.Token{tokenID: {ID: tokenID}}},
		)

		r := gin.New()
		r.PUT("/fee-configs/:id", h.UpdateFeeConfig)

		req := httptest.NewRequest(http.MethodPut, "/fee-configs/"+feeID.String(), bytes.NewBufferString(`{"chainId":`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid chain id", func(t *testing.T) {
		h := NewPaymentConfigHandler(
			nil,
			nil,
			&feeConfigRepoErrStub{
				getByIDFn: func(context.Context, uuid.UUID) (*entities.FeeConfig, error) { return &entities.FeeConfig{ID: feeID}, nil },
			},
			baseChainRepo,
			tokenRepoExistsStub{existing: map[uuid.UUID]*entities.Token{tokenID: {ID: tokenID}}},
		)

		r := gin.New()
		r.PUT("/fee-configs/:id", h.UpdateFeeConfig)

		req := httptest.NewRequest(http.MethodPut, "/fee-configs/"+feeID.String(), bytes.NewBufferString(`{"chainId":"unknown-chain","tokenId":"`+tokenID.String()+`"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid token id format", func(t *testing.T) {
		h := NewPaymentConfigHandler(
			nil,
			nil,
			&feeConfigRepoErrStub{
				getByIDFn: func(context.Context, uuid.UUID) (*entities.FeeConfig, error) { return &entities.FeeConfig{ID: feeID}, nil },
			},
			baseChainRepo,
			tokenRepoExistsStub{existing: map[uuid.UUID]*entities.Token{tokenID: {ID: tokenID}}},
		)

		r := gin.New()
		r.PUT("/fee-configs/:id", h.UpdateFeeConfig)

		req := httptest.NewRequest(http.MethodPut, "/fee-configs/"+feeID.String(), bytes.NewBufferString(`{"chainId":"eip155:8453","tokenId":"bad-uuid"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("token not found", func(t *testing.T) {
		h := NewPaymentConfigHandler(
			nil,
			nil,
			&feeConfigRepoErrStub{
				getByIDFn: func(context.Context, uuid.UUID) (*entities.FeeConfig, error) { return &entities.FeeConfig{ID: feeID}, nil },
			},
			baseChainRepo,
			tokenRepoExistsStub{existing: map[uuid.UUID]*entities.Token{}},
		)

		r := gin.New()
		r.PUT("/fee-configs/:id", h.UpdateFeeConfig)

		req := httptest.NewRequest(http.MethodPut, "/fee-configs/"+feeID.String(), bytes.NewBufferString(`{"chainId":"eip155:8453","tokenId":"`+tokenID.String()+`"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

