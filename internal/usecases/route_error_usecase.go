package usecases

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/blockchain"
)

type RouteErrorDecoded struct {
	RawHex   string `json:"rawHex"`
	Selector string `json:"selector,omitempty"`
	Name     string `json:"name,omitempty"`
	Message  string `json:"message,omitempty"`
	Details  any    `json:"details,omitempty"`
}

type RouteErrorDiagnostics struct {
	SourceChainID string            `json:"sourceChainId"`
	Gateway       string            `json:"gatewayAddress"`
	PaymentIDHex  string            `json:"paymentIdHex"`
	Decoded       RouteErrorDecoded `json:"decoded"`
}

type RouteErrorUsecase struct {
	chainRepo     repositories.ChainRepository
	contractRepo  repositories.SmartContractRepository
	clientFactory *blockchain.ClientFactory
	chainResolver *ChainResolver
}

func NewRouteErrorUsecase(
	chainRepo repositories.ChainRepository,
	contractRepo repositories.SmartContractRepository,
	clientFactory *blockchain.ClientFactory,
) *RouteErrorUsecase {
	return &RouteErrorUsecase{
		chainRepo:     chainRepo,
		contractRepo:  contractRepo,
		clientFactory: clientFactory,
		chainResolver: NewChainResolver(chainRepo),
	}
}

func (u *RouteErrorUsecase) GetRouteError(
	ctx context.Context,
	sourceChainInput string,
	paymentIDHex string,
) (*RouteErrorDiagnostics, error) {
	sourceChainUUID, sourceCAIP2, err := u.chainResolver.ResolveFromAny(ctx, sourceChainInput)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid sourceChainId")
	}
	sourceChain, err := u.chainRepo.GetByID(ctx, sourceChainUUID)
	if err != nil {
		return nil, err
	}

	gateway, err := u.contractRepo.GetActiveContract(ctx, sourceChain.ID, entities.ContractTypeGateway)
	if err != nil || gateway == nil || !common.IsHexAddress(gateway.ContractAddress) {
		return nil, domainerrors.BadRequest("active gateway contract not found on source chain")
	}

	normalizedPID, pidBytes, err := normalizeBytes32Hex(paymentIDHex)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid paymentId (must be bytes32 hex)")
	}

	rpcURL := resolveRPCURL(sourceChain)
	if rpcURL == "" {
		return nil, domainerrors.BadRequest("no active rpc url for source chain")
	}
	client, err := u.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	args := abi.Arguments{{Type: bytes32Type}}
	packedArgs, err := args.Pack(pidBytes)
	if err != nil {
		return nil, err
	}

	methodID := crypto.Keccak256([]byte("lastRouteError(bytes32)"))[:4]
	out, err := client.CallView(ctx, gateway.ContractAddress, append(methodID, packedArgs...))
	if err != nil {
		return nil, err
	}

	decodedBytes, err := decodeBytesReturn(out)
	if err != nil {
		return nil, err
	}

	decoded := decodeRouteErrorData(decodedBytes)
	return &RouteErrorDiagnostics{
		SourceChainID: sourceCAIP2,
		Gateway:       gateway.ContractAddress,
		PaymentIDHex:  normalizedPID,
		Decoded:       decoded,
	}, nil
}

func normalizeBytes32Hex(value string) (string, [32]byte, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(value, "0x"))
	if len(trimmed) != 64 {
		return "", [32]byte{}, fmt.Errorf("invalid bytes32 length")
	}
	raw, err := hex.DecodeString(trimmed)
	if err != nil {
		return "", [32]byte{}, err
	}
	var out [32]byte
	copy(out[:], raw)
	return "0x" + strings.ToLower(trimmed), out, nil
}

func decodeBytesReturn(out []byte) ([]byte, error) {
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	outputs := abi.Arguments{{Type: bytesType}}
	decoded, err := outputs.Unpack(out)
	if err != nil {
		return nil, err
	}
	if len(decoded) != 1 {
		return nil, fmt.Errorf("invalid bytes return")
	}
	data, ok := decoded[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid bytes type")
	}
	return data, nil
}

