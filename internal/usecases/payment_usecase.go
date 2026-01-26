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
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
)

// PaymentUsecase handles payment business logic
type PaymentUsecase struct {
	paymentRepo      repositories.PaymentRepository
	paymentEventRepo repositories.PaymentEventRepository
	walletRepo       repositories.WalletRepository
	merchantRepo     repositories.MerchantRepository
	contractRepo     repositories.SmartContractRepository
}

// NewPaymentUsecase creates a new payment usecase
func NewPaymentUsecase(
	paymentRepo repositories.PaymentRepository,
	paymentEventRepo repositories.PaymentEventRepository,
	walletRepo repositories.WalletRepository,
	merchantRepo repositories.MerchantRepository,
	contractRepo repositories.SmartContractRepository,
) *PaymentUsecase {
	return &PaymentUsecase{
		paymentRepo:      paymentRepo,
		paymentEventRepo: paymentEventRepo,
		walletRepo:       walletRepo,
		merchantRepo:     merchantRepo,
		contractRepo:     contractRepo,
	}
}

// FeeConfig holds fee configuration
type FeeConfig struct {
	BaseFeeCents     int64   // Base fee in cents (e.g., 50 = $0.50)
	PercentageFee    float64 // Percentage fee (e.g., 0.003 = 0.3%)
	BridgeFeeFlat    int64   // Flat bridge fee in cents
	MerchantDiscount float64 // Merchant discount percentage
}

// DefaultFeeConfig returns default fee configuration
func DefaultFeeConfig() *FeeConfig {
	return &FeeConfig{
		BaseFeeCents:  int64(DefaultFixedFeeUSD * 100),   // $0.50
		PercentageFee: DefaultPercentageFee,              // 0.3%
		BridgeFeeFlat: int64(DefaultBridgeFeeFlat * 100), // $0.10
	}
}

// CalculateFees calculates fees for a payment
func (u *PaymentUsecase) CalculateFees(amount *big.Int, decimals int, isCrossChain bool, merchantDiscount float64) *entities.FeeBreakdown {
	// Convert amount to float for calculation
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	amountFloat, _ := new(big.Float).SetInt(amount).Float64()
	amountFloat = amountFloat / float64(divisor.Acc())

	config := DefaultFeeConfig()

	// Platform fee: base + percentage
	platformFee := float64(config.BaseFeeCents)/100 + amountFloat*config.PercentageFee

	// Apply merchant discount
	if merchantDiscount > 0 {
		platformFee = platformFee * (1 - merchantDiscount)
	}

	// Bridge fee (only for cross-chain)
	bridgeFee := float64(0)
	if isCrossChain {
		bridgeFee = float64(config.BridgeFeeFlat) / 100
	}

	totalFee := platformFee + bridgeFee
	netAmount := amountFloat - totalFee

	return &entities.FeeBreakdown{
		PlatformFee: formatAmount(platformFee, decimals),
		BridgeFee:   formatAmount(bridgeFee, decimals),
		GasFee:      "0", // Gas is handled separately
		TotalFee:    formatAmount(totalFee, decimals),
		NetAmount:   formatAmount(netAmount, decimals),
	}
}

// SelectBridge selects the best bridge for cross-chain transfer
func (u *PaymentUsecase) SelectBridge(sourceChainID, destChainID string) string {
	// Bridge selection logic based on chain types
	// Priority: CCIP > Hyperlane > Hyperbridge

	// For EVM chains, prefer CCIP
	if isEVMChain(sourceChainID) && isEVMChain(destChainID) {
		return "CCIP"
	}

	// For Solana <-> EVM, use Hyperlane
	if (isSolanaChain(sourceChainID) && isEVMChain(destChainID)) ||
		(isEVMChain(sourceChainID) && isSolanaChain(destChainID)) {
		return "Hyperlane"
	}

	// Default to Hyperbridge
	return "Hyperbridge"
}

