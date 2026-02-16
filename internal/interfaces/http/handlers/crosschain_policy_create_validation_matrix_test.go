package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

type routePolicyRepoCreateNoop struct{}

func (routePolicyRepoCreateNoop) GetByID(context.Context, uuid.UUID) (*entities.RoutePolicy, error) {
	return nil, nil
}
func (routePolicyRepoCreateNoop) GetByRoute(context.Context, uuid.UUID, uuid.UUID) (*entities.RoutePolicy, error) {
	return nil, nil
}
func (routePolicyRepoCreateNoop) List(context.Context, *uuid.UUID, *uuid.UUID, utils.PaginationParams) ([]*entities.RoutePolicy, int64, error) {
	return nil, 0, nil
}
func (routePolicyRepoCreateNoop) Create(context.Context, *entities.RoutePolicy) error { return nil }
func (routePolicyRepoCreateNoop) Update(context.Context, *entities.RoutePolicy) error { return nil }
func (routePolicyRepoCreateNoop) Delete(context.Context, uuid.UUID) error             { return nil }

func TestCrosschainPolicyHandler_CreateRoutePolicy_ValidationMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sameID := utils.GenerateUUIDv7()
	otherID := utils.GenerateUUIDv7()
	chainRepo := &crosschainChainRepoStub{
		getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
			if chainID == "8453" {
				return &entities.Chain{ID: sameID}, nil
			}
			if chainID == "42161" {
				return &entities.Chain{ID: otherID}, nil
			}
			return nil, nil
		},
		getByCAIP2: func(_ context.Context, caip2 string) (*entities.Chain, error) {
			if caip2 == "eip155:8453" {
				return &entities.Chain{ID: sameID}, nil
			}
			if caip2 == "eip155:42161" {
				return &entities.Chain{ID: otherID}, nil
			}
			return nil, nil
		},
	}

	h := NewCrosschainPolicyHandler(routePolicyRepoCreateNoop{}, layerZeroRepoNoop{}, chainRepo)
	r := gin.New()
	r.POST("/route", h.CreateRoutePolicy)

	cases := []string{
		// missing required defaultBridgeType
		`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161"}`,
		// source == dest
		`{"sourceChainId":"eip155:8453","destChainId":"eip155:8453","defaultBridgeType":0}`,
		// invalid bridge type
		`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161","defaultBridgeType":9}`,
		// invalid fallback mode
		`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161","defaultBridgeType":0,"fallbackMode":"unknown"}`,
		// duplicate fallback order
		`{"sourceChainId":"eip155:8453","destChainId":"eip155:42161","defaultBridgeType":0,"fallbackMode":"strict","fallbackOrder":[0,0]}`,
	}

	for _, body := range cases {
		req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
		}
	}
}
