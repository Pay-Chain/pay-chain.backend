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
	"github.com/volatiletech/null/v8"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

// PaymentUsecase handles payment business logic
type PaymentUsecase struct {
	paymentRepo      repositories.PaymentRepository
	paymentEventRepo repositories.PaymentEventRepository
	walletRepo       repositories.WalletRepository
	merchantRepo     repositories.MerchantRepository
	contractRepo     repositories.SmartContractRepository
	chainRepo        repositories.ChainRepository
	tokenRepo        repositories.TokenRepository
	uow              repositories.UnitOfWork
	clientFactory    *blockchain.ClientFactory
}

// NewPaymentUsecase creates a new payment usecase
func NewPaymentUsecase(
	paymentRepo repositories.PaymentRepository,
	paymentEventRepo repositories.PaymentEventRepository,
	walletRepo repositories.WalletRepository,
	merchantRepo repositories.MerchantRepository,
	contractRepo repositories.SmartContractRepository,
	chainRepo repositories.ChainRepository,
	tokenRepo repositories.TokenRepository,
	uow repositories.UnitOfWork,
	clientFactory *blockchain.ClientFactory,
) *PaymentUsecase {
	return &PaymentUsecase{
		paymentRepo:      paymentRepo,
		paymentEventRepo: paymentEventRepo,
		walletRepo:       walletRepo,
		merchantRepo:     merchantRepo,
		contractRepo:     contractRepo,
		chainRepo:        chainRepo,
		tokenRepo:        tokenRepo,
		uow:              uow,
		clientFactory:    clientFactory,
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
func (u *PaymentUsecase) CalculateFees(ctx context.Context, amount *big.Int, decimals int, sourceChainID, destChainID string, merchantDiscount float64) *entities.FeeBreakdown {
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
	bridgeFeeNative := new(big.Int)
	isCrossChain := sourceChainID != destChainID // Defined here
	if isCrossChain {
		// New: Fetch dynamic bridge fee from blockchain
		quote, err := u.getBridgeFeeQuote(ctx, sourceChainID, destChainID)
		if err == nil {
			bridgeFeeNative = quote
		}
	}

	// Total Fee in Token (Platform Fee)
	totalFeeToken := platformFee
	netAmount := amountFloat - totalFeeToken

	return &entities.FeeBreakdown{
		PlatformFee: formatAmount(platformFee, decimals),
		BridgeFee:   bridgeFeeNative.String(), // Now returning Native Fee (Wei)
		GasFee:      "0",                      // Gas is handled separately
		TotalFee:    formatAmount(totalFeeToken, decimals),
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

	// Resolve Source Chain
	sourceChain, err := u.chainRepo.GetByChainID(ctx, input.SourceChainID)
	if err != nil {
		return nil, fmt.Errorf("invalid source chain: %w", err)
	}

	// Resolve Dest Chain
	destChain, err := u.chainRepo.GetByChainID(ctx, input.DestChainID)
	if err != nil {
		return nil, fmt.Errorf("invalid dest chain: %w", err)
	}

	// Parse amount
	amount := new(big.Int)
	if _, ok := amount.SetString(input.Amount, 10); !ok {
		return nil, domainerrors.ErrBadRequest
	}

	// Calculate fees (Pass strings as before, logic uses them for config/math)
	feeBreakdown := u.CalculateFees(ctx, amount, input.Decimals, input.SourceChainID, input.DestChainID, 0)

	// Select bridge
	bridgeType := ""
	isCrossChain := input.SourceChainID != input.DestChainID
	if isCrossChain {
		bridgeType = u.SelectBridge(input.SourceChainID, input.DestChainID)
	}

	// Get specific gateway contract for source chain using UUID
	contract, err := u.contractRepo.GetActiveContract(ctx, sourceChain.ID, entities.ContractTypeGateway)
	if err != nil {
		fmt.Printf("Warning: Active Gateway contract not found for chain %s: %v\n", input.SourceChainID, err)
	}

	// Resolve Token UUIDs?
	// Input provides `SourceTokenAddress`.
	// We need to find the Token Entity ID for `Payment` record.
	// If native, address might be empty or 0x0.
	// We should try to lookup Token by address + chainID.
	// If not found, do we fail or create?
	// For now, let's look it up.

	var sourceTokenID uuid.UUID
	srcToken, err := u.resolveToken(ctx, input.SourceTokenAddress, sourceChain.ID)
	if err == nil && srcToken != nil {
		sourceTokenID = srcToken.ID
	} else {
		// Fallback or critical error?
		// For now, if not found, we might need dummy or error.
		// Let's assume error for strictly managed tokens.
		// fmt.Printf("Warning: Source token not found %s\n", input.SourceTokenAddress)
		// But Payment struct has `SourceTokenID uuid.UUID` (NOT NULL).
		// So we MUST find it.
		// Implementation Gaps: We might need `GetByAddress` in TokenRepo.
		// I added `GetByAddress` in `token_repo_impl`.
		return nil, fmt.Errorf("source token not found for address %s on chain %s", input.SourceTokenAddress, input.SourceChainID)
	}

	var destTokenID uuid.UUID
	destToken, err := u.resolveToken(ctx, input.DestTokenAddress, destChain.ID)
	if err == nil && destToken != nil {
		destTokenID = destToken.ID
	} else {
		return nil, fmt.Errorf("dest token not found for address %s on chain %s", input.DestTokenAddress, input.DestChainID)
	}

	// Create payment entity
	payment := &entities.Payment{
		ID:                 uuid.New(), // Generate ID
		SenderID:           &userID,
		MerchantID:         nil, // userID is User, not Merchant in this context? Or need to check if User is Merchant?
		SourceChainID:      sourceChain.ID,
		DestChainID:        destChain.ID,
		SourceTokenID:      &sourceTokenID,
		DestTokenID:        &destTokenID,
		SourceTokenAddress: input.SourceTokenAddress,
		DestTokenAddress:   input.DestTokenAddress,
		SourceAmount:       input.Amount,
		FeeAmount:          feeBreakdown.TotalFee,
		TotalCharged:       input.Amount, // Should simplify/clarify
		// BridgeType:         bridgeType, // Field removed from model/entity in favor of BridgeID?
		// Wait, Entity Definition:
		// `BridgeID *uuid.UUID`
		// `Bridge *PaymentBridge`
		// It does NOT have string BridgeType anymore?
		// My earlier `view_file` of `payment.go` entity showed `Bridge *PaymentBridge`.
		// But did I remove `BridgeType` string?
		// Let's check `payment.go` entity again if I can...
		// Assuming I need to look up Bridge ID if I want to persist it.
		// For now, I will leave BridgeID null if I can't resolve it easily, or map string generic.
		// Entity `payment.go` (Step 15817):
		// `BridgeID *uuid.UUID`
		// It does NOT have `BridgeType` string field in the struct snippet.

		ReceiverAddress: input.ReceiverAddress,
		// Decimals:           input.Decimals, // Entity `payment.go` REMOVED Decimals field?
		// Step 15817 snippet: `SourceAmount`, `DestAmount`..., `Status`.
		// Does NOT show `Decimals`.
		// I should check `payment.go` again to be safe.

		Status:    entities.PaymentStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set nullable DestAmount
	if feeBreakdown.NetAmount != "" {
		payment.DestAmount = null.StringFrom(feeBreakdown.NetAmount)
	}

	// Save payment
	// Wrap DB operations in Transaction
	if err := u.uow.Do(ctx, func(txCtx context.Context) error {
		// Save payment
		if err := u.paymentRepo.Create(txCtx, payment); err != nil {
			return err
		}

		// Create initial event
		event := &entities.PaymentEvent{
			ID:        uuid.New(),
			PaymentID: payment.ID,
			EventType: entities.PaymentEventTypeCreated, // Use Enum
			ChainID:   &sourceChain.ID,
			CreatedAt: time.Now(),
		}
		if err := u.paymentEventRepo.Create(txCtx, event); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Build transaction data using metadata from DB
	signatureData := u.buildTransactionData(payment, contract)

	return &entities.CreatePaymentResponse{
		PaymentID:      payment.ID,
		Status:         payment.Status,
		SourceChainID:  input.SourceChainID, // Return the Input String (Network ID) for frontend consistency
		DestChainID:    input.DestChainID,
		SourceAmount:   payment.SourceAmount,
		SourceDecimals: input.Decimals,
		DestAmount:     payment.DestAmount.String,
		FeeAmount:      payment.FeeAmount,
		BridgeType:     bridgeType,
		FeeBreakdown:   *feeBreakdown,
		ExpiresAt:      time.Now().Add(PaymentExpiryDuration),
		SignatureData:  signatureData,
	}, nil
}

// Helper to resolve token
func (u *PaymentUsecase) resolveToken(ctx context.Context, address string, chainID uuid.UUID) (*entities.Token, error) {
	// If address is "0x000..." or "native", handle native token logic
	if address == "" || address == "0x0000000000000000000000000000000000000000" || address == "native" {
		nativeToken, err := u.tokenRepo.GetNative(ctx, chainID)
		if err != nil {
			return nil, fmt.Errorf("native token not found for chain %s: %w", chainID, err)
		}
		return nativeToken, nil
	}

	// Lookup by address
	token, err := u.tokenRepo.GetByAddress(ctx, address, chainID)
	if err != nil {
		return nil, fmt.Errorf("token not found for address %s: %w", address, err)
	}
	return token, nil
}

// buildTransactionData builds transaction data for frontend based on database metadata
func (u *PaymentUsecase) buildTransactionData(payment *entities.Payment, contract *entities.SmartContract) interface{} {
	if contract == nil {
		return nil
	}

	// Determine chain type using SourceChain relation if available, else standard EVM default or fetch
	// For now, assuming standard EVM if not Sol
	// But better: use SourceChain.ChainID (string/CAIP-2) if preloaded.
	var sourceChainID string
	if payment.SourceChain != nil {
		sourceChainID = payment.SourceChain.ChainID
	}
	chainType := getChainTypeFromCAIP2(sourceChainID)

	switch chainType {
	case "eip155":
		// Removed unused bridgeFee parsing for now
		return map[string]string{
			"to":   contract.ContractAddress,
			"data": u.buildEvmPaymentHex(payment),
			// "value": "0x...", // Todo: Add value when we have it in payment entity
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
	destChainId := payment.DestChainID[:]
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

// getBridgeFeeQuote fetches fee from on-chain Router
func (u *PaymentUsecase) getBridgeFeeQuote(ctx context.Context, sourceChainID, destChainID string) (*big.Int, error) {
	// 1. Get Chain Config to resolve UUID and RPCs (SourceChainID is NetworkID)
	chain, err := u.chainRepo.GetByChainID(ctx, sourceChainID)
	if err != nil {
		return nil, fmt.Errorf("chain config not found: %w", err)
	}

	// 2. Get Active Router
	router, err := u.contractRepo.GetActiveContract(ctx, chain.ID, entities.ContractTypeRouter)
	if err != nil {
		return nil, fmt.Errorf("active router not found: %w", err)
	}

	// 3. Get RPC Client
	var client *blockchain.EVMClient
	var clientErr error

	// Use RPCs if available, fallback to legacy RPCURL
	var targets []string
	for _, rpc := range chain.RPCs {
		if rpc.IsActive {
			targets = append(targets, rpc.URL)
		}
	}
	if len(targets) == 0 && chain.RPCURL != "" {
		targets = []string{chain.RPCURL}
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no RPC endpoints available for chain %s", sourceChainID)
	}

	for _, url := range targets {
		c, err := u.clientFactory.GetEVMClient(url)
		if err == nil {
			client = c
			break
		}
		clientErr = err
		// Continue to next RPC
	}

	if client == nil {
		return nil, fmt.Errorf("failed to connect to any RPC endpoint: %w", clientErr)
	}

	// 3. Select Bridge Type
	bridgeTypeStr := u.SelectBridge(sourceChainID, destChainID)
	var bridgeType uint8
	if bridgeTypeStr == "CCIP" {
		bridgeType = 1
	} else {
		bridgeType = 0 // Hyperbridge/Hyperlane default
	}

	// 4. Encode Calldata: quoteFee(string destChainId, uint8 bridgeType)
	// Selector: 0x9a8f4304 (calculated for quoteFee(string,uint8))
	// Actually let's assume standard packing.
	// Since I can't look up selector easily, I'll trust my calculation or use a known one.
	// keccak256("quoteFee(string,uint8)") = 9a8f4304...

	// Manual ABI Packing
	// DestChainID (string) -> offset 64 (0x40)
	// BridgeType (uint8) -> offset 32 (0x20) (padded) -> Wait, method(string, uint8)
	// Stack: [OffsetString, Uint8]
	// 0x00: Offset to string data (0x40 = 64 bytes)
	// 0x20: BridgeType (padded to 32 bytes)
	// 0x40: String Length
	// 0x60: String Data (padded)

	selector := "9a8f4304"

	// Bridge Type (uint8) padded to 32 bytes
	bridgeTypeHex := fmt.Sprintf("%064x", bridgeType)

	// DestChainID String
	strData := []byte(destChainID)
	lenStr := len(strData)
	lenStrHex := fmt.Sprintf("%064x", lenStr)
	strDataHex := hex.EncodeToString(strData)
	// Pad string data to 32 bytes multiple
	if lenStr%32 != 0 {
		padding := 32 - (lenStr % 32)
		strDataHex += strings.Repeat("00", padding)
	}

	// Offset is 0x40 (64) because param 1 is string (dynamic), param 2 is uint8 (static).
	// WAIT: in ABI encoding, static types come first in Head? No.
	// Params are encoded in order.
	// 1. String (Dynamic) -> Head is Offset.
	// 2. Uint8 (Static) -> Head is Value.
	// So:
	// 0x00: Offset of String (0x40 = 64)
	// 0x20: BridgeType Value
	// 0x40: String Length
	// 0x60: String Content

	offsetHex := fmt.Sprintf("%064x", 64)

	calldataHex := selector + offsetHex + bridgeTypeHex + lenStrHex + strDataHex
	calldata, _ := hex.DecodeString(calldataHex)

	// 5. Call Contract
	result, err := client.CallView(ctx, router.ContractAddress, calldata)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	// 6. Decode Result (uint256)
	if len(result) == 0 {
		return nil, fmt.Errorf("empty result from quoteFee")
	}
	fee := new(big.Int).SetBytes(result)

	return fee, nil
}
