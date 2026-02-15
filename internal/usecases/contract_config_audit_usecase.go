package usecases

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/pkg/utils"
)

type ContractConfigCheckItem struct {
	Code     string `json:"code"`
	Status   string `json:"status"` // OK, WARN, ERROR
	Message  string `json:"message"`
	Contract string `json:"contract,omitempty"`
}

type ContractConfigContractReport struct {
	ID                uuid.UUID                  `json:"id"`
	Name              string                     `json:"name"`
	Type              entities.SmartContractType `json:"type"`
	Address           string                     `json:"address"`
	ChainID           string                     `json:"chainId"`
	IsActive          bool                       `json:"isActive"`
	FunctionNames     []string                   `json:"functionNames"`
	RequiredFunctions []string                   `json:"requiredFunctions"`
	MissingFunctions  []string                   `json:"missingFunctions"`
	GeneratedFields   []string                   `json:"generatedFields"`
	Checks            []ContractConfigCheckItem  `json:"checks"`
}

type ContractConfigAuditResult struct {
	SourceChainID string                         `json:"sourceChainId"`
	DestChainID   string                         `json:"destChainId,omitempty"`
	OverallStatus string                         `json:"overallStatus"` // OK, WARN, ERROR
	Summary       map[string]int                 `json:"summary"`
	GlobalChecks  []ContractConfigCheckItem      `json:"globalChecks"`
	Contracts     []ContractConfigContractReport `json:"contracts"`
}

type ContractDestinationAudit struct {
	DestChainID   string                    `json:"destChainId"`
	DestChainName string                    `json:"destChainName"`
	OverallStatus string                    `json:"overallStatus"`
	Summary       map[string]int            `json:"summary"`
	Checks        []ContractConfigCheckItem `json:"checks"`
}

type ContractDetailAuditResult struct {
	Contract          ContractConfigContractReport `json:"contract"`
	SourceChainID     string                       `json:"sourceChainId"`
	SourceChainName   string                       `json:"sourceChainName"`
	OverallStatus     string                       `json:"overallStatus"`
	Summary           map[string]int               `json:"summary"`
	GlobalChecks      []ContractConfigCheckItem    `json:"globalChecks"`
	DestinationAudits []ContractDestinationAudit   `json:"destinationAudits"`
}

type ContractConfigAuditUsecase struct {
	chainRepo     repositories.ChainRepository
	contractRepo  repositories.SmartContractRepository
	clientFactory *blockchain.ClientFactory
	chainResolver *ChainResolver
}

func NewContractConfigAuditUsecase(
	chainRepo repositories.ChainRepository,
	contractRepo repositories.SmartContractRepository,
	clientFactory *blockchain.ClientFactory,
) *ContractConfigAuditUsecase {
	return &ContractConfigAuditUsecase{
		chainRepo:     chainRepo,
		contractRepo:  contractRepo,
		clientFactory: clientFactory,
		chainResolver: NewChainResolver(chainRepo),
	}
}

