package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/domain/errors"
	domainRepos "pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/pkg/utils"
)

const (
// Removed PaymentRequestExpiryMinutes (moved to constants.go)
)

type PaymentRequestUsecase struct {
	paymentRequestRepo domainRepos.PaymentRequestRepository
	merchantRepo       domainRepos.MerchantRepository
	walletRepo         domainRepos.WalletRepository
	chainRepo          domainRepos.ChainRepository
	contractRepo       domainRepos.SmartContractRepository
	tokenRepo          domainRepos.TokenRepository
	chainResolver      *ChainResolver
}

func NewPaymentRequestUsecase(
	paymentRequestRepo domainRepos.PaymentRequestRepository,
	merchantRepo domainRepos.MerchantRepository,
	walletRepo domainRepos.WalletRepository,
	chainRepo domainRepos.ChainRepository,
	contractRepo domainRepos.SmartContractRepository,
	tokenRepo domainRepos.TokenRepository,
) *PaymentRequestUsecase {
	return &PaymentRequestUsecase{
		paymentRequestRepo: paymentRequestRepo,
		merchantRepo:       merchantRepo,
		walletRepo:         walletRepo,
		chainRepo:          chainRepo,
		contractRepo:       contractRepo,
		tokenRepo:          tokenRepo,
		chainResolver:      NewChainResolver(chainRepo),
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

	chainUUID, caip2ID, err := uc.chainResolver.ResolveFromAny(ctx, input.ChainID)
	if err != nil {
		return nil, errors.BadRequest("invalid chain id format")
	}
	token, err := uc.tokenRepo.GetByAddress(ctx, input.TokenAddress, chainUUID)
	if err != nil {
		if input.TokenAddress == "" || input.TokenAddress == "0x0000000000000000000000000000000000000000" || input.TokenAddress == "native" {
			token, err = uc.tokenRepo.GetNative(ctx, chainUUID)
		}
		if err != nil {
			return nil, errors.BadRequest("invalid token for selected chain")
		}
	}

	contract, _ := uc.contractRepo.GetActiveContract(ctx, chainUUID, entities.ContractTypeGateway)

	decimals := token.Decimals
	if input.Decimals > 0 && input.Decimals != decimals {
		return nil, errors.BadRequest("token decimals mismatch")
	}

	// Convert human readable amount to smallest unit
	amountInSmallestUnit, convErr := convertToSmallestUnit(input.Amount, decimals)
	if convErr != nil {
		return nil, errors.BadRequest(convErr.Error())
	}

	// Create payment request
	requestID := utils.GenerateUUIDv7()
	expiresAt := time.Now().Add(PaymentRequestExpiryMinutes * time.Minute)

	paymentRequest := &entities.PaymentRequest{
		ID:            requestID,
		MerchantID:    merchant.ID,
		ChainID:       chainUUID,
		NetworkID:     caip2ID,
		TokenID:       token.ID,
		TokenAddress:  input.TokenAddress,
		WalletAddress: targetWallet.Address,
		Amount:        amountInSmallestUnit,
		Decimals:      decimals,
		Description:   input.Description,
		Status:        entities.PaymentRequestStatusPending,
		ExpiresAt:     expiresAt,
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
		txData.To = contract.ContractAddress
		txData.ProgramID = contract.ContractAddress
	}

	// Determine chain type from CAIP-2
	chainType := getChainTypeFromCAIP2(request.NetworkID)

	switch chainType {
	case "eip155": // EVM
		txData.Hex = uc.buildEvmTransactionHex(request)
	case "solana": // SVM
		txData.Base58 = uc.buildSvmTransactionBase58(request)
	}

	return txData
}

func (uc *PaymentRequestUsecase) buildEvmTransactionHex(
	request *entities.PaymentRequest,
) string {
	// Function selector defined in evm_constants.go
	functionSelector := PayRequestSelector

	// Encode parameters
	requestIdBytes := uuidToBytes32Hex(request.ID)

	// Combine
	calldata := functionSelector + requestIdBytes

	return calldata
}

func (uc *PaymentRequestUsecase) buildSvmTransactionBase58(
	request *entities.PaymentRequest,
) string {
	// Anchor instruction layout: 8-byte discriminator + args
	// pay_request(request_id: [u8;32])
	discriminator := anchorDiscriminator("pay_request")
	requestID := uuidToBytes32(request.ID)

	data := make([]byte, 0, 40)
	data = append(data, discriminator[:]...)
	data = append(data, requestID[:]...)
	return base58Encode(data)
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
	chainID := request.ChainID
	if chainID == uuid.Nil && request.NetworkID != "" {
		if cid, _, resolveErr := uc.chainResolver.ResolveFromAny(ctx, request.NetworkID); resolveErr == nil {
			chainID = cid
			request.ChainID = cid
		}
	}
	if request.NetworkID == "" && chainID != uuid.Nil {
		if chain, chainErr := uc.chainRepo.GetByID(ctx, chainID); chainErr == nil && chain != nil {
			request.NetworkID = chain.GetCAIP2ID()
		}
	}
	contract, _ := uc.contractRepo.GetActiveContract(ctx, chainID, entities.ContractTypeGateway)

	// Get wallet (by address now)
	// _, _ = uc.walletRepo.GetByID(ctx, request.WalletID) // WalletID removed

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
