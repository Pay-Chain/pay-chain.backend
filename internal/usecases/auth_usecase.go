package usecases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/pkg/crypto"
	"pay-chain.backend/pkg/jwt"
)

// AuthUsecase handles authentication business logic
type AuthUsecase struct {
	userRepo       repositories.UserRepository
	emailVerifRepo repositories.EmailVerificationRepository
	walletRepo     repositories.WalletRepository
	jwtService     *jwt.JWTService
}

// NewAuthUsecase creates a new auth usecase
func NewAuthUsecase(
	userRepo repositories.UserRepository,
	emailVerifRepo repositories.EmailVerificationRepository,
	walletRepo repositories.WalletRepository,
	jwtService *jwt.JWTService,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:       userRepo,
		emailVerifRepo: emailVerifRepo,
		walletRepo:     walletRepo,
		jwtService:     jwtService,
	}
}

// Register registers a new user with mandatory wallet
func (u *AuthUsecase) Register(ctx context.Context, input *entities.CreateUserInput) (*entities.User, string, error) {
	// Validate wallet fields are provided
	if input.WalletAddress == "" || input.WalletChainID == "" || input.WalletSignature == "" {
		return nil, "", domainerrors.ErrBadRequest
	}

	// TODO: Verify wallet signature
	// This would involve:
	// 1. Recover signer from signature
	// 2. Compare recovered address with input.WalletAddress
	// 3. Verify message format and timestamp
	// For now, we accept any valid-looking signature

	// Check if email already exists
	_, err := u.userRepo.GetByEmail(ctx, input.Email)
	if err == nil {
		return nil, "", domainerrors.ErrAlreadyExists
	}
	if !errors.Is(err, domainerrors.ErrNotFound) {
		return nil, "", err
	}

	// Check if wallet already registered to another user
	checkChainID, err := uuid.Parse(input.WalletChainID)
	if err != nil {
		return nil, "", domainerrors.ErrInvalidInput
	}
	existingWallet, err := u.walletRepo.GetByAddress(ctx, checkChainID, input.WalletAddress)
	if err != nil && !errors.Is(err, domainerrors.ErrNotFound) {
		return nil, "", err
	}
	if existingWallet != nil && existingWallet.UserID.Valid {
		return nil, "", domainerrors.NewError("wallet already registered to another user", domainerrors.ErrAlreadyExists)
	}

	// Hash password
	passwordHash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return nil, "", err
	}

	// Create user
	user := &entities.User{
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: passwordHash,
		Role:         entities.UserRoleUser,
		KYCStatus:    entities.KYCNotStarted,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, "", err
	}

	// Create wallet linked to user (as primary)
	chainID, err := uuid.Parse(input.WalletChainID)
	if err != nil {
		return nil, "", domainerrors.ErrInvalidInput
	}

	wallet := &entities.Wallet{
		UserID:    null.StringFrom(user.ID.String()),
		ChainID:   chainID,
		Address:   input.WalletAddress,
		IsPrimary: true,
	}

	if err := u.walletRepo.Create(ctx, wallet); err != nil {
		// Rollback user creation would be ideal here
		// For now, log and continue (user exists but wallet failed)
		return nil, "", err
	}

	// Generate verification token
	token, err := crypto.GenerateVerificationToken()
	if err != nil {
		return nil, "", err
	}

	// Save verification token
	if err := u.emailVerifRepo.Create(ctx, user.ID, token); err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login authenticates a user and returns tokens
func (u *AuthUsecase) Login(ctx context.Context, input *entities.LoginInput) (*entities.AuthResponse, error) {
	// Get user by email
	user, err := u.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domainerrors.ErrNotFound) {
			return nil, domainerrors.ErrInvalidCredentials
		}
		return nil, err
	}

	// Check password
	if !crypto.CheckPassword(input.Password, user.PasswordHash) {
		return nil, domainerrors.ErrInvalidCredentials
	}

	// Generate tokens
	tokenPair, err := u.jwtService.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}

	return &entities.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         user,
	}, nil
}

// VerifyEmail verifies a user's email
func (u *AuthUsecase) VerifyEmail(ctx context.Context, token string) error {
	// Get user by token
	user, err := u.emailVerifRepo.GetByToken(ctx, token)
	if err != nil {
		return err
	}

	// Mark token as verified
	if err := u.emailVerifRepo.MarkVerified(ctx, token); err != nil {
		return err
	}

	// Update user (could set email_verified flag if we had one)
	_ = user // User is already loaded, we verified via the token

	return nil
}

// RefreshToken generates new tokens from a refresh token
func (u *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	// Validate refresh token
	claims, err := u.jwtService.ValidateToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Get current user to ensure still valid
	user, err := u.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Generate new token pair
	return u.jwtService.GenerateTokenPair(user.ID, user.Email, string(user.Role))
}

// GetUserByID gets a user by ID
func (u *AuthUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	return u.userRepo.GetByID(ctx, id)
}
