package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/internal/usecases"
)

type walletService interface {
	ConnectWallet(ctx context.Context, userID uuid.UUID, input *entities.ConnectWalletInput) (*entities.Wallet, error)
	GetWallets(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error)
	SetPrimaryWallet(ctx context.Context, userID, walletID uuid.UUID) error
	DisconnectWallet(ctx context.Context, userID, walletID uuid.UUID) error
}

// WalletHandler handles wallet endpoints
type WalletHandler struct {
	walletUsecase walletService
}

// NewWalletHandler creates a new wallet handler
func NewWalletHandler(walletUsecase *usecases.WalletUsecase) *WalletHandler {
	return &WalletHandler{walletUsecase: walletUsecase}
}

// ConnectWallet connects a wallet
// POST /api/v1/wallets/connect
func (h *WalletHandler) ConnectWallet(c *gin.Context) {
	var input entities.ConnectWalletInput

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, domainerrors.Unauthorized("User not authenticated"))
		return
	}

	wallet, err := h.walletUsecase.ConnectWallet(c.Request.Context(), userID, &input)
	if err != nil {
		if err == domainerrors.ErrBadRequest {
			response.Error(c, domainerrors.BadRequest("Invalid input"))
			return
		}
		if err == domainerrors.ErrAlreadyExists {
			response.Error(c, domainerrors.Conflict("Wallet already connected to another account"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, gin.H{
		"message": "Wallet connected successfully",
		"wallet":  wallet,
	})
}

// ListWallets lists wallets for the current user
// GET /api/v1/wallets
func (h *WalletHandler) ListWallets(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, domainerrors.Unauthorized("User not authenticated"))
		return
	}

	wallets, err := h.walletUsecase.GetWallets(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	if wallets == nil {
		wallets = []*entities.Wallet{}
	}

	response.Success(c, http.StatusOK, gin.H{"wallets": wallets})
}

// SetPrimaryWallet sets a wallet as primary
// PUT /api/v1/wallets/:id/primary
func (h *WalletHandler) SetPrimaryWallet(c *gin.Context) {
	idStr := c.Param("id")
	walletID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid wallet ID"))
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, domainerrors.Unauthorized("User not authenticated"))
		return
	}

	if err := h.walletUsecase.SetPrimaryWallet(c.Request.Context(), userID, walletID); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Wallet not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Primary wallet updated"})
}

// DisconnectWallet disconnects a wallet
// DELETE /api/v1/wallets/:id
func (h *WalletHandler) DisconnectWallet(c *gin.Context) {
	idStr := c.Param("id")
	walletID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid wallet ID"))
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, domainerrors.Unauthorized("User not authenticated"))
		return
	}

	if err := h.walletUsecase.DisconnectWallet(c.Request.Context(), userID, walletID); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Wallet not found"))
			return
		}
		if err == domainerrors.ErrForbidden {
			response.Error(c, domainerrors.Forbidden("Not authorized"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Wallet disconnected"})
}
