package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
)

type walletServiceStub struct {
	connectFn    func(context.Context, uuid.UUID, *entities.ConnectWalletInput) (*entities.Wallet, error)
	listFn       func(context.Context, uuid.UUID) ([]*entities.Wallet, error)
	setPrimaryFn func(context.Context, uuid.UUID, uuid.UUID) error
	disconnectFn func(context.Context, uuid.UUID, uuid.UUID) error
}

func (s walletServiceStub) ConnectWallet(ctx context.Context, userID uuid.UUID, input *entities.ConnectWalletInput) (*entities.Wallet, error) {
	if s.connectFn != nil {
		return s.connectFn(ctx, userID, input)
	}
	return nil, nil
}
func (s walletServiceStub) GetWallets(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error) {
	if s.listFn != nil {
		return s.listFn(ctx, userID)
	}
	return []*entities.Wallet{}, nil
}
func (s walletServiceStub) SetPrimaryWallet(ctx context.Context, userID, walletID uuid.UUID) error {
	if s.setPrimaryFn != nil {
		return s.setPrimaryFn(ctx, userID, walletID)
	}
	return nil
}
func (s walletServiceStub) DisconnectWallet(ctx context.Context, userID, walletID uuid.UUID) error {
	if s.disconnectFn != nil {
		return s.disconnectFn(ctx, userID, walletID)
	}
	return nil
}

func TestWalletHandler_ConnectWallet_ErrorMapping_Gap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	h := &WalletHandler{walletUsecase: walletServiceStub{
		connectFn: func(_ context.Context, _ uuid.UUID, input *entities.ConnectWalletInput) (*entities.Wallet, error) {
			switch input.Address {
			case "0xbad":
				return nil, domainerrors.ErrBadRequest
			case "0xexists":
				return nil, domainerrors.ErrAlreadyExists
			default:
				return nil, errors.New("boom")
			}
		},
	}}

	r := gin.New()
	r.POST("/wallets/connect", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		h.ConnectWallet(c)
	})

	for address, wantCode := range map[string]int{
		"0xbad":    http.StatusBadRequest,
		"0xexists": http.StatusConflict,
		"0xboom":   http.StatusInternalServerError,
	} {
		body := `{"chainId":"eip155:8453","address":"` + address + `","signature":"sig","message":"msg"}`
		req := httptest.NewRequest(http.MethodPost, "/wallets/connect", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, wantCode, w.Code, "address=%s body=%s", address, w.Body.String())
	}
}