func decodeRouteErrorData(data []byte) RouteErrorDecoded {
	result := RouteErrorDecoded{
		RawHex: "0x" + hex.EncodeToString(data),
	}
	if len(data) < 4 {
		if len(data) == 0 {
			result.Message = "no route error recorded"
		} else {
			result.Message = "execution_reverted"
		}
		return result
	}

	selectorHex := "0x" + hex.EncodeToString(data[:4])
	result.Selector = selectorHex
	if name := knownSelectorName(selectorHex); name != "" {
		result.Name = name
	}

	// Error(string)
	if selectorHex == "0x08c379a0" && len(data) > 4 {
		stringType, err := abi.NewType("string", "", nil)
		if err == nil {
			outputs := abi.Arguments{{Type: stringType}}
			if values, unpackErr := outputs.Unpack(data[4:]); unpackErr == nil && len(values) == 1 {
				if msg, ok := values[0].(string); ok {
					result.Name = "Error"
					result.Message = msg
					return result
				}
			}
		}
	}

	// Panic(uint256)
	if selectorHex == "0x4e487b71" && len(data) >= 36 {
		result.Name = "Panic"
		result.Message = fmt.Sprintf("panic code: %s", new(big.Int).SetBytes(data[4:36]).String())
		return result
	}

	if payloadDetails, ok := decodeKnownCustomErrorPayload(selectorHex, data[4:]); ok {
		if detailMsg := buildDetailMessage(result.Name, payloadDetails); detailMsg != "" {
			result.Message = detailMsg
		}
		result.Details = payloadDetails
		return result
	}

	if result.Name != "" {
		result.Message = result.Name
	} else {
		result.Message = "execution_reverted"
	}
	return result
}

func knownSelectorName(selector string) string {
	switch strings.ToLower(strings.TrimSpace(selector)) {
	case "0x08c379a0":
		return "Error"
	case "0x4e487b71":
		return "Panic"
	case selectorHex("ChainSelectorMissing(string)"):
		return "ChainSelectorMissing"
	case selectorHex("DestinationAdapterMissing(string)"):
		return "DestinationAdapterMissing"
	case selectorHex("StateMachineIdNotSet(string)"):
		return "StateMachineIdNotSet"
	case selectorHex("DestinationNotSet(string)"):
		return "DestinationNotSet"
	case selectorHex("NativeFeeQuoteUnavailable()"):
		return "NativeFeeQuoteUnavailable"
	case selectorHex("FeeQuoteFailed(uint256,address[])"):
		return "FeeQuoteFailed"
	case selectorHex("InsufficientNativeFee(uint256,uint256)"):
		return "InsufficientNativeFee"
	case selectorHex("RouteNotConfigured(string)"):
		return "RouteNotConfigured"
	case selectorHex("CCIPTokenMismatch(bytes32,address,address)"):
		return "CCIPTokenMismatch"
	case selectorHex("UntrustedSource(uint64,bytes)"):
		return "UntrustedSource"
	case selectorHex("UntrustedPeer(uint32,bytes32)"):
		return "UntrustedPeer"
	default:
		return ""
	}
}

func selectorHex(signature string) string {
	return "0x" + hex.EncodeToString(crypto.Keccak256([]byte(signature))[:4])
}

