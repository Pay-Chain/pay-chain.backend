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
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/utils"
)

type onchainHandlerChainRepoStub struct{}

func (onchainHandlerChainRepoStub) GetByChainID(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (onchainHandlerChainRepoStub) GetByCAIP2(context.Context, string) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (onchainHandlerChainRepoStub) Create(context.Context, *entities.Chain) error { return nil }
func (onchainHandlerChainRepoStub) Update(context.Context, *entities.Chain) error { return nil }
func (onchainHandlerChainRepoStub) Delete(context.Context, uuid.UUID) error       { return nil }
func (onchainHandlerChainRepoStub) List(context.Context) ([]*entities.Chain, error) {
	return nil, nil
}
func (onchainHandlerChainRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Chain, error) {
	return nil, domainerrors.ErrNotFound
}
func (onchainHandlerChainRepoStub) GetActive(context.Context, utils.PaginationParams) ([]*entities.Chain, int64, error) {
	return nil, 0, nil
}
func (onchainHandlerChainRepoStub) GetAll(context.Context) ([]*entities.Chain, error) {
	return nil, nil
}
func (onchainHandlerChainRepoStub) GetAllRPCs(context.Context, *uuid.UUID, *bool, *string, utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	return nil, 0, nil
}

type onchainHandlerContractRepoStub struct{}

func (onchainHandlerContractRepoStub) Create(context.Context, *entities.SmartContract) error {
	return nil
}
func (onchainHandlerContractRepoStub) GetByID(context.Context, uuid.UUID) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (onchainHandlerContractRepoStub) GetByChainAndAddress(context.Context, uuid.UUID, string) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (onchainHandlerContractRepoStub) GetActiveContract(context.Context, uuid.UUID, entities.SmartContractType) (*entities.SmartContract, error) {
	return nil, domainerrors.ErrNotFound
}
func (onchainHandlerContractRepoStub) GetFiltered(context.Context, *uuid.UUID, entities.SmartContractType, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (onchainHandlerContractRepoStub) GetByChain(context.Context, uuid.UUID, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (onchainHandlerContractRepoStub) GetAll(context.Context, utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	return nil, 0, nil
}
func (onchainHandlerContractRepoStub) Update(context.Context, *entities.SmartContract) error {
	return nil
}
func (onchainHandlerContractRepoStub) SoftDelete(context.Context, uuid.UUID) error { return nil }

func TestOnchainAdapterHandler_ErrorPathsAfterValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := usecases.NewOnchainAdapterUsecase(onchainHandlerChainRepoStub{}, onchainHandlerContractRepoStub{}, nil, "")
	h := NewOnchainAdapterHandler(usecase)

	r := gin.New()
	r.GET("/status", h.GetStatus)
	r.POST("/register", h.RegisterAdapter)
	r.POST("/default-bridge", h.SetDefaultBridgeType)
	r.POST("/hyperbridge", h.SetHyperbridgeConfig)
	r.POST("/ccip", h.SetCCIPConfig)
	r.POST("/layerzero", h.SetLayerZeroConfig)

	req := httptest.NewRequest(http.MethodGet, "/status?sourceChainId=eip155:8453&destChainId=eip155:42161", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	tests := []struct {
		path string
		body map[string]interface{}
	}{
		{
			path: "/register",
			body: map[string]interface{}{
				"sourceChainId":  "eip155:8453",
				"destChainId":    "eip155:42161",
				"bridgeType":     0,
				"adapterAddress": "0x1111111111111111111111111111111111111111",
			},
		},
		{
			path: "/default-bridge",
			body: map[string]interface{}{
				"sourceChainId": "eip155:8453",
				"destChainId":   "eip155:42161",
				"bridgeType":    1,
			},
		},
		{
			path: "/hyperbridge",
			body: map[string]interface{}{
				"sourceChainId":          "eip155:8453",
				"destChainId":            "eip155:42161",
				"stateMachineIdHex":      "0x45564d2d3432313631",
				"destinationContractHex": "0x0000000000000000000000001111111111111111111111111111111111111111",
			},
		},
		{
			path: "/ccip",
			body: map[string]interface{}{
				"sourceChainId":         "eip155:8453",
				"destChainId":           "eip155:42161",
				"chainSelector":         4949039107694359620,
				"destinationAdapterHex": "0x0000000000000000000000001111111111111111111111111111111111111111",
			},
		},
		{
			path: "/layerzero",
			body: map[string]interface{}{
				"sourceChainId": "eip155:8453",
				"destChainId":   "eip155:42161",
				"dstEid":        30110,
				"peerHex":       "0x0000000000000000000000001111111111111111111111111111111111111111",
				"optionsHex":    "0x010203",
			},
		},
	}

	for _, tc := range tests {
		payload, _ := json.Marshal(tc.body)
		req = httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, "path=%s body=%s", tc.path, rec.Body.String())
	}
}
