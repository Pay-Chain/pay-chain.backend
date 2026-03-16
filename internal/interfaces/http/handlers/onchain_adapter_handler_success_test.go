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
	"payment-kita.backend/internal/usecases"
)

type onchainAdapterServiceStub struct {
	getStatus        func(context.Context, string, string) (*usecases.OnchainAdapterStatus, error)
	registerAdapter  func(context.Context, string, string, uint8, string) (string, error)
	setDefaultBridge func(context.Context, string, string, uint8) (string, error)
	setHyperbridge   func(context.Context, string, string, string, string) (string, []string, error)
	setHyperbridgeTokenGateway func(context.Context, usecases.HyperbridgeTokenGatewayConfigInput) (string, []string, error)
	setCCIP          func(context.Context, usecases.CCIPConfigInput) (string, []string, error)
	setStargate     func(context.Context, string, string, *uint32, string, string) (string, []string, error)
	configureLZE2E   func(context.Context, usecases.StargateE2EConfigureInput) (*usecases.StargateE2EConfigureResult, error)
	getLZE2EStatus   func(context.Context, usecases.StargateE2EStatusInput) (*usecases.StargateE2EStatusResult, error)
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
func (s onchainAdapterServiceStub) SetHyperbridgeTokenGatewayConfig(ctx context.Context, input usecases.HyperbridgeTokenGatewayConfigInput) (string, []string, error) {
	if s.setHyperbridgeTokenGateway == nil {
		return "0xhbtoken", []string{"0x5"}, nil
	}
	return s.setHyperbridgeTokenGateway(ctx, input)
}
func (s onchainAdapterServiceStub) SetCCIPConfig(ctx context.Context, input usecases.CCIPConfigInput) (string, []string, error) {
	return s.setCCIP(ctx, input)
}
func (s onchainAdapterServiceStub) SetStargateConfig(ctx context.Context, sourceChainInput, destChainInput string, dstEid *uint32, peerHex, optionsHex string) (string, []string, error) {
	return s.setStargate(ctx, sourceChainInput, destChainInput, dstEid, peerHex, optionsHex)
}
func (s onchainAdapterServiceStub) ConfigureStargateE2E(ctx context.Context, input usecases.StargateE2EConfigureInput) (*usecases.StargateE2EConfigureResult, error) {
	if s.configureLZE2E == nil {
		return &usecases.StargateE2EConfigureResult{Status: "SUCCESS"}, nil
	}
	return s.configureLZE2E(ctx, input)
}
func (s onchainAdapterServiceStub) GetStargateE2EStatus(ctx context.Context, input usecases.StargateE2EStatusInput) (*usecases.StargateE2EStatusResult, error) {
	if s.getLZE2EStatus == nil {
		return &usecases.StargateE2EStatusResult{Ready: true}, nil
	}
	return s.getLZE2EStatus(ctx, input)
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
			setCCIP: func(_ context.Context, _ usecases.CCIPConfigInput) (string, []string, error) {
				return "0xccip", []string{"0x3"}, nil
			},
			setStargate: func(_ context.Context, _, _ string, _ *uint32, _, _ string) (string, []string, error) {
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
	r.POST("/hyperbridge-token-gateway", h.SetHyperbridgeTokenGatewayConfig)
	r.POST("/ccip", h.SetCCIPConfig)
	r.POST("/stargate", h.SetStargateConfig)
	r.POST("/stargate-e2e", h.ConfigureStargateE2E)
	r.GET("/stargate-e2e-status", h.GetStargateE2EStatus)

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
			path: "/hyperbridge-token-gateway",
			body: map[string]interface{}{
				"sourceChainId":          "eip155:8453",
				"destChainId":            "eip155:42161",
				"stateMachineIdHex":      "0x45564d2d3432313631",
				"settlementExecutorAddress": "0x1111111111111111111111111111111111111111",
				"nativeCost":             "100000000000000",
				"relayerFee":             "0",
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
			path: "/stargate",
			body: map[string]interface{}{
				"sourceChainId": "eip155:8453",
				"destChainId":   "eip155:42161",
				"dstEid":        30110,
				"peerHex":       "0x0000000000000000000000001111111111111111111111111111111111111111",
				"optionsHex":    "0x010203",
			},
		},
		{
			path: "/stargate-e2e",
			body: map[string]interface{}{
				"sourceChainId": "eip155:8453",
				"destChainId":   "eip155:42161",
				"source": map[string]interface{}{
					"registerAdapterIfMissing": true,
					"setDefaultBridgeType":     true,
					"senderAddress":            "0x1111111111111111111111111111111111111111",
					"dstEid":                   30110,
					"dstPeerHex":               "0x0000000000000000000000002222222222222222222222222222222222222222",
				},
				"destination": map[string]interface{}{
					"receiverAddress": "0x3333333333333333333333333333333333333333",
					"srcEid":          30184,
					"srcSenderHex":    "0x0000000000000000000000001111111111111111111111111111111111111111",
				},
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

	req = httptest.NewRequest(
		http.MethodGet,
		"/stargate-e2e-status?sourceChainId=eip155:8453&destChainId=eip155:42161&destinationSrcEid=30184&destinationSrcSenderHex=0x0000000000000000000000001111111111111111111111111111111111111111",
		nil,
	)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
