package usecases

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/repositories"
)

const (
	PaymentRequestExpiryMinutes = 15
)

type PaymentRequestUsecase struct {
	paymentRequestRepo *repositories.PaymentRequestRepositoryImpl
	merchantRepo       *repositories.MerchantRepository
	walletRepo         *repositories.WalletRepository
	contractRepo       *repositories.SmartContractRepository
}

func NewPaymentRequestUsecase(
	paymentRequestRepo *repositories.PaymentRequestRepositoryImpl,
	merchantRepo *repositories.MerchantRepository,
	walletRepo *repositories.WalletRepository,
	contractRepo *repositories.SmartContractRepository,
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

	// Get smart contract for this chain
	contracts, _ := uc.contractRepo.GetByChain(ctx, input.ChainID)
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
		ChainID:      input.ChainID,
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
	txData := uc.buildTransactionData(paymentRequest, contract, targetWallet.Address)

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
	receiverAddress string,
) *entities.PaymentRequestTxData {
	txData := &entities.PaymentRequestTxData{
		RequestID: request.ID.String(),
		ChainID:   request.ChainID,
		Amount:    request.Amount,
		Decimals:  request.Decimals,
	}

	if contract != nil {
		txData.ContractAddress = contract.ContractAddress
	}

	// Determine chain type from CAIP-2
	chainType := getChainTypeFromCAIP2(request.ChainID)

	switch chainType {
	case "eip155": // EVM
		txData.Hex = uc.buildEvmTransactionHex(request, contract, receiverAddress)
	case "solana": // SVM
		txData.Base64 = uc.buildSvmTransactionBase64(request, contract, receiverAddress)
	}

	return txData
}

func (uc *PaymentRequestUsecase) buildEvmTransactionHex(
	request *entities.PaymentRequest,
	contract *entities.SmartContract,
	receiverAddress string,
) string {
	// Function selector for payRequest(bytes32 requestId, address token, uint256 amount, address receiver)
	// keccak256("payRequest(bytes32,address,uint256,address)")[:4]
	functionSelector := "0x8a4068dd"

	// Encode parameters
	requestIdBytes := padLeft(strings.TrimPrefix(request.ID.String(), "0x"), 64)

	// Token address (remove 0x prefix and pad)
	tokenAddress := padLeft(strings.TrimPrefix(request.TokenAddress, "0x"), 64)

	// Amount (convert to hex and pad)
	amount := new(big.Int)
	amount.SetString(request.Amount, 10)
	amountHex := padLeft(fmt.Sprintf("%x", amount), 64)

	// Receiver address
	receiver := padLeft(strings.TrimPrefix(receiverAddress, "0x"), 64)

	// Combine
	calldata := functionSelector + requestIdBytes + tokenAddress + amountHex + receiver

	return calldata
}

func (uc *PaymentRequestUsecase) buildSvmTransactionBase64(
	request *entities.PaymentRequest,
	contract *entities.SmartContract,
	receiverAddress string,
) string {
	// For Solana, we need to build a serialized transaction
	// This is a simplified version - actual implementation would use Solana SDK

	instructionData := map[string]interface{}{
		"instruction": "pay_request",
		"request_id":  request.ID.String(),
		"token":       request.TokenAddress,
		"amount":      request.Amount,
		"receiver":    receiverAddress,
	}

	jsonData, _ := json.Marshal(instructionData)

	// Base64 encode the instruction data
	// In production, this would be a properly serialized Solana transaction
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
	contracts, _ := uc.contractRepo.GetByChain(ctx, request.ChainID)
	var contract *entities.SmartContract
	if len(contracts) > 0 {
		contract = contracts[0]
	}

	// Get wallet
	wallet, _ := uc.walletRepo.GetByID(ctx, request.WalletID)
	receiverAddress := ""
	if wallet != nil {
		receiverAddress = wallet.Address
	}

	txData := uc.buildTransactionData(request, contract, receiverAddress)

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
func getChainTypeFromCAIP2(caip2 string) string {
	parts := strings.Split(caip2, ":")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func convertToSmallestUnit(amount string, decimals int) string {
	// Simple conversion - in production use proper decimal library
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)

	// Parse amount as float then convert
	amountFloat := new(big.Float)
	amountFloat.SetString(amount)

	multiplierFloat := new(big.Float).SetInt(multiplier)
	result := new(big.Float).Mul(amountFloat, multiplierFloat)

	resultInt, _ := result.Int(nil)
	return resultInt.String()
}

func padLeft(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return strings.Repeat("0", length-len(s)) + s
}
