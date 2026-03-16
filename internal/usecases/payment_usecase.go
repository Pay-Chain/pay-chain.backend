package usecases

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v8"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/infrastructure/blockchain"
	"payment-kita.backend/internal/infrastructure/metrics"
	"payment-kita.backend/pkg/utils"
)

var (
	newABIType  = abi.NewType
	packABIArgs = func(args abi.Arguments, values ...interface{}) ([]byte, error) {
		return args.Pack(values...)
	}
	bridgeFeeSafetyBps = big.NewInt(12000) // +20% margin on top of quoted bridge fee
	bpsDenominator     = big.NewInt(10000)
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
	routePolicyRepo  repositories.RoutePolicyRepository
	uow              repositories.UnitOfWork
	clientFactory    *blockchain.ClientFactory
	chainResolver    *ChainResolver
	*ABIResolverMixin
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
	routePolicyRepo repositories.RoutePolicyRepository,
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
		routePolicyRepo:  routePolicyRepo,
		uow:              uow,
		clientFactory:    clientFactory,
		chainResolver:    NewChainResolver(chainRepo),
		ABIResolverMixin: NewABIResolverMixin(contractRepo),
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
	destChainUUID uuid.UUID,
	sourceTokenID uuid.UUID,
	sourceTokenAddress string,
	destTokenAddress string,
	destTokenDecimals int,
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

	// Platform fee: min(amount * percentage, baseFee)
	platformValue := amountFloat * config.PercentageFee
	// config.BaseFeeToken is now treated as the Fixed Cap
	if platformValue > config.BaseFeeToken {
		platformValue = config.BaseFeeToken
	}

	platformFee := platformValue

	// Apply merchant discount
	if merchantDiscount > 0 {
		platformFee = platformFee * (1 - merchantDiscount)
	}
	if platformFee < minFeeToken {
		platformFee = minFeeToken
	}
	// maxFeeToken is still respected if set, but mostlyredundant with our new cap logic
	if maxFeeToken >= 0 && platformFee > maxFeeToken {
		platformFee = maxFeeToken
	}

	// Bridge fee (only for cross-chain)
	isCrossChain := sourceChainID != destChainID // Defined here
	bridgeFeeToken := 0.0
	if isCrossChain {
		// Bridge quote is native-gas-denominated and is paid via tx value on EVM path.
		// Do not add it to token-denominated fee for ERC20 source payments.
		isSourceNative := !u.shouldRequireEvmApproval(sourceTokenAddress)
		if isSourceNative {
			if quotedBridgeFeeWei, err := u.getBridgeFeeQuote(ctx, sourceChainID, destChainID, sourceTokenAddress, destTokenAddress, amount, big.NewInt(0)); err == nil && quotedBridgeFeeWei != nil {
				bridgeFeeFloat := new(big.Float).SetInt(quotedBridgeFeeWei)
				divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
				feeTokens, _ := new(big.Float).Quo(bridgeFeeFloat, divisor).Float64()
				bridgeFeeToken = feeTokens
			} else {
				bridgeFeeToken = config.BridgeFeeFlat
			}
		}
	}

	// Total Fee in Token (Platform Fee + bridge flat fee)
	totalFeeToken := platformFee + bridgeFeeToken

	netAmount := amountFloat - totalFeeToken
	netAmountStr := formatAmount(netAmount, decimals)

	// If tokens are different, we need a price-aware net amount in destination token units.
	if sourceTokenAddress != destTokenAddress && sourceTokenAddress != "" && destTokenAddress != "" {
		// Calculate net amount in source token first (after platform fees)
		netAmountSourceToken := new(big.Int).Sub(amount, new(big.Int).SetInt64(int64(platformFee*math.Pow10(decimals))))
		if quote, err := u.getSwapQuote(ctx, sourceChainUUID, sourceTokenAddress, destTokenAddress, netAmountSourceToken); err == nil && quote != nil {
			netAmountStr = quote.String() // Return in smallest unit of dest token
		}
	}

	return &entities.FeeBreakdown{
		PlatformFee: formatAmount(platformFee, decimals),
		BridgeFee:   formatAmount(bridgeFeeToken, decimals),
		GasFee:      "0", // Gas is handled separately
		TotalFee:    formatAmount(totalFeeToken, decimals),
		NetAmount:   netAmountStr,
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

	decimals := srcToken.Decimals
	if input.Decimals > 0 && input.Decimals != decimals {
		return nil, fmt.Errorf("source token decimals mismatch: expected %d got %d", decimals, input.Decimals)
	}

	amountSmallestUnit, convErr := convertToSmallestUnit(input.Amount, decimals)
	if convErr != nil {
		return nil, domainerrors.ErrBadRequest
	}

	amount := new(big.Int)
	amount.SetString(amountSmallestUnit, 10)

	// Calculate fees after token is resolved so chain/token-specific fee_configs can be applied.
	feeBreakdown := u.CalculateFees(
		ctx,
		amount,
		decimals,
		sourceCAIP2,
		destCAIP2,
		sourceChainUUID,
		destChainUUID,
		sourceTokenID,
		input.SourceTokenAddress,
		input.DestTokenAddress,
		destToken.Decimals,
		0,
	)

	// Calculate MinDestAmount if SlippageBps is provided
	var minDestAmountStr null.String
	if input.SlippageBps > 0 {
		netAmountBig := new(big.Int)
		if _, ok := netAmountBig.SetString(feeBreakdown.NetAmount, 10); ok {
			// min = net * (10000 - slippage) / 10000
			factor := big.NewInt(int64(10000 - input.SlippageBps))
			minDest := new(big.Int).Mul(netAmountBig, factor)
			minDest.Div(minDest, big.NewInt(10000))
			minDestAmountStr = null.StringFrom(minDest.String())
		}
	} else if input.MinAmountOut != "" {
		minDestAmountStr = null.StringFrom(input.MinAmountOut)
	}

	// Merchant Attribution
	var merchantID *uuid.UUID
	if mID, ok := ctx.Value("MerchantID").(uuid.UUID); ok {
		merchantID = &mID
	} else if input.ReceiverMerchantID != "" {
		if mID, err := uuid.Parse(input.ReceiverMerchantID); err == nil {
			merchantID = &mID
		}
	}

	// Create payment entity
	payment := &entities.Payment{
		ID:                 utils.GenerateUUIDv7(), // Generate ID
		SenderID:           &userID,
		MerchantID:         merchantID,
		BridgeID:           bridgeID,
		SourceChainID:      sourceChainUUID,
		DestChainID:        destChainUUID,
		SourceTokenID:      &sourceTokenID,
		DestTokenID:        &destTokenID,
		SourceTokenAddress: input.SourceTokenAddress,
		DestTokenAddress:   input.DestTokenAddress,
		SourceAmount:       amountSmallestUnit,
		DestAmount:         null.StringFrom(feeBreakdown.NetAmount), // Set initially
		FeeAmount:          feeBreakdown.TotalFee,
		MinDestAmount:      minDestAmountStr,
		TotalCharged:       amountSmallestUnit,
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
	if totalCharged, calcErr := addDecimalStrings(amountSmallestUnit, feeBreakdown.TotalFee); calcErr == nil {
		payment.TotalCharged = totalCharged
	}

	// Set nullable DestAmount
	// Already set above, ensuring consistency
	if feeBreakdown.NetAmount != "" {
		payment.DestAmount = null.StringFrom(feeBreakdown.NetAmount)
	}

	// Save payment in transaction.
	if err = u.uow.Do(ctx, func(txCtx context.Context) error {
		if err := u.paymentRepo.Create(txCtx, payment); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Create initial event as best-effort after payment commit.
	// Never fail payment creation when event table has FK/schema timing issues.
	event := &entities.PaymentEvent{
		ID:        utils.GenerateUUIDv7(),
		PaymentID: payment.ID,
		EventType: entities.PaymentEventTypeCreated,
		ChainID:   &sourceChain.ID,
		CreatedAt: time.Now(),
	}
	if err := u.paymentEventRepo.Create(ctx, event); err != nil {
		fmt.Printf("Warning: failed to create payment event for payment %s: %v\n", payment.ID, err)
	}

	// Build transaction data using metadata from DB
	signatureData, sigErr := u.buildTransactionDataWithInput(payment, contract, input)
	if sigErr != nil {
		return nil, sigErr
	}

	// Phase 3 (Track-B): expose gateway quotePaymentCost breakdown when available.
	var onchainCost *entities.OnchainCost
	if contract != nil && getChainTypeFromCAIP2(sourceCAIP2) == "eip155" {
		if quoted, qErr := u.quoteGatewayPaymentCost(ctx, payment, contract.ContractAddress, input); qErr == nil {
			onchainCost = quoted
		}
	}
	if snapshotMetadata := buildPaymentQuoteSnapshotMetadata(signatureData, onchainCost); snapshotMetadata != nil {
		snapshotEvent := &entities.PaymentEvent{
			ID:        utils.GenerateUUIDv7(),
			PaymentID: payment.ID,
			EventType: entities.PaymentEventType("QUOTE_SNAPSHOT_CAPTURED"),
			ChainID:   &sourceChain.ID,
			Metadata:  snapshotMetadata,
			CreatedAt: time.Now(),
		}
		if err := u.paymentEventRepo.Create(ctx, snapshotEvent); err != nil {
			fmt.Printf("Warning: failed to create quote snapshot event for payment %s: %v\n", payment.ID, err)
		}
	}

	// Record metrics
	merchantIDStr := "anonymous"
	if merchantID != nil {
		merchantIDStr = merchantID.String()
	}
	metrics.RecordSessionCreated(merchantIDStr, nil)

	return &entities.CreatePaymentResponse{
		PaymentID:      payment.ID,
		Status:         payment.Status,
		SourceChainID:  sourceCAIP2,
		DestChainID:    destCAIP2,
		SourceAmount:   payment.SourceAmount,
		SourceDecimals: decimals,
		DestAmount:     payment.DestAmount.String,
		FeeAmount:      payment.FeeAmount,
		BridgeType:     bridgeType,
		FeeBreakdown:   *feeBreakdown,
		OnchainCost:    onchainCost,
		ExpiresAt:      time.Now().Add(PaymentExpiryDuration),
		SignatureData:  signatureData,
	}, nil
}

func (u *PaymentUsecase) decideBridge(
	ctx context.Context,
	sourceChainUUID, destChainUUID uuid.UUID,
	sourceCAIP2, destCAIP2 string,
) (string, *uuid.UUID) {
	// Priority 1: explicit route policy (default bridge type)
	if u.routePolicyRepo != nil {
		if policy, err := u.routePolicyRepo.GetByRoute(ctx, sourceChainUUID, destChainUUID); err == nil && policy != nil {
			return bridgeTypeToName(policy.DefaultBridgeType), nil
		}
		// Phase 4.3: Auto-bootstrap disabled to prevent unwanted defaults.
		// else if errors.Is(err, domainerrors.ErrNotFound) {
		// 	u.bootstrapDefaultRoutePolicy(ctx, sourceChainUUID, destChainUUID, sourceCAIP2, destCAIP2)
		// }
	}

	// Priority 2: existing bridge config mapping
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

func isForeignKeyViolation(err error, constraint string) bool {
	if err == nil {
		return false
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		if string(pgErr.Code) == "23503" {
			if strings.TrimSpace(constraint) == "" {
				return true
			}
			return strings.EqualFold(strings.TrimSpace(pgErr.Constraint), strings.TrimSpace(constraint))
		}
	}
	return false
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
func (u *PaymentUsecase) buildTransactionData(payment *entities.Payment, contract *entities.SmartContract) (interface{}, error) {
	return u.buildTransactionDataWithInput(payment, contract, nil)
}

func (u *PaymentUsecase) buildTransactionDataWithInput(
	payment *entities.Payment,
	contract *entities.SmartContract,
	input *entities.CreatePaymentInput,
) (interface{}, error) {
	if contract == nil {
		return nil, nil
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
		destChainID := payment.DestChainID.String()
		if payment.DestChain != nil {
			destChainID = payment.DestChain.GetCAIP2ID()
		} else if chain, err := u.chainRepo.GetByID(context.Background(), payment.DestChainID); err == nil && chain != nil {
			destChainID = chain.GetCAIP2ID()
		}

		minDestAmount := big.NewInt(0)
		if payment.MinDestAmount.Valid {
			minDestAmount.SetString(payment.MinDestAmount.String, 10)
		}

		createPaymentData := u.buildEvmPaymentHexWithInput(payment, destChainID, minDestAmount, input)
		if createPaymentData == "" {
			return nil, domainerrors.BadRequest("failed to build gateway calldata")
		}
		txValueHex := "0x0"
		var previewApprovalAmount string
		sourceCAIP2 := ""
		if payment.SourceChain != nil {
			sourceCAIP2 = payment.SourceChain.GetCAIP2ID()
		} else if srcChain, err := u.chainRepo.GetByID(context.Background(), payment.SourceChainID); err == nil && srcChain != nil {
			sourceCAIP2 = srcChain.GetCAIP2ID()
		}
		// Always try Track-B preview for approval amount (same-chain and cross-chain).
		// For cross-chain we also consume required native fee from the same preview.
		preview, previewErr := u.previewGatewayApprovalV2(context.Background(), payment, contract.ContractAddress, input)
		if previewErr == nil && preview != nil && preview.ApprovalAmount != nil && preview.ApprovalAmount.Sign() > 0 {
			previewApprovalAmount = preview.ApprovalAmount.String()
		}
		if sourceCAIP2 != "" && sourceCAIP2 != destChainID {
			amount := new(big.Int)
			if _, ok := amount.SetString(payment.SourceAmount, 10); !ok {
				return nil, domainerrors.BadRequest("invalid source amount for bridge fee quote")
			}

			// Preferred V2 preview path for native fee.
			if previewErr == nil && preview != nil && preview.RequiredNativeFee != nil && preview.RequiredNativeFee.Sign() >= 0 {
				txValueHex = "0x" + preview.RequiredNativeFee.Text(16)
			} else {
				feeWei, err := u.getBridgeFeeQuote(context.Background(), sourceCAIP2, destChainID, payment.SourceTokenAddress, payment.DestTokenAddress, amount, minDestAmount)
				if err != nil {
					// Fallback to gateway quotePaymentCost when router quote path is temporarily unavailable.
					if fallback, fbErr := u.quoteGatewayPaymentCost(context.Background(), payment, contract.ContractAddress, input); fbErr == nil && fallback != nil && fallback.BridgeQuoteOk {
						fallbackFeeWei := new(big.Int)
						if _, ok := fallbackFeeWei.SetString(strings.TrimSpace(fallback.BridgeFeeNative), 10); ok && fallbackFeeWei.Sign() >= 0 {
							feeWei = fallbackFeeWei
						} else {
							return nil, domainerrors.BadRequest(fmt.Sprintf(
								"failed to resolve bridge fee quote for %s -> %s: %v (fallback bridge fee invalid: %q)",
								sourceCAIP2,
								destChainID,
								err,
								fallback.BridgeFeeNative,
							))
						}
					} else {
						return nil, domainerrors.BadRequest(fmt.Sprintf(
							"failed to resolve bridge fee quote for %s -> %s: primaryErr=%v fallbackErr=%v",
							sourceCAIP2,
							destChainID,
							err,
							fbErr,
						))
					}
				}
				feeWithMargin := new(big.Int).Mul(feeWei, bridgeFeeSafetyBps)
				feeWithMargin.Div(feeWithMargin, bpsDenominator)
				if feeWithMargin.Cmp(feeWei) < 0 {
					feeWithMargin = new(big.Int).Set(feeWei)
				}
				txValueHex = "0x" + feeWithMargin.Text(16)
			}
		}
		result := map[string]interface{}{
			"to":           contract.ContractAddress,
			"data":         createPaymentData,
			"value":        txValueHex,
			"transactions": []map[string]string{},
		}
		createPaymentTx := map[string]string{
			"kind":  "createPayment",
			"to":    contract.ContractAddress,
			"data":  createPaymentData,
			"value": txValueHex,
		}
		txs := make([]map[string]string, 0, 3)
		if privacyDeployTx, privacyErr := u.buildPrivacyEscrowDeployTx(
			context.Background(),
			payment,
			input,
			sourceCAIP2,
			destChainID,
		); privacyErr != nil {
			return nil, privacyErr
		} else if privacyDeployTx != nil {
			txs = append(txs, privacyDeployTx)
		}

		if u.shouldRequireEvmApproval(payment.SourceTokenAddress) {
			vaultAddress := u.resolveVaultAddressForApproval(payment.SourceChainID, contract.ContractAddress)
			if vaultAddress == "" {
				return nil, fmt.Errorf("vault contract address is not configured for source chain")
			}
			approvalAmount := strings.TrimSpace(previewApprovalAmount)
			if approvalAmount == "" || approvalAmount == "0" {
				approvalAmountResolved, approvalErr := u.calculateOnchainApprovalAmount(payment, contract.ContractAddress)
				if approvalErr != nil {
					// Keep payment creation resilient when optional quote path is unavailable.
					// Fallback to total charged amount (or source amount) to avoid API hard-fail.
					approvalAmount = strings.TrimSpace(payment.TotalCharged)
					if approvalAmount == "" || approvalAmount == "0" {
						approvalAmount = strings.TrimSpace(payment.SourceAmount)
					}
					if approvalAmount == "" || approvalAmount == "0" {
						return nil, approvalErr
					}
					fmt.Printf("Warning: using fallback approval amount for payment %s: %v\n", payment.ID, approvalErr)
				} else {
					approvalAmount = approvalAmountResolved
				}
			}
			approveData := u.buildErc20ApproveHex(vaultAddress, approvalAmount)
			approvalTx := map[string]string{
				"kind":    "approve",
				"to":      payment.SourceTokenAddress,
				"data":    approveData,
				"spender": vaultAddress,
				"amount":  approvalAmount,
			}
			result["approval"] = approvalTx
			txs = append(txs, approvalTx)
		}
		txs = append(txs, createPaymentTx)
		result["transactions"] = txs

		return result, nil
	case "solana":
		return map[string]string{
			"programId": contract.ContractAddress,
			"data":      u.buildSvmPaymentBase58(payment),
		}, nil
	}

	return nil, nil
}

func buildPaymentQuoteSnapshotMetadata(signatureData interface{}, onchainCost *entities.OnchainCost) map[string]interface{} {
	preview := extractPreviewApprovalSnapshot(signatureData)
	quote := extractOnchainCostSnapshot(onchainCost)
	if preview == nil && quote == nil {
		return nil
	}

	metadata := map[string]interface{}{
		"schema": "payment_quote_snapshot.v1",
	}
	if preview != nil {
		metadata["previewApproval"] = preview
	}
	if quote != nil {
		metadata["quotePaymentCost"] = quote
	}
	return metadata
}

func extractPreviewApprovalSnapshot(signatureData interface{}) map[string]interface{} {
	root, ok := signatureData.(map[string]interface{})
	if !ok {
		return nil
	}

	snapshot := make(map[string]interface{})
	if value := extractStringField(root, "value"); value != "" {
		snapshot["requiredNativeFee"] = value
	}

	approvalRaw, ok := root["approval"]
	if !ok {
		if len(snapshot) == 0 {
			return nil
		}
		return snapshot
	}

	approval, ok := approvalRaw.(map[string]string)
	if ok {
		if token := strings.TrimSpace(approval["to"]); token != "" {
			snapshot["approvalToken"] = token
		}
		if amount := strings.TrimSpace(approval["amount"]); amount != "" {
			snapshot["approvalAmount"] = amount
		}
		if spender := strings.TrimSpace(approval["spender"]); spender != "" {
			snapshot["approvalSpender"] = spender
		}
		if len(snapshot) == 0 {
			return nil
		}
		return snapshot
	}

	approvalAny, ok := approvalRaw.(map[string]interface{})
	if !ok {
		if len(snapshot) == 0 {
			return nil
		}
		return snapshot
	}
	if token := extractStringField(approvalAny, "to"); token != "" {
		snapshot["approvalToken"] = token
	}
	if amount := extractStringField(approvalAny, "amount"); amount != "" {
		snapshot["approvalAmount"] = amount
	}
	if spender := extractStringField(approvalAny, "spender"); spender != "" {
		snapshot["approvalSpender"] = spender
	}
	if len(snapshot) == 0 {
		return nil
	}
	return snapshot
}

func extractOnchainCostSnapshot(onchainCost *entities.OnchainCost) map[string]interface{} {
	if onchainCost == nil {
		return nil
	}
	return map[string]interface{}{
		"platformFeeToken":         onchainCost.PlatformFeeToken,
		"bridgeFeeNative":          onchainCost.BridgeFeeNative,
		"totalSourceTokenRequired": onchainCost.TotalSourceTokenRequired,
		"bridgeType":               onchainCost.BridgeType,
		"isSameChain":              onchainCost.IsSameChain,
		"bridgeQuoteOk":            onchainCost.BridgeQuoteOk,
		"bridgeQuoteReason":        onchainCost.BridgeQuoteReason,
	}
}

func extractStringField(values map[string]interface{}, key string) string {
	raw, ok := values[key]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func (u *PaymentUsecase) shouldRequireEvmApproval(tokenAddress string) bool {
	addr := strings.TrimSpace(strings.ToLower(tokenAddress))
	return addr != "" && addr != "native" && addr != "0x0000000000000000000000000000000000000000"
}

func (u *PaymentUsecase) resolveVaultAddressForApproval(sourceChainID uuid.UUID, gatewayAddress string) string {
	if vaultContract, err := u.contractRepo.GetActiveContract(context.Background(), sourceChainID, entities.ContractTypeVault); err == nil && vaultContract != nil {
		return vaultContract.ContractAddress
	}
	if gatewayAddress == "" {
		return ""
	}
	chain, err := u.chainRepo.GetByID(context.Background(), sourceChainID)
	if err != nil || chain == nil {
		return ""
	}
	rpcURL := strings.TrimSpace(chain.RPCURL)
	if rpcURL == "" {
		var fallback string
		for _, rpc := range chain.RPCs {
			url := strings.TrimSpace(rpc.URL)
			if url == "" {
				continue
			}
			if rpc.IsActive {
				rpcURL = url
				break
			}
			if fallback == "" {
				fallback = url
			}
		}
		if rpcURL == "" {
			rpcURL = fallback
		}
	}
	if rpcURL == "" {
		return ""
	}
	client, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return ""
	}
	// selector for vault() is 0xfbfa77cf
	out, err := client.CallView(context.Background(), gatewayAddress, common.FromHex("0xfbfa77cf"))
	if err != nil || len(out) < 32 {
		return ""
	}
	vaultAddress := common.BytesToAddress(out[12:32]).Hex()
	return vaultAddress
}

func (u *PaymentUsecase) buildErc20ApproveHex(spender, amount string) string {
	addressType, err := newABIType("address", "", nil)
	if err != nil {
		return ""
	}
	uintType, err := newABIType("uint256", "", nil)
	if err != nil {
		return ""
	}
	amountBig := new(big.Int)
	if _, ok := amountBig.SetString(amount, 10); !ok {
		return ""
	}
	args := abi.Arguments{
		{Type: addressType},
		{Type: uintType},
	}
	packed, err := packABIArgs(args, common.HexToAddress(normalizeEvmAddress(spender)), amountBig)
	if err != nil {
		return ""
	}
	// approve(address,uint256)
	return "0x095ea7b3" + hex.EncodeToString(packed)
}

func (u *PaymentUsecase) buildPrivacyEscrowDeployTx(
	ctx context.Context,
	payment *entities.Payment,
	input *entities.CreatePaymentInput,
	sourceCAIP2 string,
	destCAIP2 string,
) (map[string]string, error) {
	if payment == nil || input == nil {
		return nil, nil
	}
	if sourceCAIP2 == "" || destCAIP2 == "" || sourceCAIP2 != destCAIP2 {
		return nil, nil
	}
	if normalizePaymentMode(input.Mode) != PaymentModePrivacy {
		return nil, nil
	}
	if input.PrivacyIntentID == nil || input.PrivacyStealthReceiver == nil {
		return nil, nil
	}
	receiverRaw := strings.TrimSpace(payment.ReceiverAddress)
	intentRaw := strings.TrimSpace(*input.PrivacyIntentID)
	stealthRaw := strings.TrimSpace(*input.PrivacyStealthReceiver)
	if receiverRaw == "" || intentRaw == "" || stealthRaw == "" {
		return nil, fmt.Errorf("privacy routing fields are incomplete for same-chain escrow deployment")
	}
	if !common.IsHexAddress(receiverRaw) {
		return nil, fmt.Errorf("receiverAddress must be valid EVM address for same-chain privacy escrow")
	}
	if !common.IsHexAddress(stealthRaw) {
		return nil, fmt.Errorf("privacyStealthReceiver must be valid EVM address for same-chain privacy escrow")
	}

	if u.contractRepo == nil {
		return nil, fmt.Errorf("smart contract repository is required for same-chain privacy escrow deployment")
	}
	factoryContract, err := u.contractRepo.GetActiveContract(ctx, payment.SourceChainID, entities.ContractTypeStealthEscrowFactory)
	if err != nil || factoryContract == nil || !common.IsHexAddress(factoryContract.ContractAddress) {
		return nil, fmt.Errorf("active stealth escrow factory is not configured on source chain")
	}
	privacyModuleContract, err := u.contractRepo.GetActiveContract(ctx, payment.SourceChainID, entities.ContractTypePrivacyModule)
	if err != nil || privacyModuleContract == nil || !common.IsHexAddress(privacyModuleContract.ContractAddress) {
		return nil, fmt.Errorf("active privacy module is not configured on source chain")
	}

	deployData := u.buildDeployEscrowHex(
		parsePrivacyIntentID(intentRaw),
		common.HexToAddress(normalizeEvmAddress(receiverRaw)),
		common.HexToAddress(normalizeEvmAddress(privacyModuleContract.ContractAddress)),
	)
	if deployData == "" {
		return nil, fmt.Errorf("failed to build deployEscrow calldata")
	}

	return map[string]string{
		"kind":  "deployEscrow",
		"to":    factoryContract.ContractAddress,
		"data":  deployData,
		"value": "0x0",
	}, nil
}

func (u *PaymentUsecase) buildDeployEscrowHex(salt [32]byte, owner common.Address, forwarder common.Address) string {
	bytes32Type, err := newABIType("bytes32", "", nil)
	if err != nil {
		return ""
	}
	addressType, err := newABIType("address", "", nil)
	if err != nil {
		return ""
	}
	args := abi.Arguments{
		{Type: bytes32Type},
		{Type: addressType},
		{Type: addressType},
	}
	packed, err := packABIArgs(args, salt, owner, forwarder)
	if err != nil {
		return ""
	}
	return "0x" + hex.EncodeToString(append(common.FromHex(DeployEscrowSelector), packed...))
}

func (u *PaymentUsecase) calculateOnchainApprovalAmount(payment *entities.Payment, gatewayAddress string) (string, error) {
	if payment == nil || gatewayAddress == "" {
		return "", fmt.Errorf("invalid payment or gateway address")
	}
	amount := new(big.Int)
	if _, ok := amount.SetString(payment.SourceAmount, 10); !ok {
		return "", fmt.Errorf("invalid source amount")
	}
	totalCharged := new(big.Int)
	if _, ok := totalCharged.SetString(payment.TotalCharged, 10); !ok {
		totalCharged = new(big.Int).Set(amount)
	}

	// Track-B preferred path: quotePaymentCost(PaymentRequestV2)
	// Use contract-returned totalSourceTokenRequired if available.
	if quote, err := u.quoteGatewayPaymentCost(context.Background(), payment, gatewayAddress, nil); err == nil && quote != nil {
		quotedTotal := new(big.Int)
		if _, ok := quotedTotal.SetString(quote.TotalSourceTokenRequired, 10); ok {
			if quotedTotal.Cmp(totalCharged) < 0 {
				return totalCharged.String(), nil
			}
			return quotedTotal.String(), nil
		}
	}

	chain, err := u.chainRepo.GetByID(context.Background(), payment.SourceChainID)
	if err != nil || chain == nil {
		return "", fmt.Errorf("failed to resolve source chain")
	}
	rpcURL := strings.TrimSpace(chain.RPCURL)
	if rpcURL == "" {
		for _, rpc := range chain.RPCs {
			if rpc.IsActive && strings.TrimSpace(rpc.URL) != "" {
				rpcURL = rpc.URL
				break
			}
		}
	}
	if rpcURL == "" {
		return "", fmt.Errorf("no active source chain rpc url")
	}
	client, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return "", fmt.Errorf("failed to create evm client for approval quote: %w", err)
	}

	feeABI := FallbackPaymentKitaGatewayABI
	if u.ABIResolverMixin != nil {
		resolvedABI, abiErr := u.ResolveABIWithFallback(context.Background(), payment.SourceChainID, entities.ContractTypeGateway)
		if abiErr != nil {
			return "", fmt.Errorf("failed to resolve gateway ABI: %w", abiErr)
		}
		// Some DB ABI rows are stale and can miss Track-B methods.
		// Prefer resolved ABI only when it contains the method we need.
		if _, ok := resolvedABI.Methods["quoteTotalAmount"]; ok {
			feeABI = resolvedABI
		}
	}

	// Preferred path: ask contract directly for exact total amount
	quoteCall, quoteErr := feeABI.Pack("quoteTotalAmount", amount)
	if quoteErr == nil {
		quoteRaw, callErr := client.CallView(context.Background(), gatewayAddress, quoteCall)
		if callErr == nil {
			quoteVals, unpackErr := feeABI.Unpack("quoteTotalAmount", quoteRaw)
			if unpackErr == nil && len(quoteVals) >= 1 {
				if quotedTotal, ok := quoteVals[0].(*big.Int); ok && quotedTotal != nil {
					if quotedTotal.Cmp(totalCharged) < 0 {
						return totalCharged.String(), nil
					}
					return quotedTotal.String(), nil
				}
			}
		}
	}

	fixedFeeCall, _ := feeABI.Pack("FIXED_BASE_FEE")
	fixedFeeRaw, err := client.CallView(context.Background(), gatewayAddress, fixedFeeCall)
	if err != nil {
		return "", fmt.Errorf("failed to call FIXED_BASE_FEE: %w", err)
	}
	fixedVals, err := feeABI.Unpack("FIXED_BASE_FEE", fixedFeeRaw)
	if err != nil || len(fixedVals) == 0 {
		return "", fmt.Errorf("failed to decode FIXED_BASE_FEE")
	}
	fixedFee := fixedVals[0].(*big.Int)

	bpsCall, _ := feeABI.Pack("FEE_RATE_BPS")
	bpsRaw, err := client.CallView(context.Background(), gatewayAddress, bpsCall)
	if err != nil {
		return "", fmt.Errorf("failed to call FEE_RATE_BPS: %w", err)
	}
	bpsVals, err := feeABI.Unpack("FEE_RATE_BPS", bpsRaw)
	if err != nil || len(bpsVals) == 0 {
		return "", fmt.Errorf("failed to decode FEE_RATE_BPS")
	}
	feeBps := bpsVals[0].(*big.Int)

	percentageFee := new(big.Int).Mul(amount, feeBps)
	percentageFee.Div(percentageFee, big.NewInt(10000))
	// fixedFee is now the CAP
	platformFee := new(big.Int).Set(percentageFee)
	if platformFee.Cmp(fixedFee) > 0 {
		platformFee = fixedFee
	}
	onchainTotal := new(big.Int).Add(amount, platformFee)
	if onchainTotal.Cmp(totalCharged) < 0 {
		onchainTotal = totalCharged
	}
	return onchainTotal.String(), nil
}

func (u *PaymentUsecase) quoteGatewayPaymentCost(
	ctx context.Context,
	payment *entities.Payment,
	gatewayAddress string,
	input *entities.CreatePaymentInput,
) (*entities.OnchainCost, error) {
	if payment == nil || gatewayAddress == "" {
		return nil, fmt.Errorf("invalid payment or gateway address")
	}

	sourceChain, err := u.chainRepo.GetByID(ctx, payment.SourceChainID)
	if err != nil || sourceChain == nil {
		return nil, fmt.Errorf("failed to resolve source chain")
	}

	rpcURL := strings.TrimSpace(sourceChain.RPCURL)
	if rpcURL == "" {
		for _, rpc := range sourceChain.RPCs {
			if rpc.IsActive && strings.TrimSpace(rpc.URL) != "" {
				rpcURL = rpc.URL
				break
			}
		}
	}
	if rpcURL == "" {
		return nil, fmt.Errorf("no active source chain rpc url")
	}

	client, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create evm client for payment cost quote: %w", err)
	}

	destChainID := payment.DestChainID.String()
	if payment.DestChain != nil {
		destChainID = payment.DestChain.GetCAIP2ID()
	} else if chain, err := u.chainRepo.GetByID(ctx, payment.DestChainID); err == nil && chain != nil {
		destChainID = chain.GetCAIP2ID()
	}

	minAmountOut := big.NewInt(0)
	if payment.MinDestAmount.Valid {
		if _, ok := minAmountOut.SetString(payment.MinDestAmount.String, 10); !ok {
			minAmountOut = big.NewInt(0)
		}
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(payment.SourceAmount, 10); !ok {
		return nil, fmt.Errorf("invalid source amount")
	}

	feeABI := FallbackPaymentKitaGatewayABI
	if u.ABIResolverMixin != nil {
		resolvedABI, abiErr := u.ResolveABIWithFallback(ctx, payment.SourceChainID, entities.ContractTypeGateway)
		if abiErr != nil {
			// Optional Track-B path; keep legacy behavior when ABI unavailable.
			return nil, nil
		}
		// Some DB ABI rows are stale and can miss Track-B methods.
		// Prefer resolved ABI only when it contains the method we need.
		if _, ok := resolvedABI.Methods["quotePaymentCost"]; ok {
			feeABI = resolvedABI
		}
	}
	if _, ok := feeABI.Methods["quotePaymentCost"]; !ok {
		// Gateway deployed without Track-B quote method.
		return nil, nil
	}

	addressType, err := newABIType("address", "", nil)
	if err != nil {
		return nil, err
	}

	receiverPacked, err := packABIArgs(
		abi.Arguments{{Type: addressType}},
		common.HexToAddress(normalizeEvmAddress(payment.ReceiverAddress)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode receiver bytes: %w", err)
	}

	mode := normalizePaymentMode(nil)
	if input != nil {
		mode = normalizePaymentMode(input.Mode)
	}
	modeByte := uint8(0)
	if mode == "privacy" {
		modeByte = 1
	}
	bridgeOption := BridgeOptionDefaultSentinel
	if input != nil {
		var optErr error
		bridgeOption, optErr = normalizeBridgeOption(input.BridgeOption)
		if optErr != nil {
			return nil, fmt.Errorf("invalid bridge option: %w", optErr)
		}
	}
	minBridgeAmount := big.NewInt(0)
	if input != nil && input.MinBridgeAmountOut != nil {
		raw := strings.TrimSpace(*input.MinBridgeAmountOut)
		if raw != "" {
			if _, ok := minBridgeAmount.SetString(raw, 10); !ok {
				return nil, fmt.Errorf("invalid minBridgeAmountOut")
			}
		}
	}
	if input != nil && input.MinDestAmountOut != nil {
		raw := strings.TrimSpace(*input.MinDestAmountOut)
		if raw != "" {
			if _, ok := minAmountOut.SetString(raw, 10); !ok {
				return nil, fmt.Errorf("invalid minDestAmountOut")
			}
		}
	}
	bridgeTokenSource := common.Address{}
	if input != nil && input.BridgeTokenSource != nil {
		raw := strings.TrimSpace(*input.BridgeTokenSource)
		if raw != "" {
			if !common.IsHexAddress(raw) {
				return nil, fmt.Errorf("invalid bridgeTokenSource")
			}
			bridgeTokenSource = common.HexToAddress(raw)
		}
	}

	req := paymentRequestV2TupleValue{
		DestChainIdBytes:   []byte(destChainID),
		ReceiverBytes:      receiverPacked,
		SourceToken:        common.HexToAddress(normalizeEvmAddress(payment.SourceTokenAddress)),
		BridgeTokenSource:  bridgeTokenSource,
		DestToken:          common.HexToAddress(normalizeEvmAddress(payment.DestTokenAddress)),
		AmountInSource:     amount,
		MinBridgeAmountOut: minBridgeAmount,
		MinDestAmountOut:   minAmountOut,
		Mode:               modeByte,
		BridgeOption:       bridgeOption,
	}

	calldata, err := feeABI.Pack("quotePaymentCost", req)
	if err != nil {
		return nil, fmt.Errorf("failed to pack quotePaymentCost args: %w", err)
	}
	out, err := client.CallView(ctx, gatewayAddress, calldata)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty result from quotePaymentCost")
	}

	decoded, err := feeABI.Unpack("quotePaymentCost", out)
	if err != nil || len(decoded) < 6 {
		return nil, fmt.Errorf("failed to decode quotePaymentCost output")
	}

	platformFee, _ := decoded[0].(*big.Int)
	bridgeFeeNative, _ := decoded[1].(*big.Int)
	totalSourceRequired, _ := decoded[2].(*big.Int)
	bridgeType, _ := decoded[3].(uint8)
	bridgeQuoteOK, _ := decoded[4].(bool)
	bridgeReason, _ := decoded[5].(string)

	result := &entities.OnchainCost{
		PlatformFeeToken:         "0",
		BridgeFeeNative:          "0",
		TotalSourceTokenRequired: "0",
		BridgeType:               bridgeType,
		IsSameChain:              strings.EqualFold(sourceChain.GetCAIP2ID(), destChainID),
		BridgeQuoteOk:            bridgeQuoteOK,
		BridgeQuoteReason:        bridgeReason,
	}
	if platformFee != nil {
		result.PlatformFeeToken = platformFee.String()
	}
	if bridgeFeeNative != nil {
		result.BridgeFeeNative = bridgeFeeNative.String()
	}
	if totalSourceRequired != nil {
		result.TotalSourceTokenRequired = totalSourceRequired.String()
	}
	return result, nil
}

type previewApprovalResult struct {
	ApprovalToken     common.Address
	ApprovalAmount    *big.Int
	RequiredNativeFee *big.Int
}

func (u *PaymentUsecase) previewGatewayApprovalV2(
	ctx context.Context,
	payment *entities.Payment,
	gatewayAddress string,
	input *entities.CreatePaymentInput,
) (*previewApprovalResult, error) {
	if payment == nil || gatewayAddress == "" {
		return nil, fmt.Errorf("invalid payment or gateway address")
	}
	if u.chainRepo == nil || u.clientFactory == nil {
		return nil, fmt.Errorf("previewApproval dependencies are not initialized")
	}

	sourceChain, err := u.chainRepo.GetByID(ctx, payment.SourceChainID)
	if err != nil || sourceChain == nil {
		return nil, fmt.Errorf("failed to resolve source chain")
	}
	rpcURL := strings.TrimSpace(sourceChain.RPCURL)
	if rpcURL == "" {
		for _, rpc := range sourceChain.RPCs {
			if rpc.IsActive && strings.TrimSpace(rpc.URL) != "" {
				rpcURL = rpc.URL
				break
			}
		}
	}
	if rpcURL == "" {
		return nil, fmt.Errorf("no active source chain rpc url")
	}
	client, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create evm client for previewApproval: %w", err)
	}

	feeABI := FallbackPaymentKitaGatewayABI
	if u.ABIResolverMixin != nil {
		resolvedABI, abiErr := u.ResolveABIWithFallback(ctx, payment.SourceChainID, entities.ContractTypeGateway)
		if abiErr != nil {
			return nil, fmt.Errorf("failed to resolve gateway ABI: %w", abiErr)
		}
		// Some DB ABI rows are stale and can miss Track-B methods.
		// Prefer resolved ABI only when it contains the method we need.
		if _, ok := resolvedABI.Methods["previewApproval"]; ok {
			feeABI = resolvedABI
		}
	}
	if _, ok := feeABI.Methods["previewApproval"]; !ok {
		return nil, fmt.Errorf("previewApproval is not available on gateway ABI")
	}

	destChainID := payment.DestChainID.String()
	if payment.DestChain != nil {
		destChainID = payment.DestChain.GetCAIP2ID()
	} else if chain, err := u.chainRepo.GetByID(ctx, payment.DestChainID); err == nil && chain != nil {
		destChainID = chain.GetCAIP2ID()
	}

	minDestAmountOut := big.NewInt(0)
	if payment.MinDestAmount.Valid {
		if _, ok := minDestAmountOut.SetString(payment.MinDestAmount.String, 10); !ok {
			minDestAmountOut = big.NewInt(0)
		}
	}

	amountInSource := new(big.Int)
	if _, ok := amountInSource.SetString(payment.SourceAmount, 10); !ok {
		return nil, fmt.Errorf("invalid source amount")
	}

	addressType, err := newABIType("address", "", nil)
	if err != nil {
		return nil, err
	}
	receiverBytes, err := packABIArgs(
		abi.Arguments{{Type: addressType}},
		common.HexToAddress(normalizeEvmAddress(payment.ReceiverAddress)),
	)
	if err != nil {
		receiverBytes = []byte(payment.ReceiverAddress)
	}

	mode := normalizePaymentMode(nil)
	if input != nil {
		mode = normalizePaymentMode(input.Mode)
	}
	modeByte := uint8(0)
	if mode == "privacy" {
		modeByte = 1
	}
	bridgeOption := BridgeOptionDefaultSentinel
	if input != nil {
		var optErr error
		bridgeOption, optErr = normalizeBridgeOption(input.BridgeOption)
		if optErr != nil {
			return nil, fmt.Errorf("invalid bridge option: %w", optErr)
		}
	}
	minBridgeAmount := big.NewInt(0)
	if input != nil && input.MinBridgeAmountOut != nil {
		raw := strings.TrimSpace(*input.MinBridgeAmountOut)
		if raw != "" {
			if _, ok := minBridgeAmount.SetString(raw, 10); !ok {
				return nil, fmt.Errorf("invalid minBridgeAmountOut")
			}
		}
	}
	if input != nil && input.MinDestAmountOut != nil {
		raw := strings.TrimSpace(*input.MinDestAmountOut)
		if raw != "" {
			if _, ok := minDestAmountOut.SetString(raw, 10); !ok {
				return nil, fmt.Errorf("invalid minDestAmountOut")
			}
		}
	}
	bridgeTokenSource := common.Address{}
	if input != nil && input.BridgeTokenSource != nil {
		raw := strings.TrimSpace(*input.BridgeTokenSource)
		if raw != "" {
			if !common.IsHexAddress(raw) {
				return nil, fmt.Errorf("invalid bridgeTokenSource")
			}
			bridgeTokenSource = common.HexToAddress(raw)
		}
	}

	req := paymentRequestV2TupleValue{
		DestChainIdBytes:   []byte(destChainID),
		ReceiverBytes:      receiverBytes,
		SourceToken:        common.HexToAddress(normalizeEvmAddress(payment.SourceTokenAddress)),
		BridgeTokenSource:  bridgeTokenSource,
		DestToken:          common.HexToAddress(normalizeEvmAddress(payment.DestTokenAddress)),
		AmountInSource:     amountInSource,
		MinBridgeAmountOut: minBridgeAmount,
		MinDestAmountOut:   minDestAmountOut,
		Mode:               modeByte,
		BridgeOption:       bridgeOption,
	}

	calldata, err := feeABI.Pack("previewApproval", req)
	if err != nil {
		return nil, fmt.Errorf("failed to pack previewApproval args: %w", err)
	}
	out, err := client.CallView(ctx, gatewayAddress, calldata)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty result from previewApproval")
	}
	decoded, err := feeABI.Unpack("previewApproval", out)
	if err != nil || len(decoded) < 3 {
		return nil, fmt.Errorf("failed to decode previewApproval output")
	}

	approvalToken, _ := decoded[0].(common.Address)
	approvalAmount, _ := decoded[1].(*big.Int)
	requiredNativeFee, _ := decoded[2].(*big.Int)
	if approvalAmount == nil {
		approvalAmount = big.NewInt(0)
	}
	if requiredNativeFee == nil {
		requiredNativeFee = big.NewInt(0)
	}

	return &previewApprovalResult{
		ApprovalToken:     approvalToken,
		ApprovalAmount:    approvalAmount,
		RequiredNativeFee: requiredNativeFee,
	}, nil
}

func (u *PaymentUsecase) buildEvmPaymentHex(payment *entities.Payment, destChainID string, minAmountOut *big.Int) string {
	return u.buildEvmPaymentHexWithInput(payment, destChainID, minAmountOut, nil)
}

func (u *PaymentUsecase) buildEvmPaymentHexWithInput(
	payment *entities.Payment,
	destChainID string,
	minAmountOut *big.Int,
	input *entities.CreatePaymentInput,
) string {
	// V2-only calldata: createPayment(PaymentRequestV2)
	addressType, err := newABIType("address", "", nil)
	if err != nil {
		return ""
	}
	receiverBytes := []byte(payment.ReceiverAddress)
	if strings.HasPrefix(payment.ReceiverAddress, "0x") && len(payment.ReceiverAddress) == 42 {
		encodedAddress, err := packABIArgs(abi.Arguments{{Type: addressType}}, common.HexToAddress(normalizeEvmAddress(payment.ReceiverAddress)))
		if err == nil {
			receiverBytes = encodedAddress
		}
	}

	amount := new(big.Int)
	amount.SetString(payment.SourceAmount, 10)

	minDest := big.NewInt(0)
	if minAmountOut != nil && minAmountOut.Sign() > 0 {
		minDest = new(big.Int).Set(minAmountOut)
	}
	if input != nil && input.MinDestAmountOut != nil {
		raw := strings.TrimSpace(*input.MinDestAmountOut)
		if raw != "" {
			if _, ok := minDest.SetString(raw, 10); !ok {
				return ""
			}
		}
	}
	minBridge := big.NewInt(0)
	if input != nil && input.MinBridgeAmountOut != nil {
		raw := strings.TrimSpace(*input.MinBridgeAmountOut)
		if raw != "" {
			if _, ok := minBridge.SetString(raw, 10); !ok {
				return ""
			}
		}
	}
	mode := normalizePaymentMode(nil)
	if input != nil {
		mode = normalizePaymentMode(input.Mode)
	}
	modeByte := uint8(0)
	if mode == "privacy" {
		modeByte = 1
	}
	bridgeOption := BridgeOptionDefaultSentinel
	if input != nil {
		opt, optErr := normalizeBridgeOption(input.BridgeOption)
		if optErr != nil {
			return ""
		}
		bridgeOption = opt
	}
	bridgeTokenSource := common.Address{}
	if input != nil && input.BridgeTokenSource != nil {
		raw := strings.TrimSpace(*input.BridgeTokenSource)
		if raw != "" {
			if !common.IsHexAddress(raw) {
				return ""
			}
			bridgeTokenSource = common.HexToAddress(raw)
		}
	}

	req := PaymentRequestV2Args{
		DestChainIDBytes:   []byte(destChainID),
		ReceiverBytes:      receiverBytes,
		SourceToken:        common.HexToAddress(normalizeEvmAddress(payment.SourceTokenAddress)),
		BridgeTokenSource:  bridgeTokenSource,
		DestToken:          common.HexToAddress(normalizeEvmAddress(payment.DestTokenAddress)),
		AmountInSource:     amount,
		MinBridgeAmountOut: minBridge,
		MinDestAmountOut:   minDest,
		Mode:               modeByte,
		BridgeOption:       bridgeOption,
	}

	isDefaultBridgeCall := input != nil && input.BridgeOption == nil
	if mode == "privacy" {
		if input == nil || input.PrivacyIntentID == nil || input.PrivacyStealthReceiver == nil {
			return ""
		}
		privacyIntentRaw := strings.TrimSpace(*input.PrivacyIntentID)
		stealthRaw := strings.TrimSpace(*input.PrivacyStealthReceiver)
		if privacyIntentRaw == "" || stealthRaw == "" || !common.IsHexAddress(stealthRaw) {
			return ""
		}
		intentID := parsePrivacyIntentID(privacyIntentRaw)
		calldata, privateErr := packCreatePaymentPrivateV2Calldata(req, PrivateRoutingArgs{
			IntentID:        intentID,
			StealthReceiver: common.HexToAddress(stealthRaw),
		})
		if privateErr != nil {
			return ""
		}
		return calldata
	}

	if isDefaultBridgeCall {
		calldata, defaultErr := packCreatePaymentDefaultBridgeV2Calldata(req)
		if defaultErr != nil {
			return ""
		}
		return calldata
	}

	calldata, callErr := packCreatePaymentV2Calldata(req)
	if callErr != nil {
		return ""
	}
	return calldata
}

func parsePrivacyIntentID(raw string) [32]byte {
	var out [32]byte
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "0x") && len(trimmed) == 66 {
		copy(out[:], common.FromHex(trimmed))
		return out
	}
	hash := crypto.Keccak256Hash([]byte(trimmed))
	copy(out[:], hash.Bytes())
	return out
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

// GetPaymentPrivacyStatus infers privacy lifecycle stage from persisted payment/events data.
func (u *PaymentUsecase) GetPaymentPrivacyStatus(ctx context.Context, paymentID uuid.UUID) (*entities.PaymentPrivacyStatus, error) {
	payment, err := u.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	events, err := u.paymentEventRepo.GetByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	stage, isPrivacy, signals, reason := derivePaymentPrivacyLifecycle(payment, events)
	if onchainStage, onchainPrivacy, onchainSignals, onchainReason, ok := u.deriveGatewayPrivacyLifecycle(ctx, payment, stage, isPrivacy, reason); ok {
		stage = onchainStage
		isPrivacy = onchainPrivacy
		signals = append(signals, onchainSignals...)
		if strings.TrimSpace(onchainReason) != "" {
			reason = onchainReason
		}
	}
	return &entities.PaymentPrivacyStatus{
		PaymentID:          paymentID,
		Stage:              stage,
		IsPrivacyCandidate: isPrivacy,
		Signals:            signals,
		Reason:             reason,
	}, nil
}

func (u *PaymentUsecase) BuildRetryPrivacyRecoveryTx(
	ctx context.Context,
	paymentID uuid.UUID,
	onchainPaymentID string,
) (*entities.PaymentPrivacyRecoveryTx, error) {
	return u.buildPrivacyRecoveryTx(ctx, paymentID, onchainPaymentID, entities.PrivacyRecoveryActionRetry)
}

func (u *PaymentUsecase) BuildClaimPrivacyRecoveryTx(
	ctx context.Context,
	paymentID uuid.UUID,
	onchainPaymentID string,
) (*entities.PaymentPrivacyRecoveryTx, error) {
	return u.buildPrivacyRecoveryTx(ctx, paymentID, onchainPaymentID, entities.PrivacyRecoveryActionClaim)
}

func (u *PaymentUsecase) BuildRefundPrivacyRecoveryTx(
	ctx context.Context,
	paymentID uuid.UUID,
	onchainPaymentID string,
) (*entities.PaymentPrivacyRecoveryTx, error) {
	return u.buildPrivacyRecoveryTx(ctx, paymentID, onchainPaymentID, entities.PrivacyRecoveryActionRefund)
}

func (u *PaymentUsecase) buildPrivacyRecoveryTx(
	ctx context.Context,
	paymentID uuid.UUID,
	onchainPaymentID string,
	action entities.PrivacyRecoveryAction,
) (*entities.PaymentPrivacyRecoveryTx, error) {
	payment, err := u.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	events, err := u.paymentEventRepo.GetByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	stage, isPrivacy, signals, reason := derivePaymentPrivacyLifecycle(payment, events)
	if !isPrivacy {
		return nil, domainerrors.BadRequest("payment is not in privacy flow")
	}
	if err := validatePrivacyRecoveryStage(stage, action); err != nil {
		return nil, err
	}

	paymentIDBytes32, normalizedPaymentID, err := resolveOnchainPaymentID(events, onchainPaymentID)
	if err != nil {
		return nil, domainerrors.BadRequest(err.Error())
	}

	methodSig, methodSelector, err := privacyRecoveryMethodSpec(action)
	if err != nil {
		return nil, domainerrors.BadRequest(err.Error())
	}
	calldata := methodSelector + hex.EncodeToString(paymentIDBytes32[:])

	gatewayContract, err := u.contractRepo.GetActiveContract(ctx, payment.SourceChainID, entities.ContractTypeGateway)
	if err != nil {
		return nil, fmt.Errorf("active gateway contract not found: %w", err)
	}
	sourceChain, err := u.chainRepo.GetByID(ctx, payment.SourceChainID)
	if err != nil {
		return nil, fmt.Errorf("source chain not found: %w", err)
	}

	return &entities.PaymentPrivacyRecoveryTx{
		Action:           action,
		PaymentID:        paymentID,
		OnchainPaymentID: normalizedPaymentID,
		Stage:            stage,
		ChainID:          sourceChain.GetCAIP2ID(),
		ContractAddress:  gatewayContract.ContractAddress,
		Method:           methodSig,
		Calldata:         calldata,
		Value:            "0",
		Signals:          dedupeSignals(signals),
		Reason:           reason,
	}, nil
}

func derivePaymentPrivacyLifecycle(payment *entities.Payment, events []*entities.PaymentEvent) (string, bool, []string, string) {
	signals := make([]string, 0, 12)
	hasDestSettlement := payment.DestTxHash.Valid && strings.TrimSpace(payment.DestTxHash.String) != ""
	if hasDestSettlement {
		signals = append(signals, "dest_tx_hash_present")
	}

	failure := strings.TrimSpace(payment.FailureReason.String)
	failureUpper := strings.ToUpper(failure)
	hasPrivacySignal := strings.Contains(failureUpper, "PRIVACY")
	hasForwardFailure := strings.Contains(failureUpper, "PRIVACY_FORWARD_FAILED")
	hasForwardCompletedSignal := false
	hasForwardRequestedSignal := false
	hasPrivacyPreparedSignal := false
	hasRetryRequestedSignal := false
	hasEscrowClaimedSignal := false
	hasEscrowRefundedSignal := false
	hasStealthSettlementSignal := false

	if hasPrivacySignal {
		signals = append(signals, "failure_reason_contains_privacy")
	}
	if hasForwardFailure {
		signals = append(signals, "privacy_forward_failed")
	}

	for _, ev := range events {
		evType := strings.ToUpper(strings.TrimSpace(string(ev.EventType)))
		meta := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", ev.Metadata)))
		blob := evType + "|" + meta

		if strings.Contains(blob, "PRIVACY") {
			hasPrivacySignal = true
			signals = append(signals, "event_contains_privacy")
		}
		if strings.Contains(blob, "PRIVACYPAYMENTCREATED") || strings.Contains(blob, "PRIVACY_PAYMENT_CREATED") {
			hasPrivacyPreparedSignal = true
			signals = append(signals, "privacy_prepared")
		}
		if strings.Contains(blob, "PRIVACYFORWARDREQUESTED") || strings.Contains(blob, "PRIVACY_FORWARD_REQUESTED") {
			hasForwardRequestedSignal = true
			signals = append(signals, "privacy_forward_requested")
		}
		if strings.Contains(blob, "PRIVACYFORWARDRETRYREQUESTED") || strings.Contains(blob, "PRIVACY_FORWARD_RETRY_REQUESTED") {
			hasRetryRequestedSignal = true
			signals = append(signals, "privacy_forward_retry_requested")
		}
		if strings.Contains(blob, "PRIVACYFORWARDCOMPLETED") || strings.Contains(blob, "PRIVACY_FORWARD_COMPLETED") {
			hasForwardCompletedSignal = true
			signals = append(signals, "privacy_forward_completed")
		}
		if strings.Contains(blob, "PRIVACYESCROWCLAIMED") || strings.Contains(blob, "PRIVACY_ESCROW_CLAIMED") ||
			strings.Contains(blob, "PRIVACYCLAIMED") || strings.Contains(blob, "PRIVACY_CLAIMED") {
			hasEscrowClaimedSignal = true
			signals = append(signals, "privacy_escrow_claimed")
		}
		if strings.Contains(blob, "PRIVACYESCROWREFUNDED") || strings.Contains(blob, "PRIVACY_ESCROW_REFUNDED") ||
			strings.Contains(blob, "PRIVACYREFUNDED") || strings.Contains(blob, "PRIVACY_REFUNDED") {
			hasEscrowRefundedSignal = true
			signals = append(signals, "privacy_escrow_refunded")
		}
		if evType == "DESTINATION_TX_HASH" || evType == "PAYMENT_EXECUTED" || strings.Contains(blob, "STEALTH") {
			hasStealthSettlementSignal = true
		}
		if strings.Contains(blob, "PRIVACY_FORWARD_FAILED") {
			hasForwardFailure = true
			signals = append(signals, "privacy_forward_failed_event")
		}
	}

	if hasEscrowClaimedSignal || hasEscrowRefundedSignal {
		return entities.PrivacyLifecycleResolved, true, dedupeSignals(signals), ""
	}

	if hasForwardFailure {
		if hasRetryRequestedSignal {
			return entities.PrivacyLifecycleForwardFailedRetrying, true, dedupeSignals(signals), failure
		}
		if payment.Status == entities.PaymentStatusFailed || strings.Contains(failureUpper, "REFUND") {
			return entities.PrivacyLifecycleRefundable, true, dedupeSignals(signals), failure
		}
		return entities.PrivacyLifecycleClaimable, true, dedupeSignals(signals), failure
	}

	if hasRetryRequestedSignal {
		return entities.PrivacyLifecycleForwardFailedRetrying, true, dedupeSignals(signals), failure
	}

	if hasForwardCompletedSignal {
		return entities.PrivacyLifecycleForwardedFinal, true, dedupeSignals(signals), ""
	}

	if hasForwardRequestedSignal {
		return entities.PrivacyLifecycleSettledToStealth, true, dedupeSignals(signals), ""
	}

	if payment.Status == entities.PaymentStatusCompleted && (hasDestSettlement || hasStealthSettlementSignal) {
		if hasPrivacySignal {
			return entities.PrivacyLifecycleSettledToStealth, true, dedupeSignals(signals), "privacy forward confirmation not observed yet"
		}
		return entities.PrivacyLifecycleNotPrivacy, false, dedupeSignals(signals), "payment completed without privacy markers"
	}

	if hasDestSettlement || hasStealthSettlementSignal || payment.Status == entities.PaymentStatusProcessing {
		if hasPrivacySignal {
			return entities.PrivacyLifecycleSettledToStealth, true, dedupeSignals(signals), ""
		}
		return entities.PrivacyLifecycleUnknown, false, dedupeSignals(signals), "settlement signal exists but privacy markers missing"
	}

	if payment.Status == entities.PaymentStatusPending {
		if hasPrivacyPreparedSignal {
			return entities.PrivacyLifecyclePendingOnSource, true, dedupeSignals(signals), ""
		}
		if hasPrivacySignal {
			return entities.PrivacyLifecyclePendingOnSource, true, dedupeSignals(signals), ""
		}
		return entities.PrivacyLifecycleUnknown, false, dedupeSignals(signals), "payment still pending and privacy markers missing"
	}

	if hasPrivacySignal {
		return entities.PrivacyLifecyclePendingOnSource, true, dedupeSignals(signals), ""
	}

	return entities.PrivacyLifecycleUnknown, false, dedupeSignals(signals), "insufficient privacy lifecycle signals"
}

type gatewayPrivacyState struct {
	intentID         [32]byte
	stealthReceiver  common.Address
	forwardCompleted bool
	retryCount       uint8
	settledToken     common.Address
	settledAmount    *big.Int
}

func (u *PaymentUsecase) deriveGatewayPrivacyLifecycle(
	ctx context.Context,
	payment *entities.Payment,
	currentStage string,
	currentPrivacy bool,
	currentReason string,
) (string, bool, []string, string, bool) {
	if payment == nil || u.chainRepo == nil || u.contractRepo == nil || u.clientFactory == nil {
		return "", false, nil, "", false
	}

	sourceChain, err := u.chainRepo.GetByID(ctx, payment.SourceChainID)
	if err != nil || sourceChain == nil || sourceChain.Type != entities.ChainTypeEVM {
		return "", false, nil, "", false
	}

	gatewayContract, err := u.contractRepo.GetActiveContract(ctx, payment.SourceChainID, entities.ContractTypeGateway)
	if err != nil || gatewayContract == nil || !common.IsHexAddress(gatewayContract.ContractAddress) {
		return "", false, nil, "", false
	}

	rpcURL := resolveRPCURL(sourceChain)
	if rpcURL == "" {
		return "", false, nil, "", false
	}

	client, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return "", false, nil, "", false
	}
	defer client.Close()

	state, err := fetchGatewayPrivacyState(ctx, client, gatewayContract.ContractAddress, uuidToBytes32(payment.ID))
	if err != nil {
		return "", false, nil, "", false
	}

	var zeroBytes32 [32]byte
	hasPrivacyIntent := state.intentID != zeroBytes32
	hasStealthReceiver := state.stealthReceiver != (common.Address{})
	hasSettlement := state.settledToken != (common.Address{}) && state.settledAmount != nil && state.settledAmount.Sign() > 0
	hasRetryFailure := state.retryCount > 0
	isPrivacy := currentPrivacy || hasPrivacyIntent || hasStealthReceiver || state.forwardCompleted || hasRetryFailure
	if !isPrivacy {
		return "", false, nil, "", false
	}

	signals := make([]string, 0, 6)
	if hasPrivacyIntent {
		signals = append(signals, "onchain_privacy_intent_present")
	}
	if hasStealthReceiver {
		signals = append(signals, "onchain_privacy_stealth_receiver_present")
	}
	if hasSettlement {
		signals = append(signals, "onchain_privacy_settlement_present")
	}
	if state.forwardCompleted {
		signals = append(signals, "onchain_privacy_forward_completed")
	}
	if hasRetryFailure {
		signals = append(signals, "onchain_privacy_forward_retry_count_present")
	}

	switch {
	case state.forwardCompleted:
		return entities.PrivacyLifecycleForwardedFinal, true, dedupeSignals(signals), "", true
	case hasRetryFailure:
		failureUpper := strings.ToUpper(strings.TrimSpace(payment.FailureReason.String))
		if payment.Status == entities.PaymentStatusFailed ||
			payment.Status == entities.PaymentStatusRefunded ||
			strings.Contains(failureUpper, "REFUND") {
			return entities.PrivacyLifecycleRefundable, true, dedupeSignals(signals), currentReason, true
		}
		if currentStage == entities.PrivacyLifecycleForwardFailedRetrying ||
			strings.Contains(failureUpper, "RETRY") {
			return entities.PrivacyLifecycleForwardFailedRetrying, true, dedupeSignals(signals), currentReason, true
		}
		return entities.PrivacyLifecycleClaimable, true, dedupeSignals(signals), currentReason, true
	case hasSettlement:
		reason := currentReason
		if strings.TrimSpace(reason) == "" && payment.Status == entities.PaymentStatusCompleted {
			reason = "privacy forward confirmation not observed yet"
		}
		return entities.PrivacyLifecycleSettledToStealth, true, dedupeSignals(signals), reason, true
	default:
		return entities.PrivacyLifecyclePendingOnSource, true, dedupeSignals(signals), currentReason, true
	}
}

func fetchGatewayPrivacyState(
	ctx context.Context,
	client *blockchain.EVMClient,
	gatewayAddress string,
	paymentID [32]byte,
) (*gatewayPrivacyState, error) {
	intentID, err := callGatewayBytes32Getter(ctx, client, gatewayAddress, "privacyIntentByPayment(bytes32)", paymentID)
	if err != nil {
		return nil, err
	}
	stealthReceiver, err := callGatewayAddressGetter(ctx, client, gatewayAddress, "privacyStealthByPayment(bytes32)", paymentID)
	if err != nil {
		return nil, err
	}
	forwardCompleted, err := callGatewayBoolGetter(ctx, client, gatewayAddress, "privacyForwardCompleted(bytes32)", paymentID)
	if err != nil {
		return nil, err
	}
	retryCount, err := callGatewayUint8Getter(ctx, client, gatewayAddress, "privacyForwardRetryCount(bytes32)", paymentID)
	if err != nil {
		return nil, err
	}
	settledToken, err := callGatewayAddressGetter(ctx, client, gatewayAddress, "paymentSettledToken(bytes32)", paymentID)
	if err != nil {
		return nil, err
	}
	settledAmount, err := callGatewayUint256Getter(ctx, client, gatewayAddress, "paymentSettledAmount(bytes32)", paymentID)
	if err != nil {
		return nil, err
	}

	return &gatewayPrivacyState{
		intentID:         intentID,
		stealthReceiver:  stealthReceiver,
		forwardCompleted: forwardCompleted,
		retryCount:       retryCount,
		settledToken:     settledToken,
		settledAmount:    settledAmount,
	}, nil
}

func callGatewayBytes32Getter(
	ctx context.Context,
	client *blockchain.EVMClient,
	gatewayAddress string,
	methodSig string,
	paymentID [32]byte,
) ([32]byte, error) {
	out, err := callGatewayPaymentGetter(ctx, client, gatewayAddress, methodSig, paymentID)
	if err != nil {
		return [32]byte{}, err
	}
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return [32]byte{}, err
	}
	values, err := abi.Arguments{{Type: bytes32Type}}.Unpack(out)
	if err != nil {
		return [32]byte{}, err
	}
	if len(values) != 1 {
		return [32]byte{}, fmt.Errorf("invalid bytes32 getter response")
	}
	switch v := values[0].(type) {
	case [32]byte:
		return v, nil
	case []byte:
		var out32 [32]byte
		copy(out32[:], v)
		return out32, nil
	default:
		return [32]byte{}, fmt.Errorf("invalid bytes32 getter type")
	}
}

func callGatewayAddressGetter(
	ctx context.Context,
	client *blockchain.EVMClient,
	gatewayAddress string,
	methodSig string,
	paymentID [32]byte,
) (common.Address, error) {
	out, err := callGatewayPaymentGetter(ctx, client, gatewayAddress, methodSig, paymentID)
	if err != nil {
		return common.Address{}, err
	}
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return common.Address{}, err
	}
	values, err := abi.Arguments{{Type: addressType}}.Unpack(out)
	if err != nil {
		return common.Address{}, err
	}
	if len(values) != 1 {
		return common.Address{}, fmt.Errorf("invalid address getter response")
	}
	addressValue, ok := values[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("invalid address getter type")
	}
	return addressValue, nil
}

