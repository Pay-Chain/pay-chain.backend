package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/domain/repositories"
)

// ABIResolverMixin provides common ABI resolution logic
type ABIResolverMixin struct {
	contractRepo repositories.SmartContractRepository
	abiCache     sync.Map // map[string]*abi.ABI (key: chainID+contractType)
}

func NewABIResolverMixin(contractRepo repositories.SmartContractRepository) *ABIResolverMixin {
	return &ABIResolverMixin{
		contractRepo: contractRepo,
	}
}

// ResolveABI fetches the ABI for a contract type on a chain from the DB,
// parses it, and caches the result for subsequent calls.
func (u *ABIResolverMixin) ResolveABI(
	ctx context.Context, chainID uuid.UUID, contractType entities.SmartContractType,
) (*abi.ABI, string, error) {
	cacheKey := fmt.Sprintf("%s:%s", chainID.String(), contractType)

	// Check cache
	if cached, ok := u.abiCache.Load(cacheKey); ok {
		if parsedABI, ok := cached.(*abi.ABI); ok {
			contract, err := u.contractRepo.GetActiveContract(ctx, chainID, contractType)
			if err != nil {
				return nil, "", fmt.Errorf("contract %s not found: %w", contractType, err)
			}
			if contract == nil {
				return nil, "", fmt.Errorf("contract %s not found", contractType)
			}
			return parsedABI, contract.ContractAddress, nil
		}
	}

	contract, err := u.contractRepo.GetActiveContract(ctx, chainID, contractType)
	if err != nil {
		return nil, "", fmt.Errorf("contract %s not found: %w", contractType, err)
	}
	if contract == nil {
		return nil, "", fmt.Errorf("contract %s not found", contractType)
	}

	if contract.ABI == nil {
		return nil, contract.ContractAddress, nil // No ABI in DB, return empty/nil to trigger fallback
	}

	rawABI, err := json.Marshal(contract.ABI)
	if err != nil {
		return nil, contract.ContractAddress, fmt.Errorf("failed to marshal ABI JSON: %w", err)
	}

	parsed, err := abi.JSON(bytes.NewReader(rawABI))
	if err != nil {
		// Attempt to correct common format issues or fallback
		return nil, contract.ContractAddress, fmt.Errorf("failed to parse ABI for %s: %w", contractType, err)
	}

	// Update cache
	u.abiCache.Store(cacheKey, &parsed)

	return &parsed, contract.ContractAddress, nil
}

// ResolveABIWithFallback attempts to resolve ABI from DB, defaulting to hardcoded fallbacks if not found in DB
func (u *ABIResolverMixin) ResolveABIWithFallback(ctx context.Context, chainID uuid.UUID, contractType entities.SmartContractType) (abi.ABI, error) {
	// RECEIVER_STARGATE execution in admin flows uses stable admin ABI and
	// explicit contract address from payload; avoid hard dependency on DB ABI row.
	if contractType == entities.ContractTypeReceiverStargate {
		return FallbackStargateReceiverAdminABI, nil
	}

	parsed, _, err := u.ResolveABI(ctx, chainID, contractType)
	if err == nil && parsed != nil {
		// Validate that the ABI actually contains the expected admin methods
		isValid := false
		switch contractType {
		case entities.ContractTypeAdapterHyperbridge:
			_, isValid = parsed.Methods["setStateMachineId"]
			if !isValid {
				fmt.Printf("[ResolveABI] ABI for %s has %d methods but missing 'setStateMachineId'. Using fallback.\n", contractType, len(parsed.Methods))
			}
		case entities.ContractTypeAdapterCCIP:
			_, hasSetChainSelector := parsed.Methods["setChainSelector"]
			_, hasSetChainConfig := parsed.Methods["setChainConfig"]
			isValid = hasSetChainSelector || hasSetChainConfig
			if !isValid {
				fmt.Printf("[ResolveABI] ABI for %s has %d methods but missing 'setChainSelector'/'setChainConfig'. Using fallback.\n", contractType, len(parsed.Methods))
			}
		case entities.ContractTypeAdapterStargate:
			_, isValid = parsed.Methods["setRoute"]
			if !isValid {
				fmt.Printf("[ResolveABI] ABI for %s has %d methods but missing 'setRoute'. Using fallback.\n", contractType, len(parsed.Methods))
			}
		case entities.ContractTypeAdapterHBTokenSender:
			_, hasSetStateMachine := parsed.Methods["setStateMachineId"]
			_, hasSetSettlementExecutor := parsed.Methods["setRouteSettlementExecutor"]
			isValid = hasSetStateMachine && hasSetSettlementExecutor
			if !isValid {
				fmt.Printf("[ResolveABI] ABI for %s has %d methods but missing token-gateway admin methods. Using fallback.\n", contractType, len(parsed.Methods))
			}
		default:
			// For others (or if we don't need strict validation), checks length
			isValid = len(parsed.Methods) > 0
		}

		if isValid {
			fmt.Printf("[ResolveABI] Found validated ABI for %s on %s. Methods: %d\n", contractType, chainID, len(parsed.Methods))
			return *parsed, nil
		}
	} else if err != nil {
		fmt.Printf("[ResolveABI] Failed to resolve from DB for %s on %s: %v. Falling back.\n", contractType, chainID, err)
	}

	// Fallback logic
	switch contractType {
	case entities.ContractTypeGateway:
		return FallbackPaymentKitaGatewayABI, nil
	case entities.ContractTypeVault:
		return FallbackPaymentKitaVaultAdminABI, nil
	case entities.ContractTypeRouter:
		return FallbackPaymentKitaRouterAdminABI, nil
	case entities.ContractTypeAdapterHyperbridge:
		return FallbackHyperbridgeSenderAdminABI, nil
	case entities.ContractTypeAdapterCCIP:
		return FallbackCCIPSenderAdminABI, nil
	case entities.ContractTypeAdapterStargate:
		return FallbackStargateSenderAdminABI, nil
	case entities.ContractTypeAdapterHBTokenSender:
		return FallbackHyperbridgeTokenGatewaySenderAdminABI, nil
	case entities.ContractTypeReceiverStargate:
		return FallbackStargateReceiverAdminABI, nil
	}
	if err != nil {
		return abi.ABI{}, err
	}
	// Valid contract but no ABI in DB, and no fallback for this type?
	// or ResolveABI returned nil parsed but no error
	return abi.ABI{}, fmt.Errorf("no ABI found for %s", contractType)
}
