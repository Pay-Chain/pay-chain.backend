package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/pkg/utils"
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
	mode := normalizePaymentMode(input.Mode)
	if mode == PaymentModePrivacy {
		receiver := strings.TrimSpace(input.ReceiverAddress)
		if receiver == "" {
			return nil, fmt.Errorf("receiverAddress is required when mode=privacy")
		}
		if input.PrivacyStealthReceiver == nil || strings.TrimSpace(*input.PrivacyStealthReceiver) == "" {
			stealth := receiver
			input.PrivacyStealthReceiver = &stealth
		}
		if input.PrivacyIntentID == nil || strings.TrimSpace(*input.PrivacyIntentID) == "" {
			autoIntentID := utils.GenerateUUIDv7().String()
			input.PrivacyIntentID = &autoIntentID
		}
	}
	if _, err := normalizeBridgeOption(input.BridgeOption); err != nil {
		return nil, fmt.Errorf("invalid bridge option: %w", err)
	}
	if err := validatePrivacyFields(mode, input.PrivacyIntentID, input.PrivacyStealthReceiver); err != nil {
		return nil, err
	}

	senderAddress := strings.TrimSpace(input.SenderWalletAddress)

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

	wallet, err := u.walletRepo.GetByAddress(ctx, sourceChainID, senderAddress)
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
		email := fmt.Sprintf("%s_%s@app.paymentkita.local", walletPrefix(senderAddress), newUserID.String()[:8])

		newUser := &entities.User{
			ID:        newUserID,
			Email:     email,
			Name:      "App User " + walletNamePrefix(senderAddress),
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
			Address:   senderAddress,
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
		Mode:                   input.Mode,
		BridgeOption:           input.BridgeOption,
		BridgeTokenSource:      input.BridgeTokenSource,
		MinBridgeAmountOut:     input.MinBridgeAmountOut,
		MinDestAmountOut:       input.MinDestAmountOut,
		PrivacyIntentID:        input.PrivacyIntentID,
		PrivacyStealthReceiver: input.PrivacyStealthReceiver,
	}

	return u.paymentUsecase.CreatePayment(ctx, userID, paymentInput)
}

func walletPrefix(addr string) string {
	if len(addr) >= 8 {
		return addr[:8]
	}
	if addr == "" {
		return "wallet"
	}
	return addr
}

func walletNamePrefix(addr string) string {
	if len(addr) >= 6 {
		return addr[:6]
	}
	if addr == "" {
		return "wallet"
	}
	return addr
}