func callGatewayBoolGetter(
	ctx context.Context,
	client *blockchain.EVMClient,
	gatewayAddress string,
	methodSig string,
	paymentID [32]byte,
) (bool, error) {
	out, err := callGatewayPaymentGetter(ctx, client, gatewayAddress, methodSig, paymentID)
	if err != nil {
		return false, err
	}
	boolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return false, err
	}
	values, err := abi.Arguments{{Type: boolType}}.Unpack(out)
	if err != nil {
		return false, err
	}
	if len(values) != 1 {
		return false, fmt.Errorf("invalid bool getter response")
	}
	boolValue, ok := values[0].(bool)
	if !ok {
		return false, fmt.Errorf("invalid bool getter type")
	}
	return boolValue, nil
}

func callGatewayUint8Getter(
	ctx context.Context,
	client *blockchain.EVMClient,
	gatewayAddress string,
	methodSig string,
	paymentID [32]byte,
) (uint8, error) {
	out, err := callGatewayPaymentGetter(ctx, client, gatewayAddress, methodSig, paymentID)
	if err != nil {
		return 0, err
	}
	uint8Type, err := abi.NewType("uint8", "", nil)
	if err != nil {
		return 0, err
	}
	values, err := abi.Arguments{{Type: uint8Type}}.Unpack(out)
	if err != nil {
		return 0, err
	}
	if len(values) != 1 {
		return 0, fmt.Errorf("invalid uint8 getter response")
	}
	switch v := values[0].(type) {
	case uint8:
		return v, nil
	case *big.Int:
		return uint8(v.Uint64()), nil
	default:
		return 0, fmt.Errorf("invalid uint8 getter type")
	}
}

