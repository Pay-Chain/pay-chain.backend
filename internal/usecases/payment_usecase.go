package usecases

import (
	"context"
	"math/big"

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
}

// NewPaymentUsecase creates a new payment usecase
func NewPaymentUsecase(
	paymentRepo repositories.PaymentRepository,
	paymentEventRepo repositories.PaymentEventRepository,
	walletRepo repositories.WalletRepository,
	merchantRepo repositories.MerchantRepository,
) *PaymentUsecase {
	return &PaymentUsecase{
		paymentRepo:      paymentRepo,
		paymentEventRepo: paymentEventRepo,
		walletRepo:       walletRepo,
		merchantRepo:     merchantRepo,
	}
}

// FeeConfig holds fee configuration
type FeeConfig struct {
	BaseFeeCents    int64   // Base fee in cents (e.g., 50 = $0.50)
	PercentageFee   float64 // Percentage fee (e.g., 0.003 = 0.3%)
	BridgeFeeFlat   int64   // Flat bridge fee in cents
	MerchantDiscount float64 // Merchant discount percentage
}

// DefaultFeeConfig returns default fee configuration
func DefaultFeeConfig() *FeeConfig {
	return &FeeConfig{
		BaseFeeCents:  50,    // $0.50
		PercentageFee: 0.003, // 0.3%
		BridgeFeeFlat: 10,    // $0.10
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
		PlatformFee: platformFee,
		BridgeFee:   bridgeFee,
		TotalFee:    totalFee,
		NetAmount:   netAmount,
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

	// Create payment entity
	payment := &entities.Payment{
		SenderID:           userID,
		SourceChainID:      input.SourceChainID,
		DestChainID:        input.DestChainID,
		SourceTokenAddress: input.SourceTokenAddress,
		DestTokenAddress:   input.DestTokenAddress,
		SourceAmount:       input.Amount,
		DestAmount:         formatAmount(feeBreakdown.NetAmount, input.Decimals),
		FeeAmount:          formatAmount(feeBreakdown.TotalFee, input.Decimals),
		BridgeType:         bridgeType,
		ReceiverAddress:    input.ReceiverAddress,
		Decimals:           input.Decimals,
		Status:             entities.PaymentStatusPending,
	}

	// Save payment
	if err := u.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	// Create initial event
	event := &entities.PaymentEvent{
		PaymentID: payment.ID,
		EventType: "CREATED",
		Chain:     "source",
	}
	_ = u.paymentEventRepo.Create(ctx, event)

	return &entities.CreatePaymentResponse{
		PaymentID:    payment.ID.String(),
		Status:       string(payment.Status),
		SourceAmount: payment.SourceAmount,
		DestAmount:   payment.DestAmount,
		FeeAmount:    payment.FeeAmount,
		BridgeType:   payment.BridgeType,
		FeeBreakdown: feeBreakdown,
	}, nil
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

// Helper functions
func isEVMChain(chainID string) bool {
	return len(chainID) > 6 && chainID[:6] == "eip155"
}

func isSolanaChain(chainID string) bool {
	return len(chainID) > 6 && chainID[:6] == "solana"
}

func formatAmount(amount float64, decimals int) string {
	// Convert float to string with appropriate precision
	multiplier := new(big.Float).SetFloat64(amount)
	exp := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	result := new(big.Float).Mul(multiplier, exp)
	intResult, _ := result.Int(nil)
	return intResult.String()
}
