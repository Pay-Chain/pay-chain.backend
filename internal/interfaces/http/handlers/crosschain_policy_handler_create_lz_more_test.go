package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestCrosschainPolicyHandler_CreateLayerZeroConfig_NormalizationDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sourceID := uuid.New()
	destID := uuid.New()

	chainRepo := &crosschainChainRepoStub{
		getByCAIP2: func(_ context.Context, caip2 string) (*entities.Chain, error) {
			switch caip2 {
			case "eip155:8453":
				return &entities.Chain{ID: sourceID}, nil
			case "eip155:42161":
				return &entities.Chain{ID: destID}, nil
			default:
				return nil, nil
			}
		},
		getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
			switch chainID {
			case "8453":
				return &entities.Chain{ID: sourceID}, nil
			case "42161":
				return &entities.Chain{ID: destID}, nil
			default:
				return nil, nil
			}
		},
	}

	var created *entities.LayerZeroConfig
	lzRepo := &layerZeroRepoErrMatrixStub{
		createFn: func(_ context.Context, cfg *entities.LayerZeroConfig) error {
			created = cfg
			return nil
		},
	}

	h := NewCrosschainPolicyHandler(routePolicyRepoNoop{}, lzRepo, chainRepo)
	r := gin.New()
	r.POST("/lz", h.CreateLayerZeroConfig)

	body := `{"sourceChainId":"eip155:8453","destChainId":"42161","dstEid":30110,"peerHex":"` + strings.Repeat("f", 64) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/lz", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.NotNil(t, created)
	require.Equal(t, "0x"+strings.Repeat("f", 64), created.PeerHex)
	require.Equal(t, "0x", created.OptionsHex)
	require.True(t, created.IsActive)
}