func callGatewayUint256Getter(
	ctx context.Context,
	client *blockchain.EVMClient,
	gatewayAddress string,
	methodSig string,
	paymentID [32]byte,
) (*big.Int, error) {
	out, err := callGatewayPaymentGetter(ctx, client, gatewayAddress, methodSig, paymentID)
	if err != nil {
		return nil, err
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	values, err := abi.Arguments{{Type: uint256Type}}.Unpack(out)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("invalid uint256 getter response")
	}
	switch v := values[0].(type) {
	case *big.Int:
		return v, nil
	case uint64:
		return new(big.Int).SetUint64(v), nil
	default:
		return nil, fmt.Errorf("invalid uint256 getter type")
	}
}

func callGatewayPaymentGetter(
	ctx context.Context,
	client *blockchain.EVMClient,
	gatewayAddress string,
	methodSig string,
	paymentID [32]byte,
) ([]byte, error) {
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	packedArgs, err := abi.Arguments{{Type: bytes32Type}}.Pack(paymentID)
	if err != nil {
		return nil, err
	}
	methodID := crypto.Keccak256([]byte(methodSig))[:4]
	return client.CallView(ctx, gatewayAddress, append(methodID, packedArgs...))
}

func validatePrivacyRecoveryStage(stage string, action entities.PrivacyRecoveryAction) error {
	switch action {
	case entities.PrivacyRecoveryActionRetry:
		if stage != entities.PrivacyLifecycleForwardFailedRetrying &&
			stage != entities.PrivacyLifecycleClaimable &&
			stage != entities.PrivacyLifecycleRefundable {
			return domainerrors.BadRequest("retry is only allowed after forward failure")
		}
	case entities.PrivacyRecoveryActionClaim:
		if stage != entities.PrivacyLifecycleClaimable &&
			stage != entities.PrivacyLifecycleForwardFailedRetrying &&
			stage != entities.PrivacyLifecycleSettledToStealth {
			return domainerrors.BadRequest("claim is not available for current privacy stage")
		}
	case entities.PrivacyRecoveryActionRefund:
		if stage != entities.PrivacyLifecycleRefundable &&
			stage != entities.PrivacyLifecycleForwardFailedRetrying {
			return domainerrors.BadRequest("refund is not available for current privacy stage")
		}
	default:
		return domainerrors.BadRequest("invalid privacy recovery action")
	}
	return nil
}

