package usecases

import (
	"context"
	"encoding/hex"
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
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/pkg/utils"
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

	// Save payment
	// Wrap critical DB operations in transaction (payment only).
	if err = u.uow.Do(ctx, func(txCtx context.Context) error {
		// Save payment
		if err := u.paymentRepo.Create(txCtx, payment); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Create initial event as best-effort after payment is committed.
	// This prevents schema drift on payment_events from breaking core payment creation.
	event := &entities.PaymentEvent{
		ID:        utils.GenerateUUIDv7(),
		PaymentID: payment.ID,
		EventType: entities.PaymentEventTypeCreated,
		ChainID:   &sourceChain.ID,
		CreatedAt: time.Now(),
	}
	// Retry loop for event creation to handle potential DB replication lag or race on FK visibility.
	for i := 0; i < 3; i++ {
		if err := u.paymentEventRepo.Create(ctx, event); err != nil {
			if isForeignKeyViolation(err, "payment_events_payment_id_fkey") {
				if i < 2 {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				fmt.Printf("Info: skip payment event insert due to pending payment FK visibility for %s after retries\n", payment.ID)
			} else {
				fmt.Printf("Warning: failed to create payment event for payment %s: %v\n", payment.ID, err)
			}
		}
		break
	}

	// Build transaction data using metadata from DB
	signatureData, sigErr := u.buildTransactionData(payment, contract)
	if sigErr != nil {
		return nil, sigErr
	}

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

		createPaymentData := u.buildEvmPaymentHex(payment, destChainID, minDestAmount)
		txValueHex := "0x0"
		sourceCAIP2 := ""
		if payment.SourceChain != nil {
			sourceCAIP2 = payment.SourceChain.GetCAIP2ID()
		} else if srcChain, err := u.chainRepo.GetByID(context.Background(), payment.SourceChainID); err == nil && srcChain != nil {
			sourceCAIP2 = srcChain.GetCAIP2ID()
		}
		if sourceCAIP2 != "" && sourceCAIP2 != destChainID {
			amount := new(big.Int)
			if _, ok := amount.SetString(payment.SourceAmount, 10); ok {
				feeWei, err := u.getBridgeFeeQuote(context.Background(), sourceCAIP2, destChainID, payment.SourceTokenAddress, payment.DestTokenAddress, amount, minDestAmount)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve bridge fee quote for %s -> %s: %w", sourceCAIP2, destChainID, err)
				}
				feeWithMargin := new(big.Int).Mul(feeWei, bridgeFeeSafetyBps)
				feeWithMargin.Div(feeWithMargin, bpsDenominator)
				if feeWithMargin.Cmp(feeWei) < 0 {
					feeWithMargin = new(big.Int).Set(feeWei)
				}
				txValueHex = "0x" + feeWithMargin.Text(16)
			} else {
				return nil, fmt.Errorf("invalid source amount for bridge fee quote")
			}
		}
		result := map[string]interface{}{
			"to":    contract.ContractAddress,
			"data":  createPaymentData,
			"value": txValueHex,
			"transactions": []map[string]string{
				{
					"kind":  "createPayment",
					"to":    contract.ContractAddress,
					"data":  createPaymentData,
					"value": txValueHex,
				},
			},
		}

		if u.shouldRequireEvmApproval(payment.SourceTokenAddress) {
			vaultAddress := u.resolveVaultAddressForApproval(payment.SourceChainID, contract.ContractAddress)
			if vaultAddress == "" {
				return nil, fmt.Errorf("vault contract address is not configured for source chain")
			}
			approvalAmount, approvalErr := u.calculateOnchainApprovalAmount(payment, contract.ContractAddress)
			if approvalErr != nil {
				return nil, approvalErr
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
			result["transactions"] = []map[string]string{
				approvalTx,
				{
					"kind":  "createPayment",
					"to":    contract.ContractAddress,
					"data":  createPaymentData,
					"value": txValueHex,
				},
			}
		}

		return result, nil
	case "solana":
		return map[string]string{
			"programId": contract.ContractAddress,
			"data":      u.buildSvmPaymentBase58(payment),
		}, nil
	}

	return nil, nil
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

	feeABI, abiErr := u.ResolveABIWithFallback(context.Background(), payment.SourceChainID, entities.ContractTypeGateway)
	if abiErr != nil {
		return "", fmt.Errorf("failed to resolve gateway ABI: %w", abiErr)
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

func (u *PaymentUsecase) buildEvmPaymentHex(payment *entities.Payment, destChainID string, minAmountOut *big.Int) string {
	// function createPayment(bytes destChainId, bytes receiver, address sourceToken, address destToken, uint256 amount)
	// OR createPaymentWithSlippage(..., uint256 minAmountOut)
	// NOTE:
	// - destChainIdBytes MUST be CAIP-2 string bytes (e.g. "eip155:8453"), not UUID raw bytes.
	// - receiverBytes MUST be abi.encode(address) because contract decodes via abi.decode(receiverBytes, (address)).
	stringType, err := newABIType("bytes", "", nil)
	if err != nil {
		return ""
	}
	addressType, err := newABIType("address", "", nil)
	if err != nil {
		return ""
	}
	uintType, err := newABIType("uint256", "", nil)
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

	args := abi.Arguments{
		{Type: stringType},  // bytes destChainIdBytes
		{Type: stringType},  // bytes receiverBytes
		{Type: addressType}, // sourceToken
		{Type: addressType}, // destToken
		{Type: uintType},    // amount
	}

	vals := []interface{}{
		[]byte(destChainID),
		receiverBytes,
		common.HexToAddress(normalizeEvmAddress(payment.SourceTokenAddress)),
		common.HexToAddress(normalizeEvmAddress(payment.DestTokenAddress)),
		amount,
	}

	selector := CreatePaymentSelector
	if minAmountOut != nil && minAmountOut.Sign() > 0 {
		args = append(args, abi.Argument{Type: uintType}) // minAmountOut
		vals = append(vals, minAmountOut)
		selector = CreatePaymentWithSlippageSelector
	}

	packedArgs, err := packABIArgs(args, vals...)
	if err != nil {
		return ""
	}
	selectorBytes, _ := hex.DecodeString(strings.TrimPrefix(selector, "0x"))
	return "0x" + hex.EncodeToString(append(selectorBytes, packedArgs...))
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
	order := make([]uint8, 0, 3)
	add := func(bridgeType uint8) {
		if bridgeType > 2 {
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

	// ABI encode quotePaymentFee(string,uint8,(bytes32,address,address,address,uint256,string,uint256))
	messageTupleType, err := newABIType("tuple", "", []abi.ArgumentMarshaling{
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
	stringType, err := newABIType("string", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build ABI string type: %w", err)
	}
	uint8Type, err := newABIType("uint8", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build ABI uint8 type: %w", err)
	}
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
		MinAmountOut: minAmountOut,
	}

	packedArgs, err := packABIArgs(args, destCAIP2, bridgeType, msgStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to pack quotePaymentFee args: %w", err)
	}

	// 6. Call Contract
	// Try quotePaymentFeeSafe first (non-reverting)
	// (bool success, uint256 fee, string reason)
	safeSig := []byte("quotePaymentFeeSafe(string,uint8,(bytes32,address,address,address,uint256,string,uint256))")
	safeMethodID := crypto.Keccak256(safeSig)[:4]
	safeCalldata := append(safeMethodID, packedArgs...)

	safeResult, safeErr := client.CallView(ctx, routerAddress, safeCalldata)
	if safeErr == nil && len(safeResult) >= 96 {
		ok, fee, reason, err := decodeSafeQuoteResult(safeResult)
		if err == nil {
			if !ok {
				return nil, fmt.Errorf("quote failed: %s", reason)
			}
			return fee, nil
		}
		// If decoding fails, fall through to legacy method
	}

	// Fallback: use reverting quotePaymentFee for older routers
	methodSig := []byte("quotePaymentFee(string,uint8,(bytes32,address,address,address,uint256,string,uint256))")
	methodID := crypto.Keccak256(methodSig)[:4]
	calldata := append(methodID, packedArgs...)

	result, err := client.CallView(ctx, routerAddress, calldata)
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
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	// 7. Decode Result (uint256)
	if len(result) == 0 {
		return nil, fmt.Errorf("empty result from quotePaymentFee")
	}
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
		return "LayerZero"
	default:
		return "Hyperbridge"
	}
}

func bridgeNameToType(bridgeName string) uint8 {
	switch strings.ToUpper(strings.TrimSpace(bridgeName)) {
	case "CCIP":
		return 1
	case "LAYERZERO":
		return 2
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

// CheckRouteSupport checks if a route exists for a token pair on-chain
func (u *PaymentUsecase) CheckRouteSupport(
	ctx context.Context,
	chainID uuid.UUID,
	tokenIn, tokenOut string,
) (bool, bool, []string, error) {
	if tokenIn == tokenOut {
		return true, true, []string{tokenIn}, nil
	}

	chain, err := u.chainRepo.GetByID(ctx, chainID)
	if err != nil {
		return false, false, nil, err
	}

	swapper, err := u.contractRepo.GetActiveContract(ctx, chain.ID, entities.ContractTypeTokenSwapper)
	if err != nil {
		return false, false, nil, fmt.Errorf("active swapper not found")
	}

	client, err := u.clientFactory.GetEVMClient(chain.RPCURL)
	if err != nil {
		return false, false, nil, err
	}

	swapperABI, err := u.ResolveABIWithFallback(ctx, chain.ID, entities.ContractTypeTokenSwapper)
	if err != nil {
		return false, false, nil, err
	}

	findRouteCall, err := swapperABI.Pack("findRoute", common.HexToAddress(tokenIn), common.HexToAddress(tokenOut))
	if err != nil {
		return false, false, nil, err
	}

	out, err := client.CallView(ctx, swapper.ContractAddress, findRouteCall)
	if err != nil {
		return false, false, nil, err
	}

	// Unpack: (bool exists, bool isDirect, address[] path)
	results, err := swapperABI.Unpack("findRoute", out)
	if err != nil || len(results) < 3 {
		return false, false, nil, fmt.Errorf("failed to unpack findRoute")
	}

	exists, ok1 := results[0].(bool)
	isDirect, ok2 := results[1].(bool)
	pathAddrs, ok3 := results[2].([]common.Address)

	if !ok1 || !ok2 || !ok3 {
		return false, false, nil, fmt.Errorf("failed to type cast findRoute results")
	}

	path := make([]string, len(pathAddrs))
	for i, addr := range pathAddrs {
		path[i] = addr.Hex()
	}

	return exists, isDirect, path, nil
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
