package usecases

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/pkg/utils"
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
	bridgeConfigRepo repositories.BridgeConfigRepository
	feeConfigRepo    repositories.FeeConfigRepository
	uow              repositories.UnitOfWork
	clientFactory    *blockchain.ClientFactory
	chainResolver    *ChainResolver
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
	bridgeConfigRepo repositories.BridgeConfigRepository,
	feeConfigRepo repositories.FeeConfigRepository,
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
		bridgeConfigRepo: bridgeConfigRepo,
		feeConfigRepo:    feeConfigRepo,
		uow:              uow,
		clientFactory:    clientFactory,
		chainResolver:    NewChainResolver(chainRepo),
	}
}

// FeeConfig holds fee configuration
type FeeConfig struct {
	BaseFeeToken     float64 // Base fee in token amount
	PercentageFee    float64 // Percentage fee (e.g., 0.003 = 0.3%)
	BridgeFeeFlat    float64 // Flat bridge fee in token amount
	MerchantDiscount float64 // Merchant discount percentage
}

// DefaultFeeConfig returns default fee configuration
func DefaultFeeConfig() *FeeConfig {
	return &FeeConfig{
		BaseFeeToken:  DefaultFixedFeeUSD,
		PercentageFee: DefaultPercentageFee,
		BridgeFeeFlat: DefaultBridgeFeeFlat,
	}
}