func parseOnchainPaymentID(raw string) ([32]byte, string, error) {
	var out [32]byte

	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(strings.ToLower(trimmed), "0x")
	if len(trimmed) != 64 {
		return out, "", fmt.Errorf("onchainPaymentId must be 32-byte hex")
	}
	decoded, err := hex.DecodeString(trimmed)
	if err != nil {
		return out, "", fmt.Errorf("onchainPaymentId must be valid hex")
	}
	copy(out[:], decoded)
	return out, "0x" + hex.EncodeToString(out[:]), nil
}

func resolveOnchainPaymentID(events []*entities.PaymentEvent, raw string) ([32]byte, string, error) {
	trimmedInput := strings.TrimSpace(raw)
	if trimmedInput != "" {
		return parseOnchainPaymentID(trimmedInput)
	}

	for i := len(events) - 1; i >= 0; i-- {
		candidate := extractOnchainPaymentIDFromMetadata(events[i].Metadata, 0)
		if candidate == "" {
			continue
		}
		parsed, normalized, err := parseOnchainPaymentID(candidate)
		if err == nil {
			return parsed, normalized, nil
		}
	}

	var empty [32]byte
	return empty, "", fmt.Errorf("onchainPaymentId is required (not found in request or event metadata)")
}

func extractOnchainPaymentIDFromMetadata(metadata interface{}, depth int) string {
	if metadata == nil || depth > 4 {
		return ""
	}

	switch v := metadata.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return ""
		}
		if looksLikeBytes32Hex(trimmed) {
			return trimmed
		}
		var decoded interface{}
		if json.Unmarshal([]byte(trimmed), &decoded) == nil {
			return extractOnchainPaymentIDFromMetadata(decoded, depth+1)
		}
		return ""
	case map[string]interface{}:
		keys := []string{
			"onchainPaymentId", "onchain_payment_id",
			"paymentIdHex", "payment_id_hex",
			"paymentIdBytes32", "payment_id_bytes32",
			"onchainPaymentID", "onchain_paymentID",
		}
		for _, key := range keys {
			if rawValue, ok := v[key]; ok {
				if candidate := extractOnchainPaymentIDFromMetadata(rawValue, depth+1); candidate != "" {
					return candidate
				}
			}
		}
		for _, rawValue := range v {
			if candidate := extractOnchainPaymentIDFromMetadata(rawValue, depth+1); candidate != "" {
				return candidate
			}
		}
	case []interface{}:
		for _, rawValue := range v {
			if candidate := extractOnchainPaymentIDFromMetadata(rawValue, depth+1); candidate != "" {
				return candidate
			}
		}
	}
	return ""
}