func (u *ContractConfigAuditUsecase) Check(ctx context.Context, sourceChainInput, destChainInput string) (*ContractConfigAuditResult, error) {
	sourceChainUUID, sourceCAIP2, err := u.chainResolver.ResolveFromAny(ctx, strings.TrimSpace(sourceChainInput))
	if err != nil {
		return nil, fmt.Errorf("invalid sourceChainId: %w", err)
	}
	sourceChain, err := u.chainRepo.GetByID(ctx, sourceChainUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load source chain: %w", err)
	}

	result := &ContractConfigAuditResult{
		SourceChainID: sourceCAIP2,
		OverallStatus: "OK",
		Summary: map[string]int{
			"ok":    0,
			"warn":  0,
			"error": 0,
		},
	}

	var destCAIP2 string
	if strings.TrimSpace(destChainInput) != "" {
		_, resolvedDest, resolveErr := u.chainResolver.ResolveFromAny(ctx, strings.TrimSpace(destChainInput))
		if resolveErr != nil {
			result.GlobalChecks = append(result.GlobalChecks, ContractConfigCheckItem{
				Code:    "DEST_CHAIN_INVALID",
				Status:  "ERROR",
				Message: "destination chain is invalid",
			})
			result.Summary["error"]++
		} else {
			destCAIP2 = resolvedDest
			result.DestChainID = destCAIP2
		}
	}

	contracts, _, err := u.contractRepo.GetFiltered(
		ctx,
		&sourceChainUUID,
		entities.SmartContractType(""),
		utils.PaginationParams{Page: 1, Limit: 0},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list contracts: %w", err)
	}

	activeContracts := make([]*entities.SmartContract, 0, len(contracts))
	for _, c := range contracts {
		if c != nil && c.IsActive {
			activeContracts = append(activeContracts, c)
		}
	}

	if len(activeContracts) == 0 {
		result.GlobalChecks = append(result.GlobalChecks, ContractConfigCheckItem{
			Code:    "NO_ACTIVE_CONTRACTS",
			Status:  "WARN",
			Message: "no active smart contracts found on source chain",
		})
		result.Summary["warn"]++
	}

	for _, contract := range activeContracts {
		report := u.buildContractReport(contract)
		mergeSummary(result.Summary, report.Checks)
		result.Contracts = append(result.Contracts, report)
	}

	if sourceChain.Type == entities.ChainTypeEVM && destCAIP2 != "" {
		onchainChecks := u.runEVMOnchainChecks(ctx, sourceChain, activeContracts, destCAIP2)
		result.GlobalChecks = append(result.GlobalChecks, onchainChecks...)
		mergeSummary(result.Summary, onchainChecks)
	}

	result.OverallStatus = deriveOverallStatus(result.Summary)
	return result, nil
}

func (u *ContractConfigAuditUsecase) CheckByContractID(ctx context.Context, contractID uuid.UUID) (*ContractDetailAuditResult, error) {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	if contract == nil {
		return nil, fmt.Errorf("contract not found")
	}

	sourceChain, err := u.chainRepo.GetByID(ctx, contract.ChainUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load source chain: %w", err)
	}
	sourceCAIP2 := sourceChain.GetCAIP2ID()

	contractReport := u.buildContractReport(contract)
	result := &ContractDetailAuditResult{
		Contract:        contractReport,
		SourceChainID:   sourceCAIP2,
		SourceChainName: sourceChain.Name,
		Summary: map[string]int{
			"ok":    0,
			"warn":  0,
			"error": 0,
		},
	}
	mergeSummary(result.Summary, contractReport.Checks)

	chains, err := u.chainRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list chains: %w", err)
	}

	var activeContracts []*entities.SmartContract
	if sourceContracts, _, listErr := u.contractRepo.GetFiltered(ctx, &contract.ChainUUID, entities.SmartContractType(""), utils.PaginationParams{Page: 1, Limit: 0}); listErr == nil {
		for _, c := range sourceContracts {
			if c != nil && c.IsActive {
				activeContracts = append(activeContracts, c)
			}
		}
	}

	for _, ch := range chains {
		if ch == nil || !ch.IsActive || ch.ID == sourceChain.ID {
			continue
		}
		destCAIP2 := ch.GetCAIP2ID()
		destAudit := ContractDestinationAudit{
			DestChainID:   destCAIP2,
			DestChainName: ch.Name,
			Summary: map[string]int{
				"ok":    0,
				"warn":  0,
				"error": 0,
			},
		}

		if sourceChain.Type != entities.ChainTypeEVM {
			destAudit.Checks = append(destAudit.Checks, ContractConfigCheckItem{
				Code:    "ONCHAIN_AUDIT_SKIPPED",
				Status:  "WARN",
				Message: "on-chain route audit currently supports EVM source chain only",
			})
		} else {
			destAudit.Checks = append(destAudit.Checks, u.runEVMOnchainChecks(ctx, sourceChain, activeContracts, destCAIP2)...)
		}
		mergeSummary(destAudit.Summary, destAudit.Checks)
		destAudit.OverallStatus = deriveOverallStatus(destAudit.Summary)
		result.DestinationAudits = append(result.DestinationAudits, destAudit)
		mergeSummary(result.Summary, destAudit.Checks)
	}

	sort.Slice(result.DestinationAudits, func(i, j int) bool {
		return result.DestinationAudits[i].DestChainID < result.DestinationAudits[j].DestChainID
	})

	result.OverallStatus = deriveOverallStatus(result.Summary)
	return result, nil
}

