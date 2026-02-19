package usecases

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestNormalizeBytes32Hex(t *testing.T) {
	t.Parallel()

	valid := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	got, arr, err := normalizeBytes32Hex(valid)
	if err != nil {
		t.Fatalf("expected valid bytes32, got err: %v", err)
	}
	want := "0x" + valid
	if got != want {
		t.Fatalf("unexpected normalized value: got=%s want=%s", got, want)
	}
	if arr[0] != 0x01 || arr[31] != 0xef {
		t.Fatalf("unexpected bytes32 decoded content")
	}

	if _, _, err := normalizeBytes32Hex("0x1234"); err == nil {
		t.Fatalf("expected invalid length error")
	}
}

func TestDecodeRouteErrorData_NoErrorRecorded(t *testing.T) {
	t.Parallel()

	got := decodeRouteErrorData(nil)
	if got.RawHex != "0x" {
		t.Fatalf("unexpected raw hex: %s", got.RawHex)
	}
	if got.Message != "no route error recorded" {
		t.Fatalf("unexpected message: %s", got.Message)
	}
}

func TestDecodeRouteErrorData_ErrorString(t *testing.T) {
	t.Parallel()

	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		t.Fatalf("failed to build abi type: %v", err)
	}
	args := abi.Arguments{{Type: stringType}}
	packed, err := args.Pack("route failed")
	if err != nil {
		t.Fatalf("failed to pack error string: %v", err)
	}
	data := append([]byte{0x08, 0xc3, 0x79, 0xa0}, packed...)

	got := decodeRouteErrorData(data)
	if got.Name != "Error" {
		t.Fatalf("expected Error name, got: %s", got.Name)
	}
	if got.Message != "route failed" {
		t.Fatalf("unexpected decoded message: %s", got.Message)
	}
}

func TestDecodeRouteErrorData_Panic(t *testing.T) {
	t.Parallel()

	panicCode := big.NewInt(0x11)
	codeBytes := make([]byte, 32)
	panicCode.FillBytes(codeBytes)
	data := append([]byte{0x4e, 0x48, 0x7b, 0x71}, codeBytes...)

	got := decodeRouteErrorData(data)
	if got.Name != "Panic" {
		t.Fatalf("expected Panic name, got: %s", got.Name)
	}
	if got.Message != "panic code: 17" {
		t.Fatalf("unexpected panic message: %s", got.Message)
	}
}

func TestDecodeRouteErrorData_KnownCustomSelector(t *testing.T) {
	t.Parallel()

	sel := selectorHex("RouteNotConfigured(string)")
	data := []byte{0x1, 0x2, 0x3, 0x4}
	copy(data, []byte{0, 0, 0, 0})
	decodedSel := decodeRouteErrorData(mustDecodeSelectorHex(sel))
	if decodedSel.Name != "RouteNotConfigured" {
		t.Fatalf("expected known custom selector name, got: %s", decodedSel.Name)
	}
	if decodedSel.Message != "RouteNotConfigured" {
		t.Fatalf("expected known selector message, got: %s", decodedSel.Message)
	}
}

func TestDecodeRouteErrorData_KnownCustomSelectorWithPayload(t *testing.T) {
	t.Parallel()

	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		t.Fatalf("failed to build abi type: %v", err)
	}
	args := abi.Arguments{{Type: stringType}}
	packed, err := args.Pack("eip155:42161")
	if err != nil {
		t.Fatalf("failed to pack args: %v", err)
	}
	selector := mustDecodeSelectorHex(selectorHex("RouteNotConfigured(string)"))
	data := append(selector, packed...)

	got := decodeRouteErrorData(data)
	if got.Name != "RouteNotConfigured" {
		t.Fatalf("expected RouteNotConfigured name, got: %s", got.Name)
	}
	if got.Message != "route not configured for destination eip155:42161" {
		t.Fatalf("unexpected message: %s", got.Message)
	}
	details, ok := got.Details.(map[string]any)
	if !ok {
		t.Fatalf("expected details map")
	}
	if details["destChainId"] != "eip155:42161" {
		t.Fatalf("unexpected details destChainId: %#v", details["destChainId"])
	}
}

func TestDecodeRouteErrorData_InsufficientNativeFeePayload(t *testing.T) {
	t.Parallel()

	uintType, err := abi.NewType("uint256", "", nil)
	if err != nil {
		t.Fatalf("failed to build abi type: %v", err)
	}
	args := abi.Arguments{{Type: uintType}, {Type: uintType}}
	packed, err := args.Pack(big.NewInt(200), big.NewInt(50))
	if err != nil {
		t.Fatalf("failed to pack args: %v", err)
	}
	selector := mustDecodeSelectorHex(selectorHex("InsufficientNativeFee(uint256,uint256)"))
	data := append(selector, packed...)

	got := decodeRouteErrorData(data)
	if got.Name != "InsufficientNativeFee" {
		t.Fatalf("expected InsufficientNativeFee name, got: %s", got.Name)
	}
	if got.Message != "insufficient native fee (required=200 wei, provided=50 wei)" {
		t.Fatalf("unexpected message: %s", got.Message)
	}
	details, ok := got.Details.(map[string]any)
	if !ok {
		t.Fatalf("expected details map")
	}
	if details["requiredWei"] != "200" || details["providedWei"] != "50" {
		t.Fatalf("unexpected details: %#v", details)
	}
}

func TestDecodeRouteErrorData_UnknownSelector(t *testing.T) {
	t.Parallel()

	got := decodeRouteErrorData([]byte{0xde, 0xad, 0xbe, 0xef, 0x01})
	if got.Selector != "0xdeadbeef" {
		t.Fatalf("unexpected selector: %s", got.Selector)
	}
	if got.Name != "" {
		t.Fatalf("expected empty name for unknown selector, got: %s", got.Name)
	}
	if got.Message != "execution_reverted" {
		t.Fatalf("unexpected message: %s", got.Message)
	}
}

func mustDecodeSelectorHex(s string) []byte {
	if len(s) != 10 || s[:2] != "0x" {
		panic("invalid selector hex")
	}
	return []byte{mustHexByte(s[2:4]), mustHexByte(s[4:6]), mustHexByte(s[6:8]), mustHexByte(s[8:10])}
}

func mustHexByte(v string) byte {
	const digits = "0123456789abcdef"
	var out byte
	for i := 0; i < 2; i++ {
		c := v[i]
		if c >= 'A' && c <= 'F' {
			c += 'a' - 'A'
		}
		idx := byte(255)
		for j := 0; j < len(digits); j++ {
			if digits[j] == c {
				idx = byte(j)
				break
			}
		}
		if idx == 255 {
			panic("invalid hex digit")
		}
		out = out<<4 | idx
	}
	return out
}