func looksLikeBytes32Hex(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	trimmed = strings.TrimPrefix(trimmed, "0x")
	if len(trimmed) != 64 {
		return false
	}
	_, err := hex.DecodeString(trimmed)
	return err == nil
}

func privacyRecoveryMethodSpec(action entities.PrivacyRecoveryAction) (methodSig string, selector string, err error) {
	switch action {
	case entities.PrivacyRecoveryActionRetry:
		return "retryPrivacyForward(bytes32)", computeSelectorHex("retryPrivacyForward(bytes32)"), nil
	case entities.PrivacyRecoveryActionClaim:
		return "claimPrivacyEscrow(bytes32)", computeSelectorHex("claimPrivacyEscrow(bytes32)"), nil
	case entities.PrivacyRecoveryActionRefund:
		return "refundPrivacyEscrow(bytes32)", computeSelectorHex("refundPrivacyEscrow(bytes32)"), nil
	default:
		return "", "", fmt.Errorf("unknown privacy recovery action")
	}
}

func dedupeSignals(signals []string) []string {
	if len(signals) == 0 {
		return signals
	}
	seen := make(map[string]struct{}, len(signals))
	unique := make([]string, 0, len(signals))
	for _, signal := range signals {
		if signal == "" {
			continue
		}
		if _, ok := seen[signal]; ok {
			continue
		}
		seen[signal] = struct{}{}
		unique = append(unique, signal)
	}
	return unique
}

