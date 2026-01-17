package usecases

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
)

// WalletUsecase handles wallet business logic
type WalletUsecase struct {
	walletRepo repositories.WalletRepository
}

// NewWalletUsecase creates a new wallet usecase
func NewWalletUsecase(walletRepo repositories.WalletRepository) *WalletUsecase {
	return &WalletUsecase{walletRepo: walletRepo}
}

// ConnectWallet connects a wallet for a user
func (u *WalletUsecase) ConnectWallet(ctx context.Context, userID uuid.UUID, input *entities.ConnectWalletInput) (*entities.Wallet, error) {
	// Validate input
	if input.ChainID == "" || input.Address == "" {
		return nil, domainerrors.ErrBadRequest
	}

	// TODO: Verify signature
	// This would involve:
	// 1. Recover signer from signature
	// 2. Compare recovered address with input.Address
	// 3. Verify message format and timestamp

	// Check if wallet already exists
	existingWallet, err := u.walletRepo.GetByAddress(ctx, input.ChainID, input.Address)
	if err != nil && err != domainerrors.ErrNotFound {
		return nil, err
	}
	if existingWallet != nil {
		if existingWallet.UserID == userID {
			return existingWallet, nil // Already connected
		}
		return nil, domainerrors.ErrAlreadyExists // Wallet belongs to another user
	}

	// Check if user has any wallets (set first as primary)
	existingWallets, err := u.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	isPrimary := len(existingWallets) == 0

	// Create wallet
	wallet := &entities.Wallet{
		UserID:    userID,
		ChainID:   input.ChainID,
		Address:   input.Address,
		IsPrimary: isPrimary,
	}

	if err := u.walletRepo.Create(ctx, wallet); err != nil {
		return nil, err
	}

	return wallet, nil
}

// GetWallets gets all wallets for a user
func (u *WalletUsecase) GetWallets(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error) {
	return u.walletRepo.GetByUserID(ctx, userID)
}

// SetPrimaryWallet sets a wallet as primary
func (u *WalletUsecase) SetPrimaryWallet(ctx context.Context, userID, walletID uuid.UUID) error {
	return u.walletRepo.SetPrimary(ctx, userID, walletID)
}

// DisconnectWallet disconnects a wallet
func (u *WalletUsecase) DisconnectWallet(ctx context.Context, userID, walletID uuid.UUID) error {
	// Verify wallet belongs to user
	wallet, err := u.walletRepo.GetByID(ctx, walletID)
	if err != nil {
		return err
	}
	if wallet.UserID != userID {
		return domainerrors.ErrForbidden
	}

	return u.walletRepo.SoftDelete(ctx, walletID)
}
