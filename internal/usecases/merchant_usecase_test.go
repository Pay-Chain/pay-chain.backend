package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestMerchantUsecase_AdminStatusActions(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, new(MockUserRepository))

	merchantID := uuid.New()
	mockMerchantRepo.On("UpdateStatus", context.Background(), merchantID, entities.MerchantStatusActive).Return(nil).Once()
	mockMerchantRepo.On("UpdateStatus", context.Background(), merchantID, entities.MerchantStatusRejected).Return(nil).Once()
	mockMerchantRepo.On("UpdateStatus", context.Background(), merchantID, entities.MerchantStatusSuspended).Return(nil).Once()

	assert.NoError(t, uc.ApproveMerchant(context.Background(), merchantID))
	assert.NoError(t, uc.RejectMerchant(context.Background(), merchantID))
	assert.NoError(t, uc.SuspendMerchant(context.Background(), merchantID))
}

func TestMerchantUsecase_ApplyMerchant_SuccessCreate(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, mockUserRepo)

	userID := uuid.New()
	user := &entities.User{ID: userID}

	mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
	mockMerchantRepo.On("GetByUserID", context.Background(), userID).Return(nil, domainerrors.ErrNotFound).Once()
	mockMerchantRepo.On("Create", context.Background(), mock.AnythingOfType("*entities.Merchant")).Return(nil).Once()

	resp, err := uc.ApplyMerchant(context.Background(), userID, &entities.MerchantApplyInput{
		MerchantType:    entities.MerchantTypeUMKM,
		BusinessName:    "Warung Maju",
		BusinessEmail:   "warung@contoh.id",
		TaxID:           "NPWP-001",
		BusinessAddress: "Jl. Merdeka",
	})
	assert.NoError(t, err)
	assert.Equal(t, entities.MerchantStatusPending, resp.Status)
	assert.Equal(t, "Merchant application submitted successfully", resp.Message)
	assert.Equal(t, "Warung Maju", resp.BusinessName)
}

func TestMerchantUsecase_ApplyMerchant_UserLookupError(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, mockUserRepo)

	userID := uuid.New()
	mockUserRepo.On("GetByID", context.Background(), userID).Return((*entities.User)(nil), errors.New("db user error")).Once()

	_, err := uc.ApplyMerchant(context.Background(), userID, &entities.MerchantApplyInput{
		MerchantType:  entities.MerchantTypeRetail,
		BusinessName:  "Biz",
		BusinessEmail: "biz@mail.com",
	})
	assert.EqualError(t, err, "db user error")
}

func TestMerchantUsecase_ApplyMerchant_GetByUserError(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, mockUserRepo)

	userID := uuid.New()
	user := &entities.User{ID: userID}

	mockUserRepo.On("GetByID", context.Background(), userID).Return(user, nil).Once()
	mockMerchantRepo.On("GetByUserID", context.Background(), userID).Return((*entities.Merchant)(nil), errors.New("repo error")).Once()

	_, err := uc.ApplyMerchant(context.Background(), userID, &entities.MerchantApplyInput{
		MerchantType:  entities.MerchantTypeRetail,
		BusinessName:  "Biz",
		BusinessEmail: "biz@mail.com",
	})
	assert.EqualError(t, err, "repo error")
}

func TestMerchantUsecase_GetMerchantStatus_RepoError(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, mockUserRepo)

	userID := uuid.New()
	mockMerchantRepo.On("GetByUserID", context.Background(), userID).Return((*entities.Merchant)(nil), errors.New("status error")).Once()

	_, err := uc.GetMerchantStatus(context.Background(), userID)
	assert.EqualError(t, err, "status error")
}

func TestMerchantUsecase_GetMerchantStatus_MessageVariants(t *testing.T) {
	mockMerchantRepo := new(MockMerchantRepository)
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewMerchantUsecase(mockMerchantRepo, mockUserRepo)

	userID := uuid.New()
	tcases := []struct {
		status  entities.MerchantStatus
		message string
	}{
		{status: entities.MerchantStatusPending, message: "Your merchant application is under review"},
		{status: entities.MerchantStatusSuspended, message: "Your merchant account has been suspended"},
		{status: entities.MerchantStatusRejected, message: "Your merchant application was rejected"},
	}

	for _, tc := range tcases {
		mockMerchantRepo.ExpectedCalls = nil
		mockMerchantRepo.Calls = nil
		mockMerchantRepo.On("GetByUserID", context.Background(), userID).Return(&entities.Merchant{
			ID:           uuid.New(),
			UserID:       userID,
			BusinessName: "Biz",
			MerchantType: entities.MerchantTypeRetail,
			Status:       tc.status,
		}, nil).Once()

		resp, err := uc.GetMerchantStatus(context.Background(), userID)
		assert.NoError(t, err)
		assert.Equal(t, tc.message, resp.Message)
	}
}