// getBridgeFeeQuote fetches fee from on-chain Router
func (u *PaymentUsecase) getBridgeFeeQuote(
	ctx context.Context,
	sourceChainID, destChainID string,
	sourceTokenAddress, destTokenAddress string,
	amount *big.Int,
	minAmountOut *big.Int,
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
	routerAddress := router.ContractAddress

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

	bridgeOrder := u.resolveBridgeOrder(ctx, sourceChainUUID, destChainUUID, sourceCAIP2, destCAIP2)

	// Prefer gateway on-chain routing config to avoid FE/BE policy drift.
	if gateway, gwErr := u.contractRepo.GetActiveContract(ctx, chain.ID, entities.ContractTypeGateway); gwErr == nil && gateway != nil {
		if gwRouter, rErr := u.readGatewayRouterAddress(ctx, client, gateway.ContractAddress); rErr == nil && gwRouter != "" {
			routerAddress = gwRouter
		}
		if gwBridgeType, bErr := u.readGatewayDefaultBridgeType(ctx, client, gateway.ContractAddress, destCAIP2); bErr == nil {
			bridgeOrder = []uint8{gwBridgeType}
		}
	}

	var lastErr error
	for _, bridgeType := range bridgeOrder {
		fee, err := u.quoteBridgeFeeByType(ctx, client, routerAddress, destCAIP2, bridgeType, sourceTokenAddress, destTokenAddress, amount, minAmountOut)
		if err == nil && fee != nil && fee.Sign() > 0 {
			return fee, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("invalid fee quote for bridge type %d", bridgeType)
		}
	}
	return nil, lastErr
}