// CalculateFees calculates fees for a payment
func (u *PaymentUsecase) CalculateFees(
	ctx context.Context,
	amount *big.Int,
	decimals int,
	sourceChainID, destChainID string,
	sourceChainUUID uuid.UUID,
	sourceTokenID uuid.UUID,
	sourceTokenAddress string,
	destTokenAddress string,
	merchantDiscount float64,
) *entities.FeeBreakdown {
	// Convert amount to float for calculation
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	dist := new(big.Float).SetInt(amount)
	amountFloat, _ := new(big.Float).Quo(dist, divisor).Float64()

	config := DefaultFeeConfig()
	minFeeToken := 0.0
	maxFeeToken := -1.0
	if u.feeConfigRepo != nil {
		if feeCfg, err := u.feeConfigRepo.GetByChainAndToken(ctx, sourceChainUUID, sourceTokenID); err == nil && feeCfg != nil {
			if v, parseErr := strconv.ParseFloat(feeCfg.FixedBaseFee, 64); parseErr == nil {
				config.BaseFeeToken = v
			}
			if v, parseErr := strconv.ParseFloat(feeCfg.PlatformFeePercent, 64); parseErr == nil {
				config.PercentageFee = v
			}
			if v, parseErr := strconv.ParseFloat(feeCfg.MinFee, 64); parseErr == nil {
				minFeeToken = v
			}
			if feeCfg.MaxFee != nil && *feeCfg.MaxFee != "" {
				if v, parseErr := strconv.ParseFloat(*feeCfg.MaxFee, 64); parseErr == nil {
					maxFeeToken = v
				}
			}
		}
	}

	// Platform fee: base + percentage
	platformFee := config.BaseFeeToken + amountFloat*config.PercentageFee

	// Apply merchant discount
	if merchantDiscount > 0 {
		platformFee = platformFee * (1 - merchantDiscount)
	}
	if platformFee < minFeeToken {
		platformFee = minFeeToken
	}
	if maxFeeToken >= 0 && platformFee > maxFeeToken {
		platformFee = maxFeeToken
	}

	// Bridge fee (only for cross-chain)
	isCrossChain := sourceChainID != destChainID // Defined here
	bridgeFeeToken := 0.0
	if isCrossChain {
		if quotedBridgeFeeWei, err := u.getBridgeFeeQuote(ctx, sourceChainID, destChainID, sourceTokenAddress, destTokenAddress, amount); err == nil && quotedBridgeFeeWei != nil {
			bridgeFeeFloat := new(big.Float).SetInt(quotedBridgeFeeWei)
			divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
			feeTokens, _ := new(big.Float).Quo(bridgeFeeFloat, divisor).Float64()
			bridgeFeeToken = feeTokens
		} else {
			bridgeFeeToken = config.BridgeFeeFlat
		}
	}

	// Total Fee in Token (Platform Fee + bridge flat fee)
	totalFeeToken := platformFee + bridgeFeeToken
	netAmount := amountFloat - totalFeeToken

	return &entities.FeeBreakdown{
		PlatformFee: formatAmount(platformFee, decimals),
		BridgeFee:   formatAmount(bridgeFeeToken, decimals),
		GasFee:      "0", // Gas is handled separately
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

	sourceChainUUID, sourceCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.SourceChainID)
	if err != nil {
		return nil, fmt.Errorf("invalid source chain: %w", err)
	}
	destChainUUID, destCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.DestChainID)
	if err != nil {
		return nil, fmt.Errorf("invalid dest chain: %w", err)
	}

	sourceChain, err := u.chainRepo.GetByID(ctx, sourceChainUUID)
	if err != nil {
		return nil, fmt.Errorf("error fetching source chain: %w", err)
	}
	destChain, err := u.chainRepo.GetByID(ctx, destChainUUID)
	if err != nil {
		return nil, fmt.Errorf("error fetching dest chain: %w", err)
	}

	// Parse amount
	amount := new(big.Int)
	if _, ok := amount.SetString(input.Amount, 10); !ok {
		return nil, domainerrors.ErrBadRequest
	}

	// Select bridge
	bridgeType := ""
	var bridgeID *uuid.UUID
	isCrossChain := sourceCAIP2 != destCAIP2
	if isCrossChain {
		bridgeType, bridgeID = u.decideBridge(ctx, sourceChainUUID, destChainUUID, sourceCAIP2, destCAIP2)
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

	// Calculate fees after token is resolved so chain/token-specific fee_configs can be applied.
	feeBreakdown := u.CalculateFees(
		ctx,
		amount,
		input.Decimals,
		sourceCAIP2,
		destCAIP2,
		sourceChainUUID,
		sourceTokenID,
		input.SourceTokenAddress,
		input.DestTokenAddress,
		0,
	)

	// Create payment entity
	payment := &entities.Payment{
		ID:                 utils.GenerateUUIDv7(), // Generate ID
		SenderID:           &userID,
		MerchantID:         nil, // userID is User, not Merchant in this context? Or need to check if User is Merchant?
		BridgeID:           bridgeID,
		SourceChainID:      sourceChainUUID,
		DestChainID:        destChainUUID,
		SourceTokenID:      &sourceTokenID,
		DestTokenID:        &destTokenID,
		SourceTokenAddress: input.SourceTokenAddress,
		DestTokenAddress:   input.DestTokenAddress,
		SourceAmount:       input.Amount,
		FeeAmount:          feeBreakdown.TotalFee,
		TotalCharged:       input.Amount,
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
	payment.SourceChain = sourceChain
	payment.DestChain = destChain
	if totalCharged, calcErr := addDecimalStrings(input.Amount, feeBreakdown.TotalFee); calcErr == nil {
		payment.TotalCharged = totalCharged
	}

	// Set nullable DestAmount
	if feeBreakdown.NetAmount != "" {
		payment.DestAmount = null.StringFrom(feeBreakdown.NetAmount)
	}

	// Save payment
	// Wrap DB operations in Transaction
	if err = u.uow.Do(ctx, func(txCtx context.Context) error {
		// Save payment
		if err := u.paymentRepo.Create(txCtx, payment); err != nil {
			return err
		}

		// Create initial event
		event := &entities.PaymentEvent{
			ID:        utils.GenerateUUIDv7(),
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
		SourceChainID:  sourceCAIP2,
		DestChainID:    destCAIP2,
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

func (u *PaymentUsecase) decideBridge(
	ctx context.Context,
	sourceChainUUID, destChainUUID uuid.UUID,
	sourceCAIP2, destCAIP2 string,
) (string, *uuid.UUID) {
	if u.bridgeConfigRepo != nil {
		if cfg, err := u.bridgeConfigRepo.GetActive(ctx, sourceChainUUID, destChainUUID); err == nil && cfg != nil {
			if cfg.Bridge != nil && cfg.Bridge.Name != "" {
				id := cfg.Bridge.ID
				return cfg.Bridge.Name, &id
			}
		}
	}

	// Fallback to legacy deterministic selection.
	return u.SelectBridge(sourceCAIP2, destCAIP2), nil
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
	sourceChainID := ""
	if payment.SourceChain != nil {
		sourceChainID = payment.SourceChain.GetCAIP2ID()
	} else if chain, err := u.chainRepo.GetByID(context.Background(), payment.SourceChainID); err == nil && chain != nil {
		sourceChainID = chain.GetCAIP2ID()
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
			"data":      u.buildSvmPaymentBase58(payment),
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

func (u *PaymentUsecase) buildSvmPaymentBase58(payment *entities.Payment) string {
	discriminator := anchorDiscriminator("create_payment")
	paymentID := uuidToBytes32(payment.ID)
	destChainID := payment.DestChainID.String()
	if payment.DestChain != nil {
		destChainID = payment.DestChain.GetCAIP2ID()
	}
	destToken := addressToBytes32(payment.DestTokenAddress)
	receiver := addressToBytes32(payment.ReceiverAddress)
	amount := decimalStringToUint64(payment.SourceAmount)

	data := make([]byte, 0, 8+32+4+len(destChainID)+32+8+32)
	data = append(data, discriminator[:]...)
	data = append(data, paymentID[:]...)
	data = append(data, encodeAnchorString(destChainID)...)
	data = append(data, destToken[:]...)
	data = append(data, u64ToLE(amount)...)
	data = append(data, receiver[:]...)
	return base58Encode(data)
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
func (u *PaymentUsecase) getBridgeFeeQuote(
	ctx context.Context,
	sourceChainID, destChainID string,
	sourceTokenAddress, destTokenAddress string,
	amount *big.Int,
) (*big.Int, error) {
	sourceChainUUID, sourceCAIP2, err := u.chainResolver.ResolveFromAny(ctx, sourceChainID)
	if err != nil {
		return nil, fmt.Errorf("source chain config not found: %w", err)
	}
	destChainUUID, destCAIP2, err := u.chainResolver.ResolveFromAny(ctx, destChainID)
	if err != nil {
		return nil, fmt.Errorf("dest chain config not found: %w", err)
	}

	chain, err := u.chainRepo.GetByID(ctx, sourceChainUUID)
	if err != nil {
		return nil, fmt.Errorf("source chain not found: %w", err)
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

	// 4. Select bridge type from DB config first, fallback deterministic rule
	bridgeTypeStr, _ := u.decideBridge(ctx, sourceChainUUID, destChainUUID, sourceCAIP2, destCAIP2)
	bridgeType := uint8(0)
	if strings.EqualFold(bridgeTypeStr, "CCIP") {
		bridgeType = 1
	}

	// 5. ABI encode quotePaymentFee(string,uint8,(bytes32,address,address,address,uint256,string,uint256))
	messageTupleType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "paymentId", Type: "bytes32"},
		{Name: "receiver", Type: "address"},
		{Name: "sourceToken", Type: "address"},
		{Name: "destToken", Type: "address"},
		{Name: "amount", Type: "uint256"},
		{Name: "destChainId", Type: "string"},
		{Name: "minAmountOut", Type: "uint256"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build ABI tuple type: %w", err)
	}
	stringType, _ := abi.NewType("string", "", nil)
	uint8Type, _ := abi.NewType("uint8", "", nil)
	args := abi.Arguments{
		{Type: stringType},
		{Type: uint8Type},
		{Type: messageTupleType},
	}

	type bridgeMessage struct {
		PaymentId    [32]byte
		Receiver     common.Address
		SourceToken  common.Address
		DestToken    common.Address
		Amount       *big.Int
		DestChainId  string
		MinAmountOut *big.Int
	}
	msgStruct := bridgeMessage{
		PaymentId:    [32]byte{},
		Receiver:     common.Address{},
		SourceToken:  common.HexToAddress(normalizeEvmAddress(sourceTokenAddress)),
		DestToken:    common.HexToAddress(normalizeEvmAddress(destTokenAddress)),
		Amount:       amount,
		DestChainId:  destCAIP2,
		MinAmountOut: big.NewInt(0),
	}

	packedArgs, err := args.Pack(destCAIP2, bridgeType, msgStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to pack quotePaymentFee args: %w", err)
	}
	methodSig := []byte("quotePaymentFee(string,uint8,(bytes32,address,address,address,uint256,string,uint256))")
	methodID := crypto.Keccak256(methodSig)[:4]
	calldata := append(methodID, packedArgs...)

	// 6. Call Contract
	result, err := client.CallView(ctx, router.ContractAddress, calldata)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	// 7. Decode Result (uint256)
	if len(result) == 0 {
		return nil, fmt.Errorf("empty result from quotePaymentFee")
	}
	fee := new(big.Int).SetBytes(result)

	return fee, nil
}
