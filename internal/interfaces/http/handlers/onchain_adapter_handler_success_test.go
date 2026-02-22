package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/usecases"
)

type onchainAdapterServiceStub struct {
	getStatus        func(context.Context, string, string) (*usecases.OnchainAdapterStatus, error)
	registerAdapter  func(context.Context, string, string, uint8, string) (string, error)
	setDefaultBridge func(context.Context, string, string, uint8) (string, error)
	setHyperbridge   func(context.Context, string, string, string, string) (string, []string, error)
	setCCIP          func(context.Context, string, string, *uint64, string) (string, []string, error)
	setLayerZero     func(context.Context, string, string, *uint32, string, string) (string, []string, error)
	genericInteract  func(context.Context, string, string, string, string, []interface{}) (interface{}, bool, error)
}

func (s onchainAdapterServiceStub) GetStatus(ctx context.Context, sourceChainInput, destChainInput string) (*usecases.OnchainAdapterStatus, error) {
	return s.getStatus(ctx, sourceChainInput, destChainInput)
}
func (s onchainAdapterServiceStub) RegisterAdapter(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8, adapterAddress string) (string, error) {
	return s.registerAdapter(ctx, sourceChainInput, destChainInput, bridgeType, adapterAddress)
}
func (s onchainAdapterServiceStub) SetDefaultBridgeType(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8) (string, error) {
	return s.setDefaultBridge(ctx, sourceChainInput, destChainInput, bridgeType)
}
func (s onchainAdapterServiceStub) SetHyperbridgeConfig(ctx context.Context, sourceChainInput, destChainInput, stateMachineIDHex, destinationContractHex string) (string, []string, error) {
	return s.setHyperbridge(ctx, sourceChainInput, destChainInput, stateMachineIDHex, destinationContractHex)
}
func (s onchainAdapterServiceStub) SetCCIPConfig(ctx context.Context, sourceChainInput, destChainInput string, chainSelector *uint64, destinationAdapterHex string) (string, []string, error) {
	return s.setCCIP(ctx, sourceChainInput, destChainInput, chainSelector, destinationAdapterHex)
}
func (s onchainAdapterServiceStub) SetLayerZeroConfig(ctx context.Context, sourceChainInput, destChainInput string, dstEid *uint32, peerHex, optionsHex string) (string, []string, error) {
	return s.setLayerZero(ctx, sourceChainInput, destChainInput, dstEid, peerHex, optionsHex)
}
func (s onchainAdapterServiceStub) GenericInteract(ctx context.Context, sourceChainInput, contractAddress, method, abiStr string, args []interface{}) (interface{}, bool, error) {
	return s.genericInteract(ctx, sourceChainInput, contractAddress, method, abiStr, args)
}

func TestOnchainAdapterHandler_SuccessPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &OnchainAdapterHandler{
		usecase: onchainAdapterServiceStub{
			getStatus: func(_ context.Context, _, _ string) (*usecases.OnchainAdapterStatus, error) {
				return &usecases.OnchainAdapterStatus{DefaultBridgeType: 1}, nil
			},
			registerAdapter: func(_ context.Context, _, _ string, _ uint8, _ string) (string, error) {
				return "0xregister", nil
			},
			setDefaultBridge: func(_ context.Context, _, _ string, _ uint8) (string, error) {
				return "0xdefault", nil
			},
			setHyperbridge: func(_ context.Context, _, _ string, _, _ string) (string, []string, error) {
				return "0xhyper", []string{"0x1", "0x2"}, nil
			},
			setCCIP: func(_ context.Context, _, _ string, _ *uint64, _ string) (string, []string, error) {
				return "0xccip", []string{"0x3"}, nil
			},
			setLayerZero: func(_ context.Context, _, _ string, _ *uint32, _, _ string) (string, []string, error) {
				return "0xlz", []string{"0x4"}, nil
			},
			genericInteract: func(_ context.Context, _, _, _, _ string, _ []interface{}) (interface{}, bool, error) {
				return "0xresult", false, nil
			},
		},
	}

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
	require.Equal(t, http.StatusOK, rec.Code)

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
		require.Equal(t, http.StatusOK, rec.Code, "path=%s body=%s", tc.path, rec.Body.String())
	}
}
