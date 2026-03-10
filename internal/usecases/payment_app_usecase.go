package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/pkg/utils"
)

type PaymentAppUsecase struct {
	paymentUsecase *PaymentUsecase
	userRepo       repositories.UserRepository
	walletRepo     repositories.WalletRepository
	chainRepo      repositories.ChainRepository
	chainResolver  *ChainResolver
}

var predictEscrowStealthAddressFn = tryPredictEscrowStealthAddress

func NewPaymentAppUsecase(
	paymentUsecase *PaymentUsecase,
	userRepo repositories.UserRepository,
	walletRepo repositories.WalletRepository,
	chainRepo repositories.ChainRepository,
) *PaymentAppUsecase {
	return &PaymentAppUsecase{
		paymentUsecase: paymentUsecase,
		userRepo:       userRepo,
		walletRepo:     walletRepo,
		chainRepo:      chainRepo,
		chainResolver:  NewChainResolver(chainRepo),
	}
}

func (u *PaymentAppUsecase) CreatePaymentApp(ctx context.Context, input *entities.CreatePaymentAppInput) (*entities.CreatePaymentResponse, error) {
	mode := normalizePaymentMode(input.Mode)
	if _, err := normalizeBridgeOption(input.BridgeOption); err != nil {
		return nil, fmt.Errorf("invalid bridge option: %w", err)
	}

	senderAddress := strings.TrimSpace(input.SenderWalletAddress)

	sourceChainID, sourceCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.SourceChainID)
	if err != nil {
		return nil, fmt.Errorf("invalid source chain: %w", err)
	}
	_, destCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.DestChainID)
	if err != nil {
		return nil, fmt.Errorf("invalid destination chain: %w", err)
	}
	if mode == PaymentModePrivacy {
		intentID, stealthReceiver, err := u.preparePrivacyRoutingWithDB(ctx, sourceChainID, input)
		if err != nil {
			return nil, err
		}
		input.PrivacyIntentID = &intentID
		input.PrivacyStealthReceiver = &stealthReceiver
	}
	if err := validatePrivacyFields(mode, input.PrivacyIntentID, input.PrivacyStealthReceiver); err != nil {
		return nil, err
	}

	// 2. Resolve User logic
	var userID uuid.UUID

	wallet, err := u.walletRepo.GetByAddress(ctx, sourceChainID, senderAddress)
	if err == nil && wallet != nil && wallet.UserID != nil {
		// Case A: Wallet exists -> Use existing User
		userID = *wallet.UserID
	} else {
		// Case B: Wallet not found (or no user attached) -> Create new User + Wallet
		// Note: Ideally we should check if address exists on OTHER chains to link to same user (EVM),
		// but `GetByAddress` is chain-scoped. For MVP, we create new user if not found on THIS chain.
		// Improvement: Add `walletRepo.FindByAddressAnyChain(address)` later.

		// Create User
		newUserID := utils.GenerateUUIDv7()
		email := fmt.Sprintf("%s_%s@app.paymentkita.local", walletPrefix(senderAddress), newUserID.String()[:8])

		newUser := &entities.User{
			ID:        newUserID,
			Email:     email,
			Name:      "App User " + walletNamePrefix(senderAddress),
			Role:      entities.UserRoleUser,
			KYCStatus: entities.KYCNotStarted,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		// We set a dummy password hash since they auth via wallet/api-key context (not password login)
		newUser.PasswordHash = "WALLET_AUTH_NO_PASSWORD"

		if err := u.userRepo.Create(ctx, newUser); err != nil {
			return nil, fmt.Errorf("failed to auto-create user: %w", err)
		}
		userID = newUser.ID

		// Create Wallet
		newWallet := &entities.Wallet{
			ID:        utils.GenerateUUIDv7(),
			UserID:    &userID,
			ChainID:   sourceChainID,
			Address:   senderAddress,
			Type:      "EOA",
			IsPrimary: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := u.walletRepo.Create(ctx, newWallet); err != nil {
			// Basic rollback cleanup could be here, but for now just fail
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
	}

	// 3. Delegated Payment Creation
	// Map AppInput to PaymentInput
	paymentInput := &entities.CreatePaymentInput{
		SourceChainID:      sourceCAIP2,
		DestChainID:        destCAIP2,
		SourceTokenAddress: input.SourceTokenAddress,
		DestTokenAddress:   input.DestTokenAddress,
		Amount:             input.Amount,
		Decimals:           input.Decimals,
		ReceiverAddress:    input.ReceiverAddress,
		// ReceiverMerchantID is empty for App payments (any receiver allowed)
		Mode:                   input.Mode,
		BridgeOption:           input.BridgeOption,
		BridgeTokenSource:      input.BridgeTokenSource,
		MinBridgeAmountOut:     input.MinBridgeAmountOut,
		MinDestAmountOut:       input.MinDestAmountOut,
		PrivacyIntentID:        input.PrivacyIntentID,
		PrivacyStealthReceiver: input.PrivacyStealthReceiver,
	}

	return u.paymentUsecase.CreatePayment(ctx, userID, paymentInput)
}

func (u *PaymentAppUsecase) preparePrivacyRoutingWithDB(
	ctx context.Context,
	sourceChainID uuid.UUID,
	input *entities.CreatePaymentAppInput,
) (intentID string, stealthReceiver string, err error) {
	if u.paymentUsecase == nil || u.paymentUsecase.contractRepo == nil || u.chainRepo == nil {
		// Compatibility fallback for tests/wiring that don't pass repository dependencies.
		return preparePrivacyRouting(input)
	}

	receiverRaw := strings.TrimSpace(input.ReceiverAddress)
	if receiverRaw == "" {
		return "", "", fmt.Errorf("receiverAddress is required when mode=privacy")
	}
	if !common.IsHexAddress(receiverRaw) {
		return "", "", fmt.Errorf("receiverAddress must be valid EVM address when mode=privacy")
	}
	finalReceiver := common.HexToAddress(normalizeEvmAddress(receiverRaw))

	intentID = ""
	if input.PrivacyIntentID != nil {
		intentID = strings.TrimSpace(*input.PrivacyIntentID)
	}
	if intentID == "" {
		intentID = utils.GenerateUUIDv7().String()
	}

	factoryContract, err := u.paymentUsecase.contractRepo.GetActiveContract(ctx, sourceChainID, entities.ContractTypeStealthEscrowFactory)
	if err != nil || factoryContract == nil || !common.IsHexAddress(factoryContract.ContractAddress) {
		return "", "", fmt.Errorf("active stealth escrow factory is not configured on source chain")
	}
	privacyModuleContract, err := u.paymentUsecase.contractRepo.GetActiveContract(ctx, sourceChainID, entities.ContractTypePrivacyModule)
	if err != nil || privacyModuleContract == nil || !common.IsHexAddress(privacyModuleContract.ContractAddress) {
		return "", "", fmt.Errorf("active privacy module is not configured on source chain")
	}

	sourceChain, err := u.chainRepo.GetByID(ctx, sourceChainID)
	if err != nil || sourceChain == nil {
		return "", "", fmt.Errorf("failed to resolve source chain for privacy escrow prediction")
	}
	rpcURL := resolveChainRPCURL(sourceChain)
	if rpcURL == "" {
		return "", "", fmt.Errorf("source chain rpc url is not configured for privacy escrow prediction")
	}

	expectedStealth, err := predictEscrowFromFactoryRPC(
		ctx,
		rpcURL,
		common.HexToAddress(normalizeEvmAddress(factoryContract.ContractAddress)),
		parsePrivacyIntentID(intentID),
		finalReceiver,
		common.HexToAddress(normalizeEvmAddress(privacyModuleContract.ContractAddress)),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to derive escrow stealth address from factory: %w", err)
	}
	if expectedStealth == finalReceiver {
		return "", "", fmt.Errorf("privacyStealthReceiver must differ from receiverAddress")
	}

	if input.PrivacyStealthReceiver != nil {
		stealthRaw := strings.TrimSpace(*input.PrivacyStealthReceiver)
		if stealthRaw != "" {
			if !common.IsHexAddress(stealthRaw) {
				return "", "", fmt.Errorf("privacyStealthReceiver must be valid EVM address when mode=privacy")
			}
			stealth := common.HexToAddress(normalizeEvmAddress(stealthRaw))
			if stealth == finalReceiver {
				return "", "", fmt.Errorf("privacyStealthReceiver must differ from receiverAddress")
			}
			if stealth != expectedStealth {
				return "", "", fmt.Errorf("privacyStealthReceiver must match factory predicted escrow address: %s", expectedStealth.Hex())
			}
			return intentID, stealth.Hex(), nil
		}
	}

	return intentID, expectedStealth.Hex(), nil
}

func walletPrefix(addr string) string {
	if len(addr) >= 8 {
		return addr[:8]
	}
	if addr == "" {
		return "wallet"
	}
	return addr
}

func walletNamePrefix(addr string) string {
	if len(addr) >= 6 {
		return addr[:6]
	}
	if addr == "" {
		return "wallet"
	}
	return addr
}

func preparePrivacyRouting(input *entities.CreatePaymentAppInput) (intentID string, stealthReceiver string, err error) {
	receiverRaw := strings.TrimSpace(input.ReceiverAddress)
	if receiverRaw == "" {
		return "", "", fmt.Errorf("receiverAddress is required when mode=privacy")
	}
	if !common.IsHexAddress(receiverRaw) {
		return "", "", fmt.Errorf("receiverAddress must be valid EVM address when mode=privacy")
	}
	finalReceiver := common.HexToAddress(normalizeEvmAddress(receiverRaw))

	intentID = ""
	if input.PrivacyIntentID != nil {
		intentID = strings.TrimSpace(*input.PrivacyIntentID)
	}
	if intentID == "" {
		intentID = utils.GenerateUUIDv7().String()
	}

	escrowStealth, predictedFromFactory, predictErr := predictEscrowStealthAddressFn(intentID, finalReceiver)
	if predictErr != nil {
		return "", "", predictErr
	}
	if !predictedFromFactory {
		escrowStealth = prepareEscrowStealthAddress(input, intentID)
	}

	if input.PrivacyStealthReceiver != nil {
		stealthRaw := strings.TrimSpace(*input.PrivacyStealthReceiver)
		if stealthRaw != "" {
			if !common.IsHexAddress(stealthRaw) {
				return "", "", fmt.Errorf("privacyStealthReceiver must be valid EVM address when mode=privacy")
			}
			stealth := common.HexToAddress(normalizeEvmAddress(stealthRaw))
			if stealth == finalReceiver {
				return "", "", fmt.Errorf("privacyStealthReceiver must differ from receiverAddress")
			}
			if predictedFromFactory && stealth != escrowStealth {
				return "", "", fmt.Errorf("privacyStealthReceiver must match factory predicted escrow address: %s", escrowStealth.Hex())
			}
			return intentID, stealth.Hex(), nil
		}
	}

	if escrowStealth == finalReceiver {
		return "", "", fmt.Errorf("privacyStealthReceiver must differ from receiverAddress")
	}
	if escrowStealth == (common.Address{}) {
		return "", "", fmt.Errorf("failed to derive escrow stealth address")
	}
	return intentID, escrowStealth.Hex(), nil
}

func tryPredictEscrowStealthAddress(intentID string, owner common.Address) (common.Address, bool, error) {
	_ = intentID
	_ = owner
	// Runtime path already resolves from DB via preparePrivacyRoutingWithDB.
	// Keep fallback deterministic and DB-agnostic for legacy tests/wiring.
	return common.Address{}, false, nil
}

func predictEscrowFromFactoryRPC(
	ctx context.Context,
	rpcURL string,
	factory common.Address,
	salt [32]byte,
	owner common.Address,
	forwarder common.Address,
) (common.Address, error) {
	const predictEscrowABI = `[{"inputs":[{"internalType":"bytes32","name":"salt","type":"bytes32"},{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"forwarder","type":"address"}],"name":"predictEscrow","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(predictEscrowABI))
	if err != nil {
		return common.Address{}, fmt.Errorf("parse factory abi: %w", err)
	}
	calldata, err := parsedABI.Pack("predictEscrow", salt, owner, forwarder)
	if err != nil {
		return common.Address{}, fmt.Errorf("pack predictEscrow calldata: %w", err)
	}

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return common.Address{}, fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	msg := ethereum.CallMsg{To: &factory, Data: calldata}
	out, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("call predictEscrow: %w", err)
	}
	values, err := parsedABI.Unpack("predictEscrow", out)
	if err != nil {
		return common.Address{}, fmt.Errorf("decode predictEscrow return: %w", err)
	}
	if len(values) != 1 {
		return common.Address{}, fmt.Errorf("invalid predictEscrow return length")
	}
	addr, ok := values[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("invalid predictEscrow return type")
	}
	return addr, nil
}

func resolveChainRPCURL(chain *entities.Chain) string {
	if chain == nil {
		return ""
	}
	if rpcURL := strings.TrimSpace(chain.RPCURL); rpcURL != "" {
		return rpcURL
	}
	fallback := ""
	for _, rpc := range chain.RPCs {
		url := strings.TrimSpace(rpc.URL)
		if url == "" {
			continue
		}
		if rpc.IsActive {
			return url
		}
		if fallback == "" {
			fallback = url
		}
	}
	return fallback
}

func prepareEscrowStealthAddress(input *entities.CreatePaymentAppInput, intentID string) common.Address {
	seed := strings.Join([]string{
		"privacy-escrow-v2",
		strings.TrimSpace(input.SenderWalletAddress),
		strings.TrimSpace(input.ReceiverAddress),
		strings.TrimSpace(input.SourceChainID),
		strings.TrimSpace(input.DestChainID),
		strings.TrimSpace(input.SourceTokenAddress),
		strings.TrimSpace(input.DestTokenAddress),
		strings.TrimSpace(input.Amount),
		intentID,
	}, "|")
	salt := crypto.Keccak256Hash([]byte(seed))
	placeholder := crypto.Keccak256Hash([]byte("payment-kita-escrow-placeholder"), salt.Bytes())
	return common.BytesToAddress(placeholder.Bytes()[12:])
}
