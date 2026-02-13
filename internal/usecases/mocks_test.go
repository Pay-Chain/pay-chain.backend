package usecases_test

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

// Mock UnitOfWork
type MockUnitOfWork struct {
	mock.Mock
}

func (m *MockUnitOfWork) Do(ctx context.Context, f func(context.Context) error) error {
	m.Called(ctx, f)
	return f(ctx)
}

func (m *MockUnitOfWork) WithLock(ctx context.Context) context.Context {
	args := m.Called(ctx)
	return args.Get(0).(context.Context) // Return mocked context
}

// Mock PaymentRepository
type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) Create(ctx context.Context, payment *entities.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Payment), args.Error(1)
}

func (m *MockPaymentRepository) Update(ctx context.Context, payment *entities.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PaymentStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Payment, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*entities.Payment), args.Int(1), args.Error(2)
}

func (m *MockPaymentRepository) List(ctx context.Context, limit, offset int) ([]*entities.Payment, int, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*entities.Payment), args.Int(1), args.Error(2)
}

func (m *MockPaymentRepository) UpdateDestTxHash(ctx context.Context, id uuid.UUID, txHash string) error {
	args := m.Called(ctx, id, txHash)
	return args.Error(0)
}

func (m *MockPaymentRepository) MarkRefunded(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByMerchantID(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*entities.Payment, int, error) {
	args := m.Called(ctx, merchantID, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entities.Payment), args.Get(1).(int), args.Error(2)
}

// Mock PaymentEventRepository
type MockPaymentEventRepository struct {
	mock.Mock
}

func (m *MockPaymentEventRepository) Create(ctx context.Context, event *entities.PaymentEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockPaymentEventRepository) GetByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]*entities.PaymentEvent, error) {
	args := m.Called(ctx, paymentID)
	return args.Get(0).([]*entities.PaymentEvent), args.Error(1)
}

func (m *MockPaymentEventRepository) GetLatestByPaymentID(ctx context.Context, paymentID uuid.UUID) (*entities.PaymentEvent, error) {
	args := m.Called(ctx, paymentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.PaymentEvent), args.Error(1)
}

// Mock ChainRepository
type MockChainRepository struct {
	mock.Mock
}

func (m *MockChainRepository) GetByChainID(ctx context.Context, chainID string) (*entities.Chain, error) {
	args := m.Called(ctx, chainID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Chain), args.Error(1)
}
func (m *MockChainRepository) Create(ctx context.Context, chain *entities.Chain) error {
	return m.Called(ctx, chain).Error(0)
}
func (m *MockChainRepository) Update(ctx context.Context, chain *entities.Chain) error {
	return m.Called(ctx, chain).Error(0)
}
func (m *MockChainRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockChainRepository) List(ctx context.Context) ([]*entities.Chain, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Chain), args.Error(1)
}
func (m *MockChainRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Chain, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Chain), args.Error(1)
}

func (m *MockChainRepository) GetActive(ctx context.Context, pagination utils.PaginationParams) ([]*entities.Chain, int64, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entities.Chain), args.Get(1).(int64), args.Error(2)
}

func (m *MockChainRepository) GetAll(ctx context.Context) ([]*entities.Chain, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Chain), args.Error(1)
}

func (m *MockChainRepository) GetByCAIP2(ctx context.Context, caip2 string) (*entities.Chain, error) {
	args := m.Called(ctx, caip2)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Chain), args.Error(1)
}

func (m *MockChainRepository) GetAllRPCs(ctx context.Context, chainID *uuid.UUID, isActive *bool, search *string, pagination utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	args := m.Called(ctx, chainID, isActive, search, pagination)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entities.ChainRPC), args.Get(1).(int64), args.Error(2)
}

// Mock TokenRepository
type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) GetByAddress(ctx context.Context, address string, chainID uuid.UUID) (*entities.Token, error) {
	args := m.Called(ctx, address, chainID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Token), args.Error(1)
}

func (m *MockTokenRepository) GetNative(ctx context.Context, chainID uuid.UUID) (*entities.Token, error) {
	args := m.Called(ctx, chainID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Token), args.Error(1)
}
func (m *MockTokenRepository) ListSupported(ctx context.Context) ([]*entities.Token, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Token), args.Error(1)
}
func (m *MockTokenRepository) ListStablecoins(ctx context.Context) ([]*entities.Token, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Token), args.Error(1)
}
func (m *MockTokenRepository) GetBySymbol(ctx context.Context, symbol string, chainID uuid.UUID) (*entities.Token, error) {
	args := m.Called(ctx, symbol, chainID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Token), args.Error(1)
}

func (m *MockTokenRepository) GetStablecoins(ctx context.Context) ([]*entities.Token, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Token), args.Error(1)
}

func (m *MockTokenRepository) GetTokensByChain(ctx context.Context, chainID uuid.UUID, pagination utils.PaginationParams) ([]*entities.Token, int64, error) {
	args := m.Called(ctx, chainID, pagination)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entities.Token), args.Get(1).(int64), args.Error(2)
}

func (m *MockTokenRepository) Create(ctx context.Context, token *entities.Token) error {
	return m.Called(ctx, token).Error(0)
}
func (m *MockTokenRepository) Update(ctx context.Context, token *entities.Token) error {
	return m.Called(ctx, token).Error(0)
}
func (m *MockTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockTokenRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Token, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Token), args.Error(1)
}

func (m *MockTokenRepository) GetAll(ctx context.Context) ([]*entities.Token, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Token), args.Error(1)
}

func (m *MockTokenRepository) GetAllTokens(ctx context.Context, chainID *uuid.UUID, search *string, pagination utils.PaginationParams) ([]*entities.Token, int64, error) {
	args := m.Called(ctx, chainID, search, pagination)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entities.Token), args.Get(1).(int64), args.Error(2)
}

