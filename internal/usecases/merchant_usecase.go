package usecases

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
)

// MerchantUsecase handles merchant business logic
type MerchantUsecase struct {
	merchantRepo repositories.MerchantRepository
	userRepo     repositories.UserRepository
}

// NewMerchantUsecase creates a new merchant usecase
func NewMerchantUsecase(
	merchantRepo repositories.MerchantRepository,
	userRepo repositories.UserRepository,
) *MerchantUsecase {
	return &MerchantUsecase{
		merchantRepo: merchantRepo,
		userRepo:     userRepo,
	}
}

// ApplyMerchant handles merchant application
func (u *MerchantUsecase) ApplyMerchant(ctx context.Context, userID uuid.UUID, input *entities.MerchantApplyInput) (*entities.MerchantStatusResponse, error) {
	// Check if user exists
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if user already has a merchant application
	existingMerchant, err := u.merchantRepo.GetByUserID(ctx, userID)
	if err != nil && err != domainerrors.ErrNotFound {
		return nil, err
	}
	if existingMerchant != nil {
		return &entities.MerchantStatusResponse{
			MerchantID:   existingMerchant.ID,
			Status:       existingMerchant.Status,
			MerchantType: existingMerchant.MerchantType,
			BusinessName: existingMerchant.BusinessName,
			Message:      "Merchant application already exists",
			SubmittedAt:  existingMerchant.CreatedAt,
		}, nil
	}

	// Validate merchant type
	merchantType := input.MerchantType
	if merchantType != entities.MerchantTypePartner &&
		merchantType != entities.MerchantTypeCorporate &&
		merchantType != entities.MerchantTypeUMKM &&
		merchantType != entities.MerchantTypeRetail {
		return nil, domainerrors.ErrBadRequest
	}

	// Create merchant with null.String for optional fields
	merchant := &entities.Merchant{
		UserID:        user.ID,
		BusinessName:  input.BusinessName,
		BusinessEmail: input.BusinessEmail,
		MerchantType:  merchantType,
		Status:        entities.MerchantStatusPending,
	}

	// Set optional fields using null.String
	if input.TaxID != "" {
		merchant.TaxID.SetValid(input.TaxID)
	}
	if input.BusinessAddress != "" {
		merchant.BusinessAddress.SetValid(input.BusinessAddress)
	}

	if err := u.merchantRepo.Create(ctx, merchant); err != nil {
		return nil, err
	}

	return &entities.MerchantStatusResponse{
		MerchantID:   merchant.ID,
		Status:       merchant.Status,
		MerchantType: merchant.MerchantType,
		BusinessName: merchant.BusinessName,
		Message:      "Merchant application submitted successfully",
		SubmittedAt:  merchant.CreatedAt,
	}, nil
}

// GetMerchantStatus gets merchant status for a user
func (u *MerchantUsecase) GetMerchantStatus(ctx context.Context, userID uuid.UUID) (*entities.MerchantStatusResponse, error) {
	merchant, err := u.merchantRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			return &entities.MerchantStatusResponse{
				Status:  entities.MerchantStatusPending, // Use pending as default for "not_applied"
				Message: "No merchant application found",
			}, nil
		}
		return nil, err
	}

	return &entities.MerchantStatusResponse{
		MerchantID:   merchant.ID,
		Status:       merchant.Status,
		MerchantType: merchant.MerchantType,
		BusinessName: merchant.BusinessName,
		Message:      getStatusMessage(merchant.Status),
		SubmittedAt:  merchant.CreatedAt,
		ReviewedAt:   merchant.VerifiedAt,
	}, nil
}

// ApproveMerchant approves a merchant application (admin only)
func (u *MerchantUsecase) ApproveMerchant(ctx context.Context, merchantID uuid.UUID) error {
	return u.merchantRepo.UpdateStatus(ctx, merchantID, entities.MerchantStatusActive)
}

// RejectMerchant rejects a merchant application (admin only)
func (u *MerchantUsecase) RejectMerchant(ctx context.Context, merchantID uuid.UUID) error {
	return u.merchantRepo.UpdateStatus(ctx, merchantID, entities.MerchantStatusRejected)
}

// SuspendMerchant suspends a merchant (admin only)
func (u *MerchantUsecase) SuspendMerchant(ctx context.Context, merchantID uuid.UUID) error {
	return u.merchantRepo.UpdateStatus(ctx, merchantID, entities.MerchantStatusSuspended)
}

func getStatusMessage(status entities.MerchantStatus) string {
	switch status {
	case entities.MerchantStatusPending:
		return "Your merchant application is under review"
	case entities.MerchantStatusActive:
		return "Your merchant account is active"
	case entities.MerchantStatusSuspended:
		return "Your merchant account has been suspended"
	case entities.MerchantStatusRejected:
		return "Your merchant application was rejected"
	default:
		return ""
	}
}
