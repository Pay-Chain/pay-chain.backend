package usecases

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/domain/errors"
	domainRepos "pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/repositories"
	"pay-chain.backend/pkg/utils"
)

const (
// Removed PaymentRequestExpiryMinutes (moved to constants.go)
)

type PaymentRequestUsecase struct {
	paymentRequestRepo *repositories.PaymentRequestRepositoryImpl
	merchantRepo       *repositories.MerchantRepository
	walletRepo         *repositories.WalletRepository
	contractRepo       domainRepos.SmartContractRepository
}

func NewPaymentRequestUsecase(
	paymentRequestRepo *repositories.PaymentRequestRepositoryImpl,
	merchantRepo *repositories.MerchantRepository,
	walletRepo *repositories.WalletRepository,
	contractRepo domainRepos.SmartContractRepository,
) *PaymentRequestUsecase {
	return &PaymentRequestUsecase{
		paymentRequestRepo: paymentRequestRepo,
		merchantRepo:       merchantRepo,
		walletRepo:         walletRepo,
		contractRepo:       contractRepo,
	}
}

type CreatePaymentRequestInput struct {
	UserID       uuid.UUID
	ChainID      string // CAIP-2 format
	TokenAddress string
	Amount       string // Human readable amount (e.g., "100.00")
	Decimals     int
	Description  string
}

type CreatePaymentRequestOutput struct {
	RequestID     string                         `json:"requestId"`
	TxData        *entities.PaymentRequestTxData `json:"txData"`
	ExpiresAt     time.Time                      `json:"expiresAt"`
	ExpiresInSecs int                            `json:"expiresInSeconds"`
}

func (uc *PaymentRequestUsecase) CreatePaymentRequest(ctx context.Context, input CreatePaymentRequestInput) (*CreatePaymentRequestOutput, error) {
	// Get merchant by user ID
	merchant, err := uc.merchantRepo.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, errors.NotFound("merchant not found, please apply as merchant first")
	}

	if merchant.Status != entities.MerchantStatusActive {
		return nil, errors.BadRequest("merchant account is not active")
	}

	// Get merchant's wallet - prefer primary wallet
	wallets, err := uc.walletRepo.GetByUserID(ctx, input.UserID)
	if err != nil || len(wallets) == 0 {
		return nil, errors.NotFound("no wallet found for this merchant")
	}

	// Use primary wallet or first available
	var targetWallet *entities.Wallet
	for _, w := range wallets {
		if w.IsPrimary {
			targetWallet = w
			break
		}
	}
	if targetWallet == nil {
		targetWallet = wallets[0]
	}

	// Parse ChainID
	chainID, err := uuid.Parse(input.ChainID)
	if err != nil {
		return nil, errors.BadRequest("invalid chain id format")
	}

	// Get smart contract for this chain
	contracts, _, err := uc.contractRepo.GetByChain(ctx, chainID, utils.PaginationParams{Page: 1, Limit: 1})
	var contract *entities.SmartContract
	if len(contracts) > 0 {
		contract = contracts[0]
	}

	// Convert human readable amount to smallest unit
	amountInSmallestUnit := convertToSmallestUnit(input.Amount, input.Decimals)

	// Create payment request
	requestID := uuid.New()
	expiresAt := time.Now().Add(PaymentRequestExpiryMinutes * time.Minute)

	paymentRequest := &entities.PaymentRequest{
		ID:           requestID,
		MerchantID:   merchant.ID,
		WalletID:     targetWallet.ID,
		ChainID:      chainID,
		NetworkID:    input.ChainID, // Store original input as NetworkID (CAIP-2 or ID)
		TokenAddress: input.TokenAddress,
		Amount:       amountInSmallestUnit,
		Decimals:     input.Decimals,
		Description:  input.Description,
		Status:       entities.PaymentRequestStatusPending,
		ExpiresAt:    expiresAt,
	}

	if err := uc.paymentRequestRepo.Create(ctx, paymentRequest); err != nil {
		return nil, errors.InternalError(err)
	}

	// Build transaction data
	txData := uc.buildTransactionData(paymentRequest, contract)

	return &CreatePaymentRequestOutput{
		RequestID:     requestID.String(),
		TxData:        txData,
		ExpiresAt:     expiresAt,
		ExpiresInSecs: PaymentRequestExpiryMinutes * 60,
	}, nil
}

func (uc *PaymentRequestUsecase) buildTransactionData(
	request *entities.PaymentRequest,
	contract *entities.SmartContract,
) *entities.PaymentRequestTxData {
	txData := &entities.PaymentRequestTxData{
		RequestID: request.ID.String(),
		ChainID:   request.NetworkID, // external tx data needs network ID
		Amount:    request.Amount,
		Decimals:  request.Decimals,
	}

	if contract != nil {
		txData.ContractAddress = contract.ContractAddress
	}

	// Determine chain type from CAIP-2
	chainType := getChainTypeFromCAIP2(request.NetworkID)

	switch chainType {
	case "eip155": // EVM
		txData.Hex = uc.buildEvmTransactionHex(request)
	case "solana": // SVM
		txData.Base64 = uc.buildSvmTransactionBase64(request)
	}

	return txData
}

func (uc *PaymentRequestUsecase) buildEvmTransactionHex(
	request *entities.PaymentRequest,
) string {
	// Function selector defined in evm_constants.go
	functionSelector := PayRequestSelector

	// Encode parameters
	requestIdBytes := padLeft(strings.TrimPrefix(request.ID.String(), "0x"), EVMWordSizeHex)

	// Combine
	calldata := functionSelector + requestIdBytes

	return calldata
}

func (uc *PaymentRequestUsecase) buildSvmTransactionBase64(
	request *entities.PaymentRequest,
) string {
	data := map[string]interface{}{
		"instruction": "pay_request",
		"request_id":  request.ID.String(),
	}

	jsonData, _ := json.Marshal(data)
	return hex.EncodeToString(jsonData)
}

func (uc *PaymentRequestUsecase) GetPaymentRequest(ctx context.Context, requestID uuid.UUID) (*entities.PaymentRequest, *entities.PaymentRequestTxData, error) {
	request, err := uc.paymentRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, nil, errors.NotFound("payment request not found")
	}

	// Check if expired
	if request.Status == entities.PaymentRequestStatusPending && time.Now().After(request.ExpiresAt) {
		request.Status = entities.PaymentRequestStatusExpired
		_ = uc.paymentRequestRepo.UpdateStatus(ctx, requestID, entities.PaymentRequestStatusExpired)
	}

	// Get contract
	contracts, _, err := uc.contractRepo.GetByChain(ctx, request.ChainID, utils.PaginationParams{Page: 1, Limit: 1})
	var contract *entities.SmartContract
	if len(contracts) > 0 {
		contract = contracts[0]
	}

	// Get wallet
	_, _ = uc.walletRepo.GetByID(ctx, request.WalletID)

	txData := uc.buildTransactionData(request, contract)

	return request, txData, nil
}

func (uc *PaymentRequestUsecase) ListPaymentRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error) {
	merchant, err := uc.merchantRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, 0, errors.NotFound("merchant not found")
	}

	return uc.paymentRequestRepo.GetByMerchantID(ctx, merchant.ID, limit, offset)
}

func (uc *PaymentRequestUsecase) MarkPaymentCompleted(ctx context.Context, requestID uuid.UUID, txHash string) error {
	return uc.paymentRequestRepo.MarkCompleted(ctx, requestID, txHash)
}

// Helper functions