func (m *MockTokenRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// Mock SmartContractRepository
type MockSmartContractRepository struct {
	mock.Mock
}

func (m *MockSmartContractRepository) GetActiveContract(ctx context.Context, chainID uuid.UUID, type_ entities.SmartContractType) (*entities.SmartContract, error) {
	args := m.Called(ctx, chainID, type_)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.SmartContract), args.Error(1)
}

func (m *MockSmartContractRepository) GetByChainAndAddress(ctx context.Context, chainID uuid.UUID, address string) (*entities.SmartContract, error) {
	args := m.Called(ctx, chainID, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.SmartContract), args.Error(1)
}
func (m *MockSmartContractRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.SmartContract, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.SmartContract), args.Error(1)
}

func (m *MockSmartContractRepository) GetAll(ctx context.Context, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entities.SmartContract), args.Get(1).(int64), args.Error(2)
}

func (m *MockSmartContractRepository) GetByChain(ctx context.Context, chainID uuid.UUID, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	args := m.Called(ctx, chainID, pagination)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entities.SmartContract), args.Get(1).(int64), args.Error(2)
}

func (m *MockSmartContractRepository) GetFiltered(ctx context.Context, chainID *uuid.UUID, contractType entities.SmartContractType, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	args := m.Called(ctx, chainID, contractType, pagination)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entities.SmartContract), args.Get(1).(int64), args.Error(2)
}

func (m *MockSmartContractRepository) Create(ctx context.Context, contract *entities.SmartContract) error {
	return m.Called(ctx, contract).Error(0)
}
func (m *MockSmartContractRepository) Update(ctx context.Context, contract *entities.SmartContract) error {
	return m.Called(ctx, contract).Error(0)
}
func (m *MockSmartContractRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockSmartContractRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// Mock MerchantRepository
type MockMerchantRepository struct {
	mock.Mock
}

func (m *MockMerchantRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Merchant), args.Error(1)
}
func (m *MockMerchantRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.Merchant, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Merchant), args.Error(1)
}
func (m *MockMerchantRepository) Create(ctx context.Context, merchant *entities.Merchant) error {
	return m.Called(ctx, merchant).Error(0)
}
func (m *MockMerchantRepository) Update(ctx context.Context, merchant *entities.Merchant) error {
	return m.Called(ctx, merchant).Error(0)
}

func (m *MockMerchantRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.MerchantStatus) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *MockMerchantRepository) List(ctx context.Context) ([]*entities.Merchant, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// Mock WalletRepository
type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) GetByAddress(ctx context.Context, chainID uuid.UUID, address string) (*entities.Wallet, error) {
	args := m.Called(ctx, chainID, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Wallet), args.Error(1)
}
func (m *MockWalletRepository) Create(ctx context.Context, wallet *entities.Wallet) error {
	return m.Called(ctx, wallet).Error(0)
}
func (m *MockWalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Wallet, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*entities.Wallet), args.Error(1)
}
func (m *MockWalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Wallet, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Wallet), args.Error(1)
}
func (m *MockWalletRepository) Update(ctx context.Context, wallet *entities.Wallet) error {
	return m.Called(ctx, wallet).Error(0)
}
func (m *MockWalletRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockWalletRepository) SetPrimary(ctx context.Context, userID, walletID uuid.UUID) error {
	return m.Called(ctx, userID, walletID).Error(0)
}

func (m *MockWalletRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// Mock PaymentRequestRepository
type MockPaymentRequestRepository struct {
	mock.Mock
}

func (m *MockPaymentRequestRepository) Create(ctx context.Context, request *entities.PaymentRequest) error {
	return m.Called(ctx, request).Error(0)
}
func (m *MockPaymentRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.PaymentRequest, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.PaymentRequest), args.Error(1)
}
func (m *MockPaymentRequestRepository) GetByMerchantID(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error) {
	args := m.Called(ctx, merchantID, limit, offset)
	return args.Get(0).([]*entities.PaymentRequest), args.Int(1), args.Error(2)
}
func (m *MockPaymentRequestRepository) MarkCompleted(ctx context.Context, id uuid.UUID, txHash string) error {
	return m.Called(ctx, id, txHash).Error(0)
}
func (m *MockPaymentRequestRepository) GetExpiredPending(ctx context.Context, limit int) ([]*entities.PaymentRequest, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*entities.PaymentRequest), args.Error(1)
}
func (m *MockPaymentRequestRepository) ExpireRequests(ctx context.Context, ids []uuid.UUID) error {
	return m.Called(ctx, ids).Error(0)
}

// Mock UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *entities.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *entities.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return m.Called(ctx, id, passwordHash).Error(0)
}

func (m *MockUserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, search string) ([]*entities.User, error) {
	args := m.Called(ctx, search)
	return args.Get(0).([]*entities.User), args.Error(1)
}

// Mock ApiKeyRepository
type MockApiKeyRepository struct {
	mock.Mock
}

func (m *MockApiKeyRepository) Create(ctx context.Context, apiKey *entities.ApiKey) error {
	return m.Called(ctx, apiKey).Error(0)
}

func (m *MockApiKeyRepository) FindByKeyHash(ctx context.Context, keyHash string) (*entities.ApiKey, error) {
	args := m.Called(ctx, keyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ApiKey), args.Error(1)
}

func (m *MockApiKeyRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.ApiKey, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*entities.ApiKey), args.Error(1)
}

func (m *MockApiKeyRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.ApiKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ApiKey), args.Error(1)
}

func (m *MockApiKeyRepository) Update(ctx context.Context, apiKey *entities.ApiKey) error {
	return m.Called(ctx, apiKey).Error(0)
}

func (m *MockApiKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