func (u *ContractConfigAuditUsecase) buildContractReport(contract *entities.SmartContract) ContractConfigContractReport {
	functionNames := extractFunctionNames(contract.ABI)
	required := requiredFunctions(contract.Type)
	missing := make([]string, 0)
	existing := make(map[string]struct{}, len(functionNames))
	for _, fn := range functionNames {
		existing[strings.ToLower(fn)] = struct{}{}
	}
	for _, req := range required {
		if _, ok := existing[strings.ToLower(req)]; !ok {
			missing = append(missing, req)
		}
	}

	checks := make([]ContractConfigCheckItem, 0)
	if contract.ContractAddress == "" {
		checks = append(checks, ContractConfigCheckItem{
			Code:     "ADDRESS_EMPTY",
			Status:   "ERROR",
			Message:  "contract address is empty",
			Contract: contract.Name,
		})
	}
	if contract.StartBlock == 0 {
		checks = append(checks, ContractConfigCheckItem{
			Code:     "START_BLOCK_ZERO",
			Status:   "WARN",
			Message:  "start block is 0; indexer catch-up may be expensive",
			Contract: contract.Name,
		})
	}
	if len(functionNames) == 0 {
		checks = append(checks, ContractConfigCheckItem{
			Code:     "ABI_EMPTY_OR_INVALID",
			Status:   "ERROR",
			Message:  "ABI is empty or invalid",
			Contract: contract.Name,
		})
	}
	if len(missing) > 0 {
		checks = append(checks, ContractConfigCheckItem{
			Code:     "ABI_MISSING_REQUIRED_FUNCTIONS",
			Status:   "ERROR",
			Message:  "required functions missing from ABI: " + strings.Join(missing, ", "),
			Contract: contract.Name,
		})
	} else if len(required) > 0 {
		checks = append(checks, ContractConfigCheckItem{
			Code:     "ABI_REQUIRED_FUNCTIONS_OK",
			Status:   "OK",
			Message:  "required ABI functions are complete",
			Contract: contract.Name,
		})
	}

	if contract.Type == entities.ContractTypePool {
		if !contract.Token0Address.Valid || !contract.Token1Address.Valid {
			checks = append(checks, ContractConfigCheckItem{
				Code:     "POOL_TOKEN_PAIR_MISSING",
				Status:   "ERROR",
				Message:  "pool token0/token1 is not configured",
				Contract: contract.Name,
			})
		}
	}

	sort.Strings(functionNames)
	sort.Strings(required)
	sort.Strings(missing)
	return ContractConfigContractReport{
		ID:                contract.ID,
		Name:              contract.Name,
		Type:              contract.Type,
		Address:           contract.ContractAddress,
		ChainID:           contract.BlockchainID,
		IsActive:          contract.IsActive,
		FunctionNames:     functionNames,
		RequiredFunctions: required,
		MissingFunctions:  missing,
		GeneratedFields:   generateFieldsFromFunctions(functionNames),
		Checks:            checks,
	}
}

