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
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
)

type walletRepoSetPrimaryErrStub struct {
	*walletRepoStub
}

func (s *walletRepoSetPrimaryErrStub) SetPrimary(context.Context, uuid.UUID, uuid.UUID) error {
	return errors.New("set primary failed")
}

func TestMerchantHandler_GapBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	t.Run("apply unauthorized with valid payload", func(t *testing.T) {
		uc := usecases.NewMerchantUsecase(
			newMerchantRepoStub(),
			merchantUserRepoStub{users: map[uuid.UUID]*entities.User{
				userID: {ID: userID, Email: "merchant@paychain.io", Role: entities.UserRoleUser},
			}},
		)
		h := NewMerchantHandler(uc)
		r := gin.New()
		r.POST("/merchants/apply", h.ApplyMerchant)

		body := `{"merchantType":"UMKM","businessName":"Warung","businessEmail":"warung@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/merchants/apply", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("apply/get status usecase error branch", func(t *testing.T) {
		uc := usecases.NewMerchantUsecase(
			newMerchantRepoStub(),
			merchantUserRepoStub{users: map[uuid.UUID]*entities.User{}}, // force not found
		)
		h := NewMerchantHandler(uc)
		r := gin.New()
		withUser := func(c *gin.Context) {
			c.Set(middleware.UserIDKey, userID)
			c.Next()
		}
		r.POST("/merchants/apply", withUser, h.ApplyMerchant)
		r.GET("/merchants/status", withUser, h.GetMerchantStatus)

		body := `{"merchantType":"UMKM","businessName":"Warung","businessEmail":"warung@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/merchants/apply", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)

		req = httptest.NewRequest(http.MethodGet, "/merchants/status", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestWalletHandler_SetPrimary_GapBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	chainID := uuid.New()
	walletID := uuid.New()

	ucUnauthorized := usecases.NewWalletUsecase(
		newWalletRepoStub(),
		walletUserRepoStub{user: &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}},
		walletChainRepoStub{chain: &entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}},
	)
	hUnauthorized := NewWalletHandler(ucUnauthorized)
	noAuth := gin.New()
	noAuth.PUT("/wallets/:id/primary", hUnauthorized.SetPrimaryWallet)
	req := httptest.NewRequest(http.MethodPut, "/wallets/"+walletID.String()+"/primary", nil)
	w := httptest.NewRecorder()
	noAuth.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	repo := &walletRepoSetPrimaryErrStub{walletRepoStub: newWalletRepoStub()}
	repo.items[walletID] = &entities.Wallet{ID: walletID, UserID: &userID, ChainID: chainID, Address: "0xabc"}
	ucErr := usecases.NewWalletUsecase(
		repo,
		walletUserRepoStub{user: &entities.User{ID: userID, Role: entities.UserRoleUser, KYCStatus: entities.KYCFullyVerified}},
		walletChainRepoStub{chain: &entities.Chain{ID: chainID, ChainID: "8453", Type: entities.ChainTypeEVM}},
	)
	hErr := NewWalletHandler(ucErr)
	r := gin.New()
	withUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
	r.PUT("/wallets/:id/primary", withUser, hErr.SetPrimaryWallet)
	req = httptest.NewRequest(http.MethodPut, "/wallets/"+walletID.String()+"/primary", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPaymentConfigHandler_UpdateBridgeConfig_GapBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configID := uuid.New()
	sourceChainID := uuid.New()
	destChainID := uuid.New()
	bridgeID := uuid.New()

	chainRepo := &crosschainChainRepoStub{
		getByChainID: func(_ context.Context, chainID string) (*entities.Chain, error) {
			switch chainID {
			case "eip155:8453":
				return &entities.Chain{ID: sourceChainID}, nil
			case "eip155:42161":
				return &entities.Chain{ID: destChainID}, nil
			default:
				return nil, domainerrors.ErrNotFound
			}
		},
		getByCAIP2: func(context.Context, string) (*entities.Chain, error) { return nil, domainerrors.ErrNotFound },
	}

	makeRouter := func(repo *bridgeConfigRepoErrStub) *gin.Engine {
		h := NewPaymentConfigHandler(nil, repo, nil, chainRepo, nil)
		r := gin.New()
		r.PUT("/bridge-configs/:id", h.UpdateBridgeConfig)
		return r
	}

	repoGetErr := &bridgeConfigRepoErrStub{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.BridgeConfig, error) { return nil, errors.New("get failed") },
	}
	r := makeRouter(repoGetErr)
	req := httptest.NewRequest(http.MethodPut, "/bridge-configs/"+configID.String(), bytes.NewBufferString(`{"bridgeId":"`+bridgeID.String()+`","sourceChainId":"eip155:8453","destChainId":"eip155:42161"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	repoBase := &bridgeConfigRepoErrStub{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.BridgeConfig, error) { return &entities.BridgeConfig{ID: configID}, nil },
		updateFn:  func(context.Context, *entities.BridgeConfig) error { return nil },
	}
	r = makeRouter(repoBase)

	req = httptest.NewRequest(http.MethodPut, "/bridge-configs/"+configID.String(), bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/bridge-configs/"+configID.String(), bytes.NewBufferString(`{"bridgeId":"bad","sourceChainId":"eip155:8453","destChainId":"eip155:42161"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/bridge-configs/"+configID.String(), bytes.NewBufferString(`{"bridgeId":"`+bridgeID.String()+`","sourceChainId":"bad-source","destChainId":"eip155:42161"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/bridge-configs/"+configID.String(), bytes.NewBufferString(`{"bridgeId":"`+bridgeID.String()+`","sourceChainId":"eip155:8453","destChainId":"bad-dest"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	repoUpdateErr := &bridgeConfigRepoErrStub{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.BridgeConfig, error) { return &entities.BridgeConfig{ID: configID}, nil },
		updateFn:  func(context.Context, *entities.BridgeConfig) error { return errors.New("update failed") },
	}
	r = makeRouter(repoUpdateErr)
	req = httptest.NewRequest(http.MethodPut, "/bridge-configs/"+configID.String(), bytes.NewBufferString(`{"bridgeId":"`+bridgeID.String()+`","sourceChainId":"eip155:8453","destChainId":"eip155:42161"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