func decodeKnownCustomErrorPayload(selector string, payload []byte) (map[string]any, bool) {
	switch strings.ToLower(strings.TrimSpace(selector)) {
	case selectorHex("ChainSelectorMissing(string)"),
		selectorHex("DestinationAdapterMissing(string)"),
		selectorHex("StateMachineIdNotSet(string)"),
		selectorHex("DestinationNotSet(string)"),
		selectorHex("RouteNotConfigured(string)"):
		values, ok := unpackCustomErrorArgs(payload, []string{"string"})
		if !ok || len(values) != 1 {
			return nil, false
		}
		value, castOK := values[0].(string)
		if !castOK {
			return nil, false
		}
		return map[string]any{"destChainId": value}, true
	case selectorHex("InsufficientNativeFee(uint256,uint256)"):
		values, ok := unpackCustomErrorArgs(payload, []string{"uint256", "uint256"})
		if !ok || len(values) != 2 {
			return nil, false
		}
		required, okReq := values[0].(*big.Int)
		provided, okProv := values[1].(*big.Int)
		if !okReq || !okProv {
			return nil, false
		}
		return map[string]any{
			"requiredWei": required.String(),
			"providedWei": provided.String(),
		}, true
	case selectorHex("CCIPTokenMismatch(bytes32,address,address)"):
		values, ok := unpackCustomErrorArgs(payload, []string{"bytes32", "address", "address"})
		if !ok || len(values) != 3 {
			return nil, false
		}
		chain, okChain := values[0].([32]byte)
		source, okSource := values[1].(common.Address)
		dest, okDest := values[2].(common.Address)
		if !okChain || !okSource || !okDest {
			return nil, false
		}
		return map[string]any{
			"destChainIdHex": "0x" + hex.EncodeToString(chain[:]),
			"sourceToken":    source.Hex(),
			"destToken":      dest.Hex(),
		}, true
	case selectorHex("UntrustedSource(uint64,bytes)"):
		values, ok := unpackCustomErrorArgs(payload, []string{"uint64", "bytes"})
		if !ok || len(values) != 2 {
			return nil, false
		}
		selector, okSel := values[0].(uint64)
		sender, okSender := values[1].([]byte)
		if !okSel || !okSender {
			return nil, false
		}
		return map[string]any{
			"sourceSelector": selector,
			"sourceSender":   "0x" + hex.EncodeToString(sender),
		}, true
	case selectorHex("UntrustedPeer(uint32,bytes32)"):
		values, ok := unpackCustomErrorArgs(payload, []string{"uint32", "bytes32"})
		if !ok || len(values) != 2 {
			return nil, false
		}
		eid, okEid := values[0].(uint32)
		peer, okPeer := values[1].([32]byte)
		if !okEid || !okPeer {
			return nil, false
		}
		return map[string]any{
			"srcEid": eid,
			"peer":   "0x" + hex.EncodeToString(peer[:]),
		}, true
	case selectorHex("FeeQuoteFailed(uint256,address[])"):
		values, ok := unpackCustomErrorArgs(payload, []string{"uint256", "address[]"})
		if !ok || len(values) != 2 {
			return nil, false
		}
		fee, okFee := values[0].(*big.Int)
		tokens, okTokens := values[1].([]common.Address)
		if !okFee || !okTokens {
			return nil, false
		}
		tokenStrs := make([]string, len(tokens))
		for i, t := range tokens {
			tokenStrs[i] = t.Hex()
		}
		return map[string]any{
			"requiredFee": fee.String(),
			"tokens":      tokenStrs,
		}, true
	default:
		return nil, false
	}
}

func unpackCustomErrorArgs(payload []byte, typeNames []string) ([]any, bool) {
	args := make(abi.Arguments, 0, len(typeNames))
	for _, typeName := range typeNames {
		argType, err := abi.NewType(typeName, "", nil)
		if err != nil {
			return nil, false
		}
		args = append(args, abi.Argument{Type: argType})
	}
	values, err := args.Unpack(payload)
	if err != nil {
		return nil, false
	}
	return values, true
}

func buildDetailMessage(name string, details map[string]any) string {
	switch name {
	case "RouteNotConfigured":
		if v, ok := details["destChainId"].(string); ok && v != "" {
			return "route not configured for destination " + v
		}
	case "ChainSelectorMissing":
		if v, ok := details["destChainId"].(string); ok && v != "" {
			return "ccip chain selector missing for destination " + v
		}
	case "DestinationAdapterMissing":
		if v, ok := details["destChainId"].(string); ok && v != "" {
			return "destination adapter missing for destination " + v
		}
	case "StateMachineIdNotSet":
		if v, ok := details["destChainId"].(string); ok && v != "" {
			return "hyperbridge state machine id not set for destination " + v
		}
	case "DestinationNotSet":
		if v, ok := details["destChainId"].(string); ok && v != "" {
			return "hyperbridge destination not set for destination " + v
		}
	case "InsufficientNativeFee":
		required, _ := details["requiredWei"].(string)
		provided, _ := details["providedWei"].(string)
		if required != "" && provided != "" {
			return fmt.Sprintf("insufficient native fee (required=%s wei, provided=%s wei)", required, provided)
		}
	case "FeeQuoteFailed":
		required, _ := details["requiredFee"].(string)
		if required != "" {
			return fmt.Sprintf("fee quote failed (required fee: %s native)", required)
		}
		return "fee quote failed"
	}
	return ""
}