func (u *PaymentUsecase) readGatewayRouterAddress(
	ctx context.Context,
	client *blockchain.EVMClient,
	gatewayAddress string,
) (string, error) {
	selector := crypto.Keccak256([]byte("router()"))[:4]
	out, err := client.CallView(ctx, gatewayAddress, selector)
	if err != nil {
		return "", err
	}
	if len(out) < 32 {
		return "", fmt.Errorf("invalid router() response")
	}
	addr := common.BytesToAddress(out[len(out)-20:]).Hex()
	if strings.TrimSpace(addr) == "" || strings.EqualFold(addr, common.Address{}.Hex()) {
		return "", fmt.Errorf("gateway router is zero address")
	}
	return addr, nil
}

func (u *PaymentUsecase) readGatewayDefaultBridgeType(
	ctx context.Context,
	client *blockchain.EVMClient,
	gatewayAddress string,
	destCAIP2 string,
) (uint8, error) {
	stringType, err := newABIType("string", "", nil)
	if err != nil {
		return 0, err
	}
	args := abi.Arguments{{Type: stringType}}
	packedArgs, err := packABIArgs(args, destCAIP2)
	if err != nil {
		return 0, err
	}
	methodID := crypto.Keccak256([]byte("defaultBridgeTypes(string)"))[:4]
	calldata := append(methodID, packedArgs...)
	out, err := client.CallView(ctx, gatewayAddress, calldata)
	if err != nil {
		return 0, err
	}
	if len(out) == 0 {
		return 0, fmt.Errorf("empty defaultBridgeTypes response")
	}
	value := new(big.Int).SetBytes(out)
	if !value.IsUint64() {
		return 0, fmt.Errorf("invalid default bridge type")
	}
	v := value.Uint64()
	if v > 255 {
		return 0, fmt.Errorf("default bridge type out of range")
	}
	return uint8(v), nil
}

func (u *PaymentUsecase) resolveBridgeOrder(
	ctx context.Context,
	sourceChainUUID, destChainUUID uuid.UUID,
	sourceCAIP2, destCAIP2 string,
) []uint8 {
	if u.routePolicyRepo != nil {
		if policy, err := u.routePolicyRepo.GetByRoute(ctx, sourceChainUUID, destChainUUID); err == nil && policy != nil {
			return buildBridgeOrderFromPolicy(policy)
		} else if errors.Is(err, domainerrors.ErrNotFound) {
			u.bootstrapDefaultRoutePolicy(ctx, sourceChainUUID, destChainUUID, sourceCAIP2, destCAIP2)
		}
	}

	bridgeTypeStr, _ := u.decideBridge(ctx, sourceChainUUID, destChainUUID, sourceCAIP2, destCAIP2)
	return []uint8{bridgeNameToType(bridgeTypeStr)}
}

func buildBridgeOrderFromPolicy(policy *entities.RoutePolicy) []uint8 {
	if policy == nil {
		return []uint8{0}
	}

	added := make(map[uint8]struct{}, 3)
	order := make([]uint8, 0, 4)
	add := func(bridgeType uint8) {
		if bridgeType > 3 {
			return
		}
		if _, ok := added[bridgeType]; ok {
			return
		}
		added[bridgeType] = struct{}{}
		order = append(order, bridgeType)
	}

	add(policy.DefaultBridgeType)
	if policy.FallbackMode == entities.BridgeFallbackModeAutoFallback {
		for _, bridgeType := range policy.FallbackOrder {
			add(bridgeType)
		}
	}
	if len(order) == 0 {
		return []uint8{0}
	}
	return order
}