// CreatePayment creates a new payment
func (u *PaymentUsecase) CreatePayment(ctx context.Context, userID uuid.UUID, input *entities.CreatePaymentInput) (*entities.CreatePaymentResponse, error) {
	// Validate input
	if input.SourceChainID == "" || input.DestChainID == "" {
		return nil, domainerrors.ErrBadRequest
	}
	if input.ReceiverAddress == "" {
		return nil, domainerrors.ErrBadRequest
	}

	// Parse amount
	amount := new(big.Int)
	if _, ok := amount.SetString(input.Amount, 10); !ok {
		return nil, domainerrors.ErrBadRequest
	}

	// Calculate fees
	isCrossChain := input.SourceChainID != input.DestChainID
	feeBreakdown := u.CalculateFees(amount, input.Decimals, isCrossChain, 0)

	// Select bridge
	bridgeType := ""
	if isCrossChain {
		bridgeType = u.SelectBridge(input.SourceChainID, input.DestChainID)
	}

	// Get smart contract for source chain from database
	contracts, _ := u.contractRepo.GetByChain(ctx, input.SourceChainID)
	var contract *entities.SmartContract
	if len(contracts) > 0 {
		contract = contracts[0]
	}

	// Create payment entity
	payment := &entities.Payment{
		SenderID:           userID,
		SourceChainID:      input.SourceChainID,
		DestChainID:        input.DestChainID,
		SourceTokenAddress: input.SourceTokenAddress,
		DestTokenAddress:   input.DestTokenAddress,
		SourceAmount:       input.Amount,
		FeeAmount:          feeBreakdown.TotalFee,
		BridgeType:         bridgeType,
		ReceiverAddress:    input.ReceiverAddress,
		Decimals:           input.Decimals,
		Status:             entities.PaymentStatusPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	// Set nullable DestAmount
	payment.DestAmount.SetValid(feeBreakdown.NetAmount)

	// Save payment
	if err := u.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	// Create initial event
	event := &entities.PaymentEvent{
		PaymentID: payment.ID,
		EventType: "CREATED",
		Chain:     "source",
		CreatedAt: time.Now(),
	}
	_ = u.paymentEventRepo.Create(ctx, event)

	// Build transaction data using metadata from DB
	signatureData := u.buildTransactionData(payment, contract)

	return &entities.CreatePaymentResponse{
		PaymentID:      payment.ID,
		Status:         payment.Status,
		SourceChainID:  payment.SourceChainID,
		DestChainID:    payment.DestChainID,
		SourceAmount:   payment.SourceAmount,
		SourceDecimals: payment.Decimals,
		DestAmount:     payment.DestAmount.String,
		FeeAmount:      payment.FeeAmount,
		BridgeType:     payment.BridgeType,
		FeeBreakdown:   *feeBreakdown,
		ExpiresAt:      time.Now().Add(PaymentExpiryDuration),
		SignatureData:  signatureData,
	}, nil
}

// buildTransactionData builds transaction data for frontend based on database metadata
func (u *PaymentUsecase) buildTransactionData(payment *entities.Payment, contract *entities.SmartContract) interface{} {
	if contract == nil {
		return nil
	}

	chainType := getChainTypeFromCAIP2(payment.SourceChainID)

	switch chainType {
	case "eip155":
		return map[string]string{
			"to":   contract.ContractAddress,
			"data": u.buildEvmPaymentHex(payment),
		}
	case "solana":
		return map[string]string{
			"programId": contract.ContractAddress,
			"data":      u.buildSvmPaymentBase64(payment),
		}
	}

	return nil
}

func (u *PaymentUsecase) buildEvmPaymentHex(payment *entities.Payment) string {
	// function createPayment(bytes destChainId, bytes receiver, address sourceToken, address destToken, uint256 amount)
	// selector defined in evm_constants.go
	selector := CreatePaymentSelector

	// Prepare dynamic data
	destChainId := []byte(payment.DestChainID)
	// Receiver can be EVM address (20 bytes) or SVM address (32 bytes)
	// If it starts with 0x and is 42 chars, it's EVM.
	var receiver []byte
	if strings.HasPrefix(payment.ReceiverAddress, "0x") && len(payment.ReceiverAddress) == 42 {
		addrBytes, _ := hex.DecodeString(strings.TrimPrefix(payment.ReceiverAddress, "0x"))
		receiver = addrBytes
	} else {
		// Assume Solana or other raw bytes
		receiver = []byte(payment.ReceiverAddress)
	}

	// ABI Encoding offsets (offsets are from the start of the data area, 5 * EVMWordSize = 160 bytes)
	offsetDestChainId := 5 * EVMWordSize
	lenDestChainIdPadded := ((len(destChainId) + (EVMWordSize - 1)) / EVMWordSize) * EVMWordSize
	offsetReceiver := offsetDestChainId + EVMWordSize + lenDestChainIdPadded

	// Static parts
	destChainIdOffsetHex := padLeft(fmt.Sprintf("%x", offsetDestChainId), EVMWordSizeHex)
	receiverOffsetHex := padLeft(fmt.Sprintf("%x", offsetReceiver), EVMWordSizeHex)
	sourceTokenHex := padLeft(strings.TrimPrefix(payment.SourceTokenAddress, "0x"), EVMWordSizeHex)
	destTokenHex := padLeft(strings.TrimPrefix(payment.DestTokenAddress, "0x"), EVMWordSizeHex)

	amount := new(big.Int)
	amount.SetString(payment.SourceAmount, 10)
	amountHex := padLeft(fmt.Sprintf("%x", amount), EVMWordSizeHex)

	// Dynamic parts
	destChainIdLenHex := padLeft(fmt.Sprintf("%x", len(destChainId)), EVMWordSizeHex)
	destChainIdDataHex := padRight(hex.EncodeToString(destChainId), lenDestChainIdPadded*2)

	receiverLenHex := padLeft(fmt.Sprintf("%x", len(receiver)), EVMWordSizeHex)
	receiverDataHex := padRight(hex.EncodeToString(receiver), ((len(receiver)+(EVMWordSize-1))/EVMWordSize)*EVMWordSize*2)

	return selector + destChainIdOffsetHex + receiverOffsetHex + sourceTokenHex + destTokenHex + amountHex +
		destChainIdLenHex + destChainIdDataHex + receiverLenHex + receiverDataHex
}

func (u *PaymentUsecase) buildSvmPaymentBase64(payment *entities.Payment) string {
	data := map[string]interface{}{
		"instruction":   "create_payment",
		"payment_id":    payment.ID.String(),
		"dest_chain_id": payment.DestChainID,
		"dest_token":    payment.DestTokenAddress,
		"amount":        payment.SourceAmount,
		"receiver":      payment.ReceiverAddress,
	}
	jsonData, _ := json.Marshal(data)
	return hex.EncodeToString(jsonData)
}

// GetPayment gets a payment by ID
func (u *PaymentUsecase) GetPayment(ctx context.Context, paymentID uuid.UUID) (*entities.Payment, error) {
	return u.paymentRepo.GetByID(ctx, paymentID)
}

// GetPaymentsByUser gets payments for a user
func (u *PaymentUsecase) GetPaymentsByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*entities.Payment, int, error) {
	offset := (page - 1) * limit
	return u.paymentRepo.GetByUserID(ctx, userID, limit, offset)
}

// GetPaymentEvents gets events for a payment
func (u *PaymentUsecase) GetPaymentEvents(ctx context.Context, paymentID uuid.UUID) ([]*entities.PaymentEvent, error) {
	return u.paymentEventRepo.GetByPaymentID(ctx, paymentID)
}

// UpdatePaymentStatus updates a payment's status
func (u *PaymentUsecase) UpdatePaymentStatus(ctx context.Context, paymentID uuid.UUID, status entities.PaymentStatus) error {
	return u.paymentRepo.UpdateStatus(ctx, paymentID, status)
}
