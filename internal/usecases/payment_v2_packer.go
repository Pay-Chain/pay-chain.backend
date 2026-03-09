package usecases

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type PaymentRequestV2Args struct {
	DestChainIDBytes   []byte
	ReceiverBytes      []byte
	SourceToken        common.Address
	BridgeTokenSource  common.Address
	DestToken          common.Address
	AmountInSource     *big.Int
	MinBridgeAmountOut *big.Int
	MinDestAmountOut   *big.Int
	Mode               uint8
	BridgeOption       uint8
}

type PrivateRoutingArgs struct {
	IntentID        [32]byte
	StealthReceiver common.Address
}

type paymentRequestV2TupleValue struct {
	DestChainIdBytes   []byte         `abi:"destChainIdBytes"`
	ReceiverBytes      []byte         `abi:"receiverBytes"`
	SourceToken        common.Address `abi:"sourceToken"`
	BridgeTokenSource  common.Address `abi:"bridgeTokenSource"`
	DestToken          common.Address `abi:"destToken"`
	AmountInSource     *big.Int       `abi:"amountInSource"`
	MinBridgeAmountOut *big.Int       `abi:"minBridgeAmountOut"`
	MinDestAmountOut   *big.Int       `abi:"minDestAmountOut"`
	Mode               uint8          `abi:"mode"`
	BridgeOption       uint8          `abi:"bridgeOption"`
}

type privateRoutingTupleValue struct {
	IntentId        [32]byte       `abi:"intentId"`
	StealthReceiver common.Address `abi:"stealthReceiver"`
}

var (
	paymentRequestV2TupleType = mustNewType("tuple", []abi.ArgumentMarshaling{
		{Name: "destChainIdBytes", Type: "bytes"},
		{Name: "receiverBytes", Type: "bytes"},
		{Name: "sourceToken", Type: "address"},
		{Name: "bridgeTokenSource", Type: "address"},
		{Name: "destToken", Type: "address"},
		{Name: "amountInSource", Type: "uint256"},
		{Name: "minBridgeAmountOut", Type: "uint256"},
		{Name: "minDestAmountOut", Type: "uint256"},
		{Name: "mode", Type: "uint8"},
		{Name: "bridgeOption", Type: "uint8"},
	})
	privateRoutingTupleType = mustNewType("tuple", []abi.ArgumentMarshaling{
		{Name: "intentId", Type: "bytes32"},
		{Name: "stealthReceiver", Type: "address"},
	})
)

func mustNewType(t string, components []abi.ArgumentMarshaling) abi.Type {
	typ, err := abi.NewType(t, "", components)
	if err != nil {
		panic(err)
	}
	return typ
}

func packCreatePaymentV2Calldata(req PaymentRequestV2Args) (string, error) {
	if req.AmountInSource == nil || req.MinBridgeAmountOut == nil || req.MinDestAmountOut == nil {
		return "", fmt.Errorf("pack createPaymentV2 args: amount fields must not be nil")
	}

	args := abi.Arguments{{Type: paymentRequestV2TupleType}}
	packed, err := packABIArgs(args, paymentRequestV2TupleValue{
		DestChainIdBytes:   req.DestChainIDBytes,
		ReceiverBytes:      req.ReceiverBytes,
		SourceToken:        req.SourceToken,
		BridgeTokenSource:  req.BridgeTokenSource,
		DestToken:          req.DestToken,
		AmountInSource:     req.AmountInSource,
		MinBridgeAmountOut: req.MinBridgeAmountOut,
		MinDestAmountOut:   req.MinDestAmountOut,
		Mode:               req.Mode,
		BridgeOption:       req.BridgeOption,
	})
	if err != nil {
		return "", fmt.Errorf("pack createPaymentV2 args: %w", err)
	}
	return buildCalldataHex(CreatePaymentSelector, packed)
}

func packCreatePaymentDefaultBridgeV2Calldata(req PaymentRequestV2Args) (string, error) {
	if req.AmountInSource == nil || req.MinBridgeAmountOut == nil || req.MinDestAmountOut == nil {
		return "", fmt.Errorf("pack createPaymentDefaultBridgeV2 args: amount fields must not be nil")
	}

	args := abi.Arguments{{Type: paymentRequestV2TupleType}}
	packed, err := packABIArgs(args, paymentRequestV2TupleValue{
		DestChainIdBytes:   req.DestChainIDBytes,
		ReceiverBytes:      req.ReceiverBytes,
		SourceToken:        req.SourceToken,
		BridgeTokenSource:  req.BridgeTokenSource,
		DestToken:          req.DestToken,
		AmountInSource:     req.AmountInSource,
		MinBridgeAmountOut: req.MinBridgeAmountOut,
		MinDestAmountOut:   req.MinDestAmountOut,
		Mode:               req.Mode,
		BridgeOption:       req.BridgeOption,
	})
	if err != nil {
		return "", fmt.Errorf("pack createPaymentDefaultBridgeV2 args: %w", err)
	}
	return buildCalldataHex(CreatePaymentDefaultBridgeSelector, packed)
}

func packCreatePaymentPrivateV2Calldata(req PaymentRequestV2Args, privacy PrivateRoutingArgs) (string, error) {
	if req.AmountInSource == nil || req.MinBridgeAmountOut == nil || req.MinDestAmountOut == nil {
		return "", fmt.Errorf("pack createPaymentPrivateV2 args: amount fields must not be nil")
	}

	args := abi.Arguments{
		{Type: paymentRequestV2TupleType},
		{Type: privateRoutingTupleType},
	}
	packed, err := packABIArgs(args,
		paymentRequestV2TupleValue{
			DestChainIdBytes:   req.DestChainIDBytes,
			ReceiverBytes:      req.ReceiverBytes,
			SourceToken:        req.SourceToken,
			BridgeTokenSource:  req.BridgeTokenSource,
			DestToken:          req.DestToken,
			AmountInSource:     req.AmountInSource,
			MinBridgeAmountOut: req.MinBridgeAmountOut,
			MinDestAmountOut:   req.MinDestAmountOut,
			Mode:               req.Mode,
			BridgeOption:       req.BridgeOption,
		},
		privateRoutingTupleValue{
			IntentId:        privacy.IntentID,
			StealthReceiver: privacy.StealthReceiver,
		},
	)
	if err != nil {
		return "", fmt.Errorf("pack createPaymentPrivateV2 args: %w", err)
	}
	return buildCalldataHex(CreatePaymentPrivateSelector, packed)
}

func buildCalldataHex(selector string, packedArgs []byte) (string, error) {
	selectorBytes, err := hex.DecodeString(strings.TrimPrefix(selector, "0x"))
	if err != nil {
		return "", fmt.Errorf("decode selector: %w", err)
	}
	return "0x" + hex.EncodeToString(append(selectorBytes, packedArgs...)), nil
}
