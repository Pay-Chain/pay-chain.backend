package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/pkg/utils"
)

type PaymentAppUsecase struct {
	paymentUsecase *PaymentUsecase
	userRepo       repositories.UserRepository
	walletRepo     repositories.WalletRepository
	chainRepo      repositories.ChainRepository
	chainResolver  *ChainResolver
}

func NewPaymentAppUsecase(
	paymentUsecase *PaymentUsecase,
	userRepo repositories.UserRepository,
	walletRepo repositories.WalletRepository,
	chainRepo repositories.ChainRepository,
) *PaymentAppUsecase {
	return &PaymentAppUsecase{
		paymentUsecase: paymentUsecase,
		userRepo:       userRepo,
		walletRepo:     walletRepo,
		chainRepo:      chainRepo,
		chainResolver:  NewChainResolver(chainRepo),
	}
}

func (u *PaymentAppUsecase) CreatePaymentApp(ctx context.Context, input *entities.CreatePaymentAppInput) (*entities.CreatePaymentResponse, error) {
	sourceChainID, sourceCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.SourceChainID)
	if err != nil {
		return nil, fmt.Errorf("invalid source chain: %w", err)
	}
	_, destCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.DestChainID)
	if err != nil {
		return nil, fmt.Errorf("invalid destination chain: %w", err)
	}

	// 2. Resolve User logic
	var userID uuid.UUID

	wallet, err := u.walletRepo.GetByAddress(ctx, sourceChainID, input.SenderWalletAddress)
	if err == nil && wallet != nil && wallet.UserID != nil {
		// Case A: Wallet exists -> Use existing User
		userID = *wallet.UserID
	} else {
		// Case B: Wallet not found (or no user attached) -> Create new User + Wallet
		// Note: Ideally we should check if address exists on OTHER chains to link to same user (EVM),
		// but `GetByAddress` is chain-scoped. For MVP, we create new user if not found on THIS chain.
		// Improvement: Add `walletRepo.FindByAddressAnyChain(address)` later.

		// Create User
		newUserID := utils.GenerateUUIDv7()
		email := fmt.Sprintf("%s_%s@app.paychain.local", input.SenderWalletAddress[:8], newUserID.String()[:8])

		newUser := &entities.User{
			ID:        newUserID,
			Email:     email,
			Name:      "App User " + input.SenderWalletAddress[:6],
			Role:      entities.UserRoleUser,
			KYCStatus: entities.KYCNotStarted,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		// We set a dummy password hash since they auth via wallet/api-key context (not password login)
		newUser.PasswordHash = "WALLET_AUTH_NO_PASSWORD"

		if err := u.userRepo.Create(ctx, newUser); err != nil {
			return nil, fmt.Errorf("failed to auto-create user: %w", err)
		}
		userID = newUser.ID

		// Create Wallet
		newWallet := &entities.Wallet{
			ID:        utils.GenerateUUIDv7(),
			UserID:    &userID,
			ChainID:   sourceChainID,
			Address:   input.SenderWalletAddress,
			Type:      "EOA",
			IsPrimary: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := u.walletRepo.Create(ctx, newWallet); err != nil {
			// Basic rollback cleanup could be here, but for now just fail
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
	}

	// 3. Delegated Payment Creation
	// Map AppInput to PaymentInput
	paymentInput := &entities.CreatePaymentInput{
		SourceChainID:      sourceCAIP2,
		DestChainID:        destCAIP2,
		SourceTokenAddress: input.SourceTokenAddress,
		DestTokenAddress:   input.DestTokenAddress,
		Amount:             input.Amount,
		Decimals:           input.Decimals,
		ReceiverAddress:    input.ReceiverAddress,
		// ReceiverMerchantID is empty for App payments (any receiver allowed)
	}

	return u.paymentUsecase.CreatePayment(ctx, userID, paymentInput)
}