func (u *ContractConfigAuditUsecase) runEVMOnchainChecks(
	ctx context.Context,
	sourceChain *entities.Chain,
	contracts []*entities.SmartContract,
	destCAIP2 string,
) []ContractConfigCheckItem {
	checks := make([]ContractConfigCheckItem, 0)
	rpcURL := resolveRPCURL(sourceChain)
	if rpcURL == "" {
		return append(checks, ContractConfigCheckItem{
			Code:    "RPC_MISSING",
			Status:  "ERROR",
			Message: "source chain has no active RPC URL",
		})
	}

	client, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return append(checks, ContractConfigCheckItem{
			Code:    "RPC_CONNECT_FAILED",
			Status:  "ERROR",
			Message: "failed to connect source chain RPC",
		})
	}
	defer client.Close()

	gateway := findActiveContractByType(contracts, entities.ContractTypeGateway)
	router := findActiveContractByType(contracts, entities.ContractTypeRouter)
	_ = findActiveContractByType(contracts, entities.ContractTypeAdapterHyperbridge)
	_ = findActiveContractByType(contracts, entities.ContractTypeAdapterCCIP)

	if gateway == nil {
		checks = append(checks, ContractConfigCheckItem{Code: "GATEWAY_MISSING", Status: "ERROR", Message: "active gateway contract is missing"})
	}
	if router == nil {
		checks = append(checks, ContractConfigCheckItem{Code: "ROUTER_MISSING", Status: "ERROR", Message: "active router contract is missing"})
	}
	if gateway == nil || router == nil {
		return checks
	}

	defaultBridgeType, err := callUint8View(ctx, client, gateway.ContractAddress, `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}]`, "defaultBridgeTypes", destCAIP2)
	if err != nil {
		checks = append(checks, ContractConfigCheckItem{
			Code:    "DEFAULT_BRIDGE_READ_FAILED",
			Status:  "ERROR",
			Message: "failed to read default bridge type for destination chain",
		})
		return checks
	}
	checks = append(checks, ContractConfigCheckItem{
		Code:    "DEFAULT_BRIDGE_TYPE",
		Status:  "OK",
		Message: fmt.Sprintf("default bridge type for %s is %d", destCAIP2, defaultBridgeType),
	})

	hasAdapter, err := callBoolView(ctx, client, router.ContractAddress, `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`, "hasAdapter", destCAIP2, defaultBridgeType)
	if err != nil {
		checks = append(checks, ContractConfigCheckItem{
			Code:    "HAS_ADAPTER_READ_FAILED",
			Status:  "ERROR",
			Message: "failed to read adapter availability on router",
		})
		return checks
	}
	if !hasAdapter {
		checks = append(checks, ContractConfigCheckItem{
			Code:    "ADAPTER_NOT_REGISTERED",
			Status:  "ERROR",
			Message: "router has no adapter registered for destination/default bridge type",
		})
	} else {
		checks = append(checks, ContractConfigCheckItem{
			Code:    "ADAPTER_REGISTERED",
			Status:  "OK",
			Message: "router adapter registration exists for destination/default bridge type",
		})
	}

	adapterAddress, err := callAddressView(ctx, client, router.ContractAddress, `[{"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`, "getAdapter", destCAIP2, defaultBridgeType)
	if err == nil {
		if adapterAddress == (common.Address{}) {
			checks = append(checks, ContractConfigCheckItem{
				Code:    "ADAPTER_ADDRESS_ZERO",
				Status:  "ERROR",
				Message: "router adapter address is zero",
			})
		} else {
			checks = append(checks, ContractConfigCheckItem{
				Code:    "ADAPTER_ADDRESS_FOUND",
				Status:  "OK",
				Message: "router adapter address: " + adapterAddress.Hex(),
			})
		}
	}

	if defaultBridgeType == 0 {
		hyperbridgeAdapterAddress := adapterAddress.Hex()
		if hyperbridgeAdapterAddress == "" || hyperbridgeAdapterAddress == "0x0000000000000000000000000000000000000000" {
			checks = append(checks, ContractConfigCheckItem{
				Code:    "HYPERBRIDGE_ADAPTER_ADDRESS_INVALID",
				Status:  "ERROR",
				Message: "hyperbridge adapter address is invalid",
			})
		} else {
			configured, cfgErr := callBoolView(ctx, client, hyperbridgeAdapterAddress, `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"}]`, "isChainConfigured", destCAIP2)
			if cfgErr != nil {
				// Fallback path for older adapter variants: infer from stateMachineIds + destinationContracts
				sm, smErr := callBytesView(ctx, client, hyperbridgeAdapterAddress, `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`, "stateMachineIds", destCAIP2)
				dst, dstErr := callBytesView(ctx, client, hyperbridgeAdapterAddress, `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`, "destinationContracts", destCAIP2)
				if smErr == nil && dstErr == nil {
					if len(sm) > 0 && len(dst) > 0 {
						checks = append(checks, ContractConfigCheckItem{
							Code:    "HYPERBRIDGE_CHAIN_CONFIGURED_FALLBACK",
							Status:  "OK",
							Message: "hyperbridge destination inferred as configured from stateMachineIds/destinationContracts",
						})
					} else {
						checks = append(checks, ContractConfigCheckItem{
							Code:    "HYPERBRIDGE_CHAIN_NOT_CONFIGURED",
							Status:  "ERROR",
							Message: "hyperbridge adapter destination is not configured (state machine/destination contract)",
						})
					}
				} else {
					checks = append(checks, ContractConfigCheckItem{
						Code:    "HYPERBRIDGE_CONFIG_READ_FAILED",
						Status:  "ERROR",
						Message: "failed to read hyperbridge destination config",
					})
				}
			} else if !configured {
				checks = append(checks, ContractConfigCheckItem{
					Code:    "HYPERBRIDGE_CHAIN_NOT_CONFIGURED",
					Status:  "ERROR",
					Message: "hyperbridge adapter destination is not configured (state machine/destination contract)",
				})
			} else {
				checks = append(checks, ContractConfigCheckItem{
					Code:    "HYPERBRIDGE_CHAIN_CONFIGURED",
					Status:  "OK",
					Message: "hyperbridge adapter destination is configured",
				})
			}
		}
	}

	if defaultBridgeType == 1 {
		ccipAdapterAddress := adapterAddress.Hex()
		if ccipAdapterAddress == "" || ccipAdapterAddress == "0x0000000000000000000000000000000000000000" {
			checks = append(checks, ContractConfigCheckItem{
				Code:    "CCIP_ADAPTER_ADDRESS_INVALID",
				Status:  "ERROR",
				Message: "ccip adapter address is invalid",
			})
			return checks
		}
		selector, selectorErr := callUint64View(ctx, client, ccipAdapterAddress, `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"}]`, "chainSelectors", destCAIP2)
		if selectorErr != nil || selector == 0 {
			checks = append(checks, ContractConfigCheckItem{
				Code:    "CCIP_SELECTOR_MISSING",
				Status:  "ERROR",
				Message: "ccip chain selector is missing for destination chain",
			})
		} else {
			checks = append(checks, ContractConfigCheckItem{
				Code:    "CCIP_SELECTOR_CONFIGURED",
				Status:  "OK",
				Message: fmt.Sprintf("ccip chain selector configured: %d", selector),
			})
		}

		destAdapterBytes, bytesErr := callBytesView(ctx, client, ccipAdapterAddress, `[{"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}]`, "destinationAdapters", destCAIP2)
		if bytesErr != nil || len(destAdapterBytes) == 0 {
			checks = append(checks, ContractConfigCheckItem{
				Code:    "CCIP_DEST_ADAPTER_MISSING",
				Status:  "ERROR",
				Message: "ccip destination adapter bytes is missing",
			})
		} else {
			checks = append(checks, ContractConfigCheckItem{
				Code:    "CCIP_DEST_ADAPTER_CONFIGURED",
				Status:  "OK",
				Message: "ccip destination adapter is configured",
			})
		}
	}

	return checks
}

