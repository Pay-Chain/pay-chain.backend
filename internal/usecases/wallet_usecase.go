package usecases

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
)

// WalletUsecase handles wallet business logic
type WalletUsecase struct {
	walletRepo repositories.WalletRepository
	userRepo   repositories.UserRepository
}

// NewWalletUsecase creates a new wallet usecase
func NewWalletUsecase(walletRepo repositories.WalletRepository, userRepo repositories.UserRepository) *WalletUsecase {
	return &WalletUsecase{
		walletRepo: walletRepo,
		userRepo:   userRepo,
	}
}

// ConnectWallet connects a wallet for a user
func (u *WalletUsecase) ConnectWallet(ctx context.Context, userID uuid.UUID, input *entities.ConnectWalletInput) (*entities.Wallet, error) {
	// Validate input
	if input.ChainID == "" || input.Address == "" {
		return nil, domainerrors.ErrBadRequest
	}

	// Get user to check role and KYC status
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if user already has wallets
	existingWallets, err := u.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// If user already has a wallet (not first wallet), check KYC for non-admin/sub_admin roles
	if len(existingWallets) > 0 {
		// Admin and sub_admin can add wallets without KYC
		if user.Role != entities.UserRoleAdmin && user.Role != entities.UserRoleSubAdmin {
			// Check if KYC is fully verified
			if user.KYCStatus != entities.KYCFullyVerified {
				return nil, domainerrors.NewError("KYC verification required to add additional wallets", domainerrors.ErrForbidden)
			}
		}
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
		if existingWallet.UserID.Valid && existingWallet.UserID.String == userID.String() {
			return existingWallet, nil // Already connected
		}
		return nil, domainerrors.ErrAlreadyExists // Wallet belongs to another user
	}

	// First wallet is set as primary
	isPrimary := len(existingWallets) == 0

	// Parse chain ID to int
	chainID, err := strconv.Atoi(input.ChainID)
	if err != nil {
		// If not a pure number, might be CAIP-2 format (e.g., "eip155:1")
		// For now, try to extract the chain ID portion
		chainID = 0 // Default, repository should handle CAIP-2 format
	}

	// Create wallet with null.String for UserID
	wallet := &entities.Wallet{
		UserID:    null.StringFrom(userID.String()),
		ChainID:   chainID,
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
	if !wallet.UserID.Valid || wallet.UserID.String != userID.String() {
		return domainerrors.ErrForbidden
	}

	return u.walletRepo.SoftDelete(ctx, walletID)
}