func (u *PaymentUsecase) quoteBridgeFeeByType(
	ctx context.Context,
	client *blockchain.EVMClient,
	routerAddress string,
	destCAIP2 string,
	bridgeType uint8,
	sourceTokenAddress, destTokenAddress string,
	amount *big.Int,
	minAmountOut *big.Int,
) (*big.Int, error) {
	routeConfigured, routeCheckErr := u.checkRouterRouteConfigured(ctx, client, routerAddress, destCAIP2, bridgeType)
	// Backward-compatible: older routers may not expose isRouteConfigured.
	// In that case continue and rely on quote/hasAdapter checks.
	if routeCheckErr == nil && !routeConfigured {
		return nil, fmt.Errorf("route not configured for %s bridge type %d", destCAIP2, bridgeType)
	}

	hasAdapter, hasAdapterErr := u.checkRouterHasAdapter(ctx, client, routerAddress, destCAIP2, bridgeType)
	if hasAdapterErr == nil && !hasAdapter {
		return nil, fmt.Errorf("adapter not registered for %s bridge type %d", destCAIP2, bridgeType)
	}

	stringType, err := newABIType("string", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build ABI string type: %w", err)
	}
	uint8Type, err := newABIType("uint8", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build ABI uint8 type: %w", err)
	}
	// ABI encode quotePaymentFee with BridgeMessage tuple.
	// New schema (v2): (bytes32,address,address,address,uint256,string,uint256,address payer)
	messageTupleTypeV2, err := newABIType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "paymentId", Type: "bytes32"},
		{Name: "receiver", Type: "address"},
		{Name: "sourceToken", Type: "address"},
		{Name: "destToken", Type: "address"},
		{Name: "amount", Type: "uint256"},
		{Name: "destChainId", Type: "string"},
		{Name: "minAmountOut", Type: "uint256"},
		{Name: "payer", Type: "address"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build ABI tuple type: %w", err)
	}
	// Legacy schema (v1): without payer
	messageTupleTypeV1, err := newABIType("tuple", "", []abi.ArgumentMarshaling{
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
	argsV2 := abi.Arguments{
		{Type: stringType},
		{Type: uint8Type},
		{Type: messageTupleTypeV2},
	}
	argsV1 := abi.Arguments{
		{Type: stringType},
		{Type: uint8Type},
		{Type: messageTupleTypeV1},
	}

	type bridgeMessageV2 struct {
		PaymentId    [32]byte
		Receiver     common.Address
		SourceToken  common.Address
		DestToken    common.Address
		Amount       *big.Int
		DestChainId  string
		MinAmountOut *big.Int
		Payer        common.Address
	}
	type bridgeMessageV1 struct {
		PaymentId    [32]byte
		Receiver     common.Address
		SourceToken  common.Address
		DestToken    common.Address
		Amount       *big.Int
		DestChainId  string
		MinAmountOut *big.Int
	}
	msgStructV2 := bridgeMessageV2{
		PaymentId:    [32]byte{},
		Receiver:     common.Address{},
		SourceToken:  common.HexToAddress(normalizeEvmAddress(sourceTokenAddress)),
		DestToken:    common.HexToAddress(normalizeEvmAddress(destTokenAddress)),
		Amount:       amount,
		DestChainId:  destCAIP2,
		MinAmountOut: minAmountOut,
		Payer:        common.Address{},
	}
	msgStructV1 := bridgeMessageV1{
		PaymentId:    [32]byte{},
		Receiver:     common.Address{},
		SourceToken:  common.HexToAddress(normalizeEvmAddress(sourceTokenAddress)),
		DestToken:    common.HexToAddress(normalizeEvmAddress(destTokenAddress)),
		Amount:       amount,
		DestChainId:  destCAIP2,
		MinAmountOut: minAmountOut,
	}

	packedArgsV2, err := packABIArgs(argsV2, destCAIP2, bridgeType, msgStructV2)
	if err != nil {
		return nil, fmt.Errorf("failed to pack quotePaymentFee args: %w", err)
	}
	packedArgsV1, err := packABIArgs(argsV1, destCAIP2, bridgeType, msgStructV1)
	if err != nil {
		return nil, fmt.Errorf("failed to pack quotePaymentFee args: %w", err)
	}

	// 6. Call Contract
	// Try quotePaymentFeeSafe first (non-reverting) with v2 schema.
	// (bool success, uint256 fee, string reason)
	safeSigV2 := []byte("quotePaymentFeeSafe(string,uint8,(bytes32,address,address,address,uint256,string,uint256,address))")
	safeMethodIDV2 := crypto.Keccak256(safeSigV2)[:4]
	safeCalldataV2 := append(safeMethodIDV2, packedArgsV2...)

	safeResult, safeErr := client.CallView(ctx, routerAddress, safeCalldataV2)
	if safeErr == nil && len(safeResult) >= 96 {
		ok, fee, reason, err := decodeSafeQuoteResult(safeResult)
		if err == nil {
			if !ok {
				if isQuoteSchemaMismatchReason(reason) {
					return nil, fmt.Errorf("quote failed: quote_failed_schema_mismatch")
				}
				return nil, fmt.Errorf("quote failed: %s", reason)
			}
			return fee, nil
		}
		// If decoding fails, fall through to legacy method
	}

	// Fallback order:
	// 1) reverting quotePaymentFee with v2 schema
	// 2) reverting quotePaymentFee with legacy v1 schema
	methodSigV2 := []byte("quotePaymentFee(string,uint8,(bytes32,address,address,address,uint256,string,uint256,address))")
	methodIDV2 := crypto.Keccak256(methodSigV2)[:4]
	calldataV2 := append(methodIDV2, packedArgsV2...)

	result, err := client.CallView(ctx, routerAddress, calldataV2)
	if err != nil {
		methodSigV1 := []byte("quotePaymentFee(string,uint8,(bytes32,address,address,address,uint256,string,uint256))")
		methodIDV1 := crypto.Keccak256(methodSigV1)[:4]
		calldataV1 := append(methodIDV1, packedArgsV1...)

		result, err = client.CallView(ctx, routerAddress, calldataV1)
		if err != nil {
			if decoded, ok := decodeRevertDataFromError(err); ok {
				if decoded.Selector != "" {
					return nil, fmt.Errorf(
						"contract call failed: %w (decoded_revert=%s selector=%s)",
						err,
						decoded.Message,
						decoded.Selector,
					)
				}
				return nil, fmt.Errorf("contract call failed: %w (decoded_revert=%s)", err, decoded.Message)
			}
			if isQuoteSchemaMismatchReason(err.Error()) {
				return nil, fmt.Errorf("quote failed: quote_failed_schema_mismatch")
			}
			return nil, fmt.Errorf("contract call failed: %w", err)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("quote failed: quote_failed_schema_mismatch (empty result from quotePaymentFee)")
	}
	// 7. Decode Result (uint256)
	fee := new(big.Int).SetBytes(result)

	return fee, nil
}

func (u *PaymentUsecase) checkRouterHasAdapter(
	ctx context.Context,
	client *blockchain.EVMClient,
	routerAddress string,
	destCAIP2 string,
	bridgeType uint8,
) (bool, error) {
	stringType, err := newABIType("string", "", nil)
	if err != nil {
		return false, err
	}
	uint8Type, err := newABIType("uint8", "", nil)
	if err != nil {
		return false, err
	}
	args := abi.Arguments{{Type: stringType}, {Type: uint8Type}}
	packedArgs, err := packABIArgs(args, destCAIP2, bridgeType)
	if err != nil {
		return false, err
	}
	methodID := crypto.Keccak256([]byte("hasAdapter(string,uint8)"))[:4]
	calldata := append(methodID, packedArgs...)
	out, err := client.CallView(ctx, routerAddress, calldata)
	if err != nil {
		return false, err
	}
	if len(out) == 0 {
		return false, fmt.Errorf("empty hasAdapter result")
	}
	value := new(big.Int).SetBytes(out)
	return value.Sign() > 0, nil
}

func (u *PaymentUsecase) checkRouterRouteConfigured(
	ctx context.Context,
	client *blockchain.EVMClient,
	routerAddress string,
	destCAIP2 string,
	bridgeType uint8,
) (bool, error) {
	stringType, err := newABIType("string", "", nil)
	if err != nil {
		return false, err
	}
	uint8Type, err := newABIType("uint8", "", nil)
	if err != nil {
		return false, err
	}
	args := abi.Arguments{{Type: stringType}, {Type: uint8Type}}
	packedArgs, err := packABIArgs(args, destCAIP2, bridgeType)
	if err != nil {
		return false, err
	}
	methodID := crypto.Keccak256([]byte("isRouteConfigured(string,uint8)"))[:4]
	calldata := append(methodID, packedArgs...)
	out, err := client.CallView(ctx, routerAddress, calldata)
	if err != nil {
		return false, err
	}
	if len(out) == 0 {
		return false, fmt.Errorf("empty isRouteConfigured result")
	}
	value := new(big.Int).SetBytes(out)
	return value.Sign() > 0, nil
}

func bridgeTypeToName(bridgeType uint8) string {
	switch bridgeType {
	case 0:
		return "Hyperbridge"
	case 1:
		return "CCIP"
	case 2:
		return "Stargate"
	case 3:
		return "HyperbridgeTokenGateway"
	default:
		return "Hyperbridge"
	}
}

func bridgeNameToType(bridgeName string) uint8 {
	switch strings.ToUpper(strings.TrimSpace(bridgeName)) {
	case "CCIP":
		return 1
	case "STARGATE":
		return 2
	case "HYPERBRIDGE_TOKEN_GATEWAY", "HYPERBRIDGETOKENGATEWAY", "HBTOKENGATEWAY":
		return 3
	default:
		return 0
	}
}

func (u *PaymentUsecase) bootstrapDefaultRoutePolicy(
	ctx context.Context,
	sourceChainUUID, destChainUUID uuid.UUID,
	sourceCAIP2, destCAIP2 string,
) {
	if u.routePolicyRepo == nil {
		return
	}
	defaultBridge := bridgeNameToType(u.SelectBridge(sourceCAIP2, destCAIP2))
	_ = u.routePolicyRepo.Create(ctx, &entities.RoutePolicy{
		ID:                utils.GenerateUUIDv7(),
		SourceChainID:     sourceChainUUID,
		DestChainID:       destChainUUID,
		DefaultBridgeType: defaultBridge,
		FallbackMode:      entities.BridgeFallbackModeStrict,
		FallbackOrder:     []uint8{defaultBridge},
	})
}

// getSwapQuote fetches real-time swap price from TokenSwapper
func (u *PaymentUsecase) getSwapQuote(
	ctx context.Context,
	chainID uuid.UUID,
	tokenIn, tokenOut string,
	amountIn *big.Int,
) (*big.Int, error) {
	if tokenIn == tokenOut {
		return new(big.Int).Set(amountIn), nil
	}

	chain, err := u.chainRepo.GetByID(ctx, chainID)
	if err != nil {
		return nil, err
	}

	swapper, err := u.contractRepo.GetActiveContract(ctx, chain.ID, entities.ContractTypeTokenSwapper)
	if err != nil {
		return nil, fmt.Errorf("active swapper not found")
	}

	client, err := u.clientFactory.GetEVMClient(chain.RPCURL)
	if err != nil {
		return nil, err
	}

	swapperABI, err := u.ResolveABIWithFallback(ctx, chain.ID, entities.ContractTypeTokenSwapper)
	if err != nil {
		return nil, err
	}

	quoteCall, err := swapperABI.Pack("getRealQuote", common.HexToAddress(tokenIn), common.HexToAddress(tokenOut), amountIn)
	if err != nil {
		return nil, err
	}

	out, err := client.CallView(ctx, swapper.ContractAddress, quoteCall)
	if err != nil {
		return nil, err
	}

	results, err := swapperABI.Unpack("getRealQuote", out)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("failed to unpack quote")
	}

	if amountOut, ok := results[0].(*big.Int); ok {
		return amountOut, nil
	}

	return nil, fmt.Errorf("invalid quote result")
}

type TokenRouteSupportStatus struct {
	Exists       bool     `json:"exists"`
	IsDirect     bool     `json:"isDirect"`
	Path         []string `json:"path"`
	Executable   bool     `json:"executable"`
	Reasons      []string `json:"reasons,omitempty"`
	SwapRouterV3 string   `json:"swapRouterV3,omitempty"`
	UniversalV4  string   `json:"universalRouter,omitempty"`
}

// CheckRouteSupport checks if a route exists for a token pair on-chain.
// Backward-compatible wrapper; use CheckRouteSupportDetailed for readiness diagnostics.
func (u *PaymentUsecase) CheckRouteSupport(
	ctx context.Context,
	chainID uuid.UUID,
	tokenIn, tokenOut string,
) (bool, bool, []string, error) {
	status, err := u.CheckRouteSupportDetailed(ctx, chainID, tokenIn, tokenOut)
	if err != nil {
		return false, false, nil, err
	}
	return status.Exists, status.IsDirect, status.Path, nil
}

// CheckRouteSupportDetailed returns route registration + execution readiness signals.
func (u *PaymentUsecase) CheckRouteSupportDetailed(
	ctx context.Context,
	chainID uuid.UUID,
	tokenIn, tokenOut string,
) (*TokenRouteSupportStatus, error) {
	if tokenIn == tokenOut {
		return &TokenRouteSupportStatus{
			Exists:     true,
			IsDirect:   true,
			Path:       []string{tokenIn},
			Executable: true,
		}, nil
	}

	chain, err := u.chainRepo.GetByID(ctx, chainID)
	if err != nil {
		return nil, err
	}

	swapper, err := u.contractRepo.GetActiveContract(ctx, chain.ID, entities.ContractTypeTokenSwapper)
	if err != nil {
		return nil, fmt.Errorf("active swapper not found")
	}

	client, err := u.clientFactory.GetEVMClient(chain.RPCURL)
	if err != nil {
		return nil, err
	}

	swapperABI, err := u.ResolveABIWithFallback(ctx, chain.ID, entities.ContractTypeTokenSwapper)
	if err != nil {
		return nil, err
	}

	findRouteCall, err := swapperABI.Pack("findRoute", common.HexToAddress(tokenIn), common.HexToAddress(tokenOut))
	if err != nil {
		return nil, err
	}

	out, err := client.CallView(ctx, swapper.ContractAddress, findRouteCall)
	if err != nil {
		return nil, err
	}

	results, err := swapperABI.Unpack("findRoute", out)
	if err != nil || len(results) < 3 {
		return nil, fmt.Errorf("failed to unpack findRoute")
	}

	exists, ok1 := results[0].(bool)
	isDirect, ok2 := results[1].(bool)
	pathAddrs, ok3 := results[2].([]common.Address)
	if !ok1 || !ok2 || !ok3 {
		return nil, fmt.Errorf("failed to type cast findRoute results")
	}

	path := make([]string, len(pathAddrs))
	for i, addr := range pathAddrs {
		path[i] = addr.Hex()
	}

	status := &TokenRouteSupportStatus{
		Exists:     exists,
		IsDirect:   isDirect,
		Path:       path,
		Executable: false,
	}

	if !exists {
		status.Reasons = append(status.Reasons, "NO_ROUTE_REGISTERED")
		return status, nil
	}

	v3Router, v3Err := callAddressBySignature(ctx, client, swapper.ContractAddress, "swapRouterV3()")
	uniRouter, uniErr := callAddressBySignature(ctx, client, swapper.ContractAddress, "universalRouter()")

	if v3Err == nil {
		status.SwapRouterV3 = v3Router.Hex()
	} else {
		status.Reasons = append(status.Reasons, "SWAP_ROUTER_V3_READ_FAILED")
	}

	if uniErr == nil {
		status.UniversalV4 = uniRouter.Hex()
	} else {
		status.Reasons = append(status.Reasons, "UNIVERSAL_ROUTER_READ_FAILED")
	}

	v3Ready := v3Err == nil && v3Router != (common.Address{})
	v4Ready := uniErr == nil && uniRouter != (common.Address{})

	if !v3Ready {
		status.Reasons = append(status.Reasons, "SWAP_ROUTER_V3_NOT_SET")
	}
	if !v4Ready {
		status.Reasons = append(status.Reasons, "UNIVERSAL_ROUTER_NOT_SET")
	}

	status.Executable = v3Ready || v4Ready
	if !status.Executable {
		status.Reasons = append(status.Reasons, "NO_EXECUTOR_CONFIGURED")
	}

	return status, nil
}

func callAddressBySignature(ctx context.Context, client *blockchain.EVMClient, contractAddress, signature string) (common.Address, error) {
	selector := crypto.Keccak256([]byte(signature))[:4]
	out, err := client.CallView(ctx, contractAddress, selector)
	if err != nil {
		return common.Address{}, err
	}
	if len(out) < 32 {
		return common.Address{}, fmt.Errorf("invalid %s response", signature)
	}
	return common.BytesToAddress(out[len(out)-20:]), nil
}

// decodeSafeQuoteResult decodes (bool, uint256, string) from a SAFE quote call.
func decodeSafeQuoteResult(data []byte) (bool, *big.Int, string, error) {
	boolType, _ := abi.NewType("bool", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	stringType, _ := abi.NewType("string", "", nil)
	args := abi.Arguments{
		{Type: boolType},
		{Type: uint256Type},
		{Type: stringType},
	}
	results, err := args.Unpack(data)
	if err != nil {
		return false, nil, "", err
	}
	return results[0].(bool), results[1].(*big.Int), results[2].(string), nil
}