func requiredFunctions(contractType entities.SmartContractType) []string {
	switch contractType {
	case entities.ContractTypeGateway:
		return []string{"createPayment", "createPaymentRequest", "setDefaultBridgeType", "defaultBridgeTypes"}
	case entities.ContractTypeRouter:
		return []string{"registerAdapter", "hasAdapter", "quotePaymentFee", "routePayment"}
	case entities.ContractTypeVault:
		return []string{"depositToken", "pushTokens", "setSpender"}
	case entities.ContractTypeTokenRegistry:
		return []string{"isSupportedToken"}
	case entities.ContractTypeTokenSwapper:
		return []string{"swap"}
	case entities.ContractTypeAdapterCCIP:
		return []string{"quoteFee", "sendMessage", "setChainSelector", "setDestinationAdapter"}
	case entities.ContractTypeAdapterHyperbridge:
		return []string{"quoteFee", "sendMessage", "setStateMachineId", "setDestinationContract"}
	case entities.ContractTypeAdapterLayerZero:
		return []string{"quoteFee", "sendMessage", "setRoute", "setEnforcedOptions"}
	default:
		return []string{}
	}
}

func extractFunctionNames(rawABI interface{}) []string {
	entries, ok := rawABI.([]interface{})
	if !ok {
		return []string{}
	}
	names := make([]string, 0)
	seen := map[string]struct{}{}
	for _, item := range entries {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		entryType, _ := entry["type"].(string)
		if entryType != "function" {
			continue
		}
		name, _ := entry["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func generateFieldsFromFunctions(functionNames []string) []string {
	fields := make([]string, 0)
	seen := map[string]struct{}{}
	for _, fn := range functionNames {
		if strings.HasPrefix(fn, "set") && len(fn) > 3 {
			field := strings.ToLower(fn[3:4]) + fn[4:]
			if field == "" {
				continue
			}
			if _, exists := seen[field]; exists {
				continue
			}
			seen[field] = struct{}{}
			fields = append(fields, field)
		}
	}
	sort.Strings(fields)
	return fields
}

func findActiveContractByType(contracts []*entities.SmartContract, t entities.SmartContractType) *entities.SmartContract {
	for _, c := range contracts {
		if c != nil && c.IsActive && c.Type == t {
			return c
		}
	}
	return nil
}

func mergeSummary(summary map[string]int, checks []ContractConfigCheckItem) {
	for _, check := range checks {
		switch strings.ToUpper(check.Status) {
		case "ERROR":
			summary["error"]++
		case "WARN":
			summary["warn"]++
		case "OK":
			summary["ok"]++
		}
	}
}

func deriveOverallStatus(summary map[string]int) string {
	if summary["error"] > 0 {
		return "ERROR"
	}
	if summary["warn"] > 0 {
		return "WARN"
	}
	return "OK"
}

func parseABI(raw string) (abi.ABI, error) {
	return abi.JSON(strings.NewReader(raw))
}

func callBoolView(ctx context.Context, client *blockchain.EVMClient, contractAddress, rawABI, method string, args ...interface{}) (bool, error) {
	parsed, err := parseABI(rawABI)
	if err != nil {
		return false, err
	}
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return false, err
	}
	out, err := client.CallView(ctx, contractAddress, data)
	if err != nil {
		return false, err
	}
	vals, err := parsed.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return false, fmt.Errorf("failed to decode bool result")
	}
	value, ok := vals[0].(bool)
	if !ok {
		return false, fmt.Errorf("unexpected bool result type")
	}
	return value, nil
}

