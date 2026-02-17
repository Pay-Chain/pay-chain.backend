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

func TestCrosschainPolicyHandler_UpdateLayerZeroConfig_NormalizationAndOptionalActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sourceID := uuid.New()
	destID := uuid.New()
	configID := uuid.New()

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

	existing := &entities.LayerZeroConfig{
		ID:            configID,
		SourceChainID: sourceID,
		DestChainID:   destID,
		DstEID:        30110,
		PeerHex:       "0x" + strings.Repeat("a", 64),
		OptionsHex:    "0x0102",
		IsActive:      true,
	}

	lzRepo := &layerZeroRepoErrMatrixStub{
		item: existing,
		getByIDFn: func(_ context.Context, id uuid.UUID) (*entities.LayerZeroConfig, error) {
			if id == configID {
				return existing, nil
			}
			return nil, nil
		},
	}

	h := NewCrosschainPolicyHandler(routePolicyRepoNoop{}, lzRepo, chainRepo)
	r := gin.New()
	r.PUT("/lz/:id", h.UpdateLayerZeroConfig)

	// optionsHex omitted + peer without 0x should normalize; isActive omitted should preserve existing value.
	body := `{"sourceChainId":"eip155:8453","destChainId":"42161","dstEid":30111,"peerHex":"` + strings.Repeat("b", 64) + `"}`
	req := httptest.NewRequest(http.MethodPut, "/lz/"+configID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, uint32(30111), existing.DstEID)
	require.Equal(t, "0x"+strings.Repeat("b", 64), existing.PeerHex)
	require.Equal(t, "0x", existing.OptionsHex)
	require.True(t, existing.IsActive)
}

