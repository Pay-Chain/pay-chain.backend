package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"

	"pay-chain.backend/pkg/crypto"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/redis"
	"pay-chain.backend/pkg/utils"
)

var (
	authHashPassword              = crypto.HashPassword
	authGenerateVerificationToken = crypto.GenerateVerificationToken
	authJSONMarshal               = json.Marshal
	authRedisSet                  = redis.Set
	authGenerateTokenPair         = func(s *jwt.JWTService, userID uuid.UUID, email, role string) (*jwt.TokenPair, error) {
		return s.GenerateTokenPair(userID, email, role)
	}
)

// AuthUsecase handles authentication business logic
type AuthUsecase struct {
	userRepo       repositories.UserRepository
	emailVerifRepo repositories.EmailVerificationRepository
	walletRepo     repositories.WalletRepository
	chainRepo      repositories.ChainRepository
	chainResolver  *ChainResolver
	jwtService     *jwt.JWTService
}

// NewAuthUsecase creates a new auth usecase
func NewAuthUsecase(
	userRepo repositories.UserRepository,
	emailVerifRepo repositories.EmailVerificationRepository,
	walletRepo repositories.WalletRepository,
	chainRepo repositories.ChainRepository,
	jwtService *jwt.JWTService,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:       userRepo,
		emailVerifRepo: emailVerifRepo,
		walletRepo:     walletRepo,
		chainRepo:      chainRepo,
		chainResolver:  NewChainResolver(chainRepo),
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
	chainUUID, _, err := u.chainResolver.ResolveFromAny(ctx, input.WalletChainID)
	if err != nil {
		return nil, "", domainerrors.ErrInvalidInput
	}
	existingWallet, err := u.walletRepo.GetByAddress(ctx, chainUUID, input.WalletAddress)
	if err != nil && !errors.Is(err, domainerrors.ErrNotFound) {
		return nil, "", err
	}
	if existingWallet != nil && existingWallet.UserID != nil {
		return nil, "", domainerrors.NewError("wallet already registered to another user", domainerrors.ErrAlreadyExists)
	}

	// Hash password
	passwordHash, err := authHashPassword(input.Password)
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
	wallet := &entities.Wallet{
		UserID:    &user.ID,
		ChainID:   chainUUID,
		Address:   input.WalletAddress,
		IsPrimary: true,
	}

	if err := u.walletRepo.Create(ctx, wallet); err != nil {
		// Rollback user creation would be ideal here
		// For now, log and continue (user exists but wallet failed)
		return nil, "", err
	}

	// Generate verification token
	token, err := authGenerateVerificationToken()
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
	tokenPair, err := authGenerateTokenPair(u.jwtService, user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}

	// Handle Session Request
	if input.UseSession {
		sessionID := utils.GenerateUUIDv7().String()
		sessionKey := fmt.Sprintf("session:%s", sessionID)

		// Store session data in Redis
		expiration := 7 * 24 * time.Hour

		sessionData := map[string]interface{}{
			"userId":       user.ID.String(),
			"email":        user.Email,
			"role":         string(user.Role),
			"accessToken":  tokenPair.AccessToken,
			"refreshToken": tokenPair.RefreshToken,
		}

		jsonData, err := authJSONMarshal(sessionData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal session data: %w", err)
		}

		if err := authRedisSet(ctx, sessionKey, jsonData, expiration); err != nil {
			return nil, fmt.Errorf("failed to store session in redis: %w", err)
		}

		return &entities.AuthResponse{
			SessionID: sessionID,
			User:      user,
		}, nil
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
	return authGenerateTokenPair(u.jwtService, user.ID, user.Email, string(user.Role))
}

// GetUserByID gets a user by ID
func (u *AuthUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	return u.userRepo.GetByID(ctx, id)
}

// GetTokenExpiry returns token expiry unix timestamp.
func (u *AuthUsecase) GetTokenExpiry(token string) (int64, error) {
	claims, err := u.jwtService.ValidateToken(token)
	if err != nil {
		return 0, err
	}
	if claims.RegisteredClaims.ExpiresAt == nil {
		return 0, fmt.Errorf("token missing exp claim")
	}
	return claims.RegisteredClaims.ExpiresAt.Time.Unix(), nil
}

// ChangePassword updates password after verifying current password.
func (u *AuthUsecase) ChangePassword(ctx context.Context, userID uuid.UUID, input *entities.ChangePasswordInput) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !crypto.CheckPassword(input.CurrentPassword, user.PasswordHash) {
		return domainerrors.NewAppError(401, domainerrors.CodeInvalidCredentials, "Current password is incorrect", domainerrors.ErrInvalidCredentials)
	}

	newPasswordHash, err := crypto.HashPassword(input.NewPassword)
	if err != nil {
		return err
	}

	return u.userRepo.UpdatePassword(ctx, userID, newPasswordHash)
}