func callUint8View(ctx context.Context, client *blockchain.EVMClient, contractAddress, rawABI, method string, args ...interface{}) (uint8, error) {
	parsed, err := parseABI(rawABI)
	if err != nil {
		return 0, err
	}
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return 0, err
	}
	out, err := client.CallView(ctx, contractAddress, data)
	if err != nil {
		return 0, err
	}
	vals, err := parsed.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return 0, fmt.Errorf("failed to decode uint8 result")
	}
	value, ok := vals[0].(uint8)
	if !ok {
		return 0, fmt.Errorf("unexpected uint8 result type")
	}
	return value, nil
}

func callUint64View(ctx context.Context, client *blockchain.EVMClient, contractAddress, rawABI, method string, args ...interface{}) (uint64, error) {
	parsed, err := parseABI(rawABI)
	if err != nil {
		return 0, err
	}
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return 0, err
	}
	out, err := client.CallView(ctx, contractAddress, data)
	if err != nil {
		return 0, err
	}
	vals, err := parsed.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return 0, fmt.Errorf("failed to decode uint64 result")
	}
	value, ok := vals[0].(uint64)
	if !ok {
		return 0, fmt.Errorf("unexpected uint64 result type")
	}
	return value, nil
}

func callAddressView(ctx context.Context, client *blockchain.EVMClient, contractAddress, rawABI, method string, args ...interface{}) (common.Address, error) {
	parsed, err := parseABI(rawABI)
	if err != nil {
		return common.Address{}, err
	}
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return common.Address{}, err
	}
	out, err := client.CallView(ctx, contractAddress, data)
	if err != nil {
		return common.Address{}, err
	}
	vals, err := parsed.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return common.Address{}, fmt.Errorf("failed to decode address result")
	}
	value, ok := vals[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("unexpected address result type")
	}
	return value, nil
}

func callBytesView(ctx context.Context, client *blockchain.EVMClient, contractAddress, rawABI, method string, args ...interface{}) ([]byte, error) {
	parsed, err := parseABI(rawABI)
	if err != nil {
		return nil, err
	}
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	out, err := client.CallView(ctx, contractAddress, data)
	if err != nil {
		return nil, err
	}
	vals, err := parsed.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return nil, fmt.Errorf("failed to decode bytes result")
	}
	value, ok := vals[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected bytes result type")
	}
	return value, nil
}
