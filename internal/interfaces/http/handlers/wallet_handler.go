package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
)

// WalletHandler handles wallet endpoints
type WalletHandler struct {
	walletUsecase *usecases.WalletUsecase
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)

	wallet, err := h.walletUsecase.ConnectWallet(c.Request.Context(), userID, &input)
	if err != nil {
		if err == domainerrors.ErrBadRequest {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		if err == domainerrors.ErrAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Wallet already connected to another account"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect wallet"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Wallet connected successfully",
		"wallet":  wallet,
	})
}

// ListWallets lists wallets for the current user
// GET /api/v1/wallets
func (h *WalletHandler) ListWallets(c *gin.Context) {
	userID := middleware.GetUserID(c)

	wallets, err := h.walletUsecase.GetWallets(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list wallets"})
		return
	}

	if wallets == nil {
		wallets = []*entities.Wallet{}
	}

	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

// SetPrimaryWallet sets a wallet as primary
// PUT /api/v1/wallets/:id/primary
func (h *WalletHandler) SetPrimaryWallet(c *gin.Context) {
	idStr := c.Param("id")
	walletID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet ID"})
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.walletUsecase.SetPrimaryWallet(c.Request.Context(), userID, walletID); err != nil {
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set primary wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Primary wallet updated"})
}

// DisconnectWallet disconnects a wallet
// DELETE /api/v1/wallets/:id
func (h *WalletHandler) DisconnectWallet(c *gin.Context) {
	idStr := c.Param("id")
	walletID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet ID"})
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.walletUsecase.DisconnectWallet(c.Request.Context(), userID, walletID); err != nil {
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
			return
		}
		if err == domainerrors.ErrForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wallet disconnected"})
}
