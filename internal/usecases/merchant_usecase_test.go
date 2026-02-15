package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/usecases"
)

func TestMerchantUsecase_ApplyMerchant_ExistingApplication(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, mockUserRepo)

	userID := uuid.New()
	now := time.Now()
	user := &entities.User{ID: userID}
	existing := &entities.Merchant{
		ID:           uuid.New(),
		UserID:       userID,
		BusinessName: "Biz",
		MerchantType: entities.MerchantTypeCorporate,
		Status:       entities.MerchantStatusPending,
		CreatedAt:    now,
	}

	mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
	mockMerchantRepo.On("GetByUserID", context.Background(), userID).Return(existing, nil).Once()

	resp, err := uc.ApplyMerchant(context.Background(), userID, &entities.MerchantApplyInput{
		MerchantType:  entities.MerchantTypeCorporate,
		BusinessName:  "Biz",
		BusinessEmail: "biz@mail.com",
	})
	assert.NoError(t, err)
	assert.Equal(t, existing.ID, resp.MerchantID)
	assert.Equal(t, "Merchant application already exists", resp.Message)
}

func TestMerchantUsecase_ApplyMerchant_InvalidType(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, mockUserRepo)

	userID := uuid.New()
	user := &entities.User{ID: userID}

	mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
	mockMerchantRepo.On("GetByUserID", context.Background(), userID).Return(nil, domainerrors.ErrNotFound).Once()

	_, err := uc.ApplyMerchant(context.Background(), userID, &entities.MerchantApplyInput{
		MerchantType:  entities.MerchantType("INVALID"),
		BusinessName:  "Biz",
		BusinessEmail: "biz@mail.com",
	})
	assert.ErrorIs(t, err, domainerrors.ErrBadRequest)
}

func TestMerchantUsecase_GetMerchantStatus_NotFound(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, mockUserRepo)

	userID := uuid.New()
	mockMerchantRepo.On("GetByUserID", context.Background(), userID).Return(nil, domainerrors.ErrNotFound).Once()

	resp, err := uc.GetMerchantStatus(context.Background(), userID)
	assert.NoError(t, err)
	assert.Equal(t, entities.MerchantStatusPending, resp.Status)
	assert.Equal(t, "No merchant application found", resp.Message)
}

func TestMerchantUsecase_GetMerchantStatus_Active(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, mockUserRepo)

	userID := uuid.New()
	merchant := &entities.Merchant{
		ID:           uuid.New(),
		UserID:       userID,
		BusinessName: "Biz",
		MerchantType: entities.MerchantTypeRetail,
		Status:       entities.MerchantStatusActive,
	}
	mockMerchantRepo.On("GetByUserID", context.Background(), userID).Return(merchant, nil).Once()

	resp, err := uc.GetMerchantStatus(context.Background(), userID)
	assert.NoError(t, err)
	assert.Equal(t, "Your merchant account is active", resp.Message)
}
