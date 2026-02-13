package usecases

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/google/uuid"
)

func padLeft(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return strings.Repeat("0", length-len(s)) + s
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat("0", length-len(s))
}

func getChainTypeFromCAIP2(caip2 string) string {
	parts := strings.Split(caip2, ":")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func isEVMChain(chainID string) bool {
	return len(chainID) > 6 && chainID[:6] == "eip155"
}

func isSolanaChain(chainID string) bool {
	return len(chainID) > 6 && chainID[:6] == "solana"
}

func formatAmount(amount float64, decimals int) string {
	// Convert float to string with appropriate precision
	multiplier := new(big.Float).SetFloat64(amount)
	exp := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	result := new(big.Float).Mul(multiplier, exp)
	intResult, _ := result.Int(nil)
	return intResult.String()
}

func convertToSmallestUnit(amount string, decimals int) string {
	// Simple conversion - in production use proper decimal library
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)

	// Parse amount as float then convert
	amountFloat := new(big.Float)
	amountFloat.SetString(amount)

	multiplierFloat := new(big.Float).SetInt(multiplier)
	result := new(big.Float).Mul(amountFloat, multiplierFloat)

	resultInt, _ := result.Int(nil)
	return resultInt.String()
}

func uuidToBytes32Hex(id uuid.UUID) string {
	b := uuidToBytes32(id)
	hexID := hex.EncodeToString(b[:])
	return padLeft(hexID, EVMWordSizeHex)
}

func uuidToBytes32(id uuid.UUID) [32]byte {
	var out [32]byte
	copy(out[16:], id[:])
	return out
}

func anchorDiscriminator(name string) [8]byte {
	hash := sha256.Sum256([]byte("global:" + name))
	var out [8]byte
	copy(out[:], hash[:8])
	return out
}

func addDecimalStrings(a, b string) (string, error) {
	aa := new(big.Int)
	if _, ok := aa.SetString(a, 10); !ok {
		return "", fmt.Errorf("invalid decimal string: %s", a)
	}
	bb := new(big.Int)
	if _, ok := bb.SetString(b, 10); !ok {
		return "", fmt.Errorf("invalid decimal string: %s", b)
	}
	return new(big.Int).Add(aa, bb).String(), nil
}

func base58Encode(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	x := new(big.Int).SetBytes(data)
	base := big.NewInt(58)
	mod := new(big.Int)

	var out []byte
	for x.Sign() > 0 {
		x.DivMod(x, base, mod)
		out = append(out, alphabet[mod.Int64()])
	}

	for _, b := range data {
		if b != 0 {
			break
		}
		out = append(out, alphabet[0])
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

func base58Decode(s string) []byte {
	if s == "" {
		return nil
	}

	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	index := map[rune]int{}
	for i, c := range alphabet {
		index[c] = i
	}

	x := big.NewInt(0)
	base := big.NewInt(58)
	for _, c := range s {
		val, ok := index[c]
		if !ok {
			return nil
		}
		x.Mul(x, base)
		x.Add(x, big.NewInt(int64(val)))
	}

	decoded := x.Bytes()
	leadingOnes := 0
	for _, c := range s {
		if c != '1' {
			break
		}
		leadingOnes++
	}
	if leadingOnes > 0 {
		out := make([]byte, leadingOnes+len(decoded))
		copy(out[leadingOnes:], decoded)
		return out
	}
	return decoded
}

func normalizeEvmAddress(addr string) string {
	if addr == "" || addr == "native" || addr == "0x0000000000000000000000000000000000000000" {
		return "0x0000000000000000000000000000000000000000"
	}
	if !strings.HasPrefix(addr, "0x") {
		return "0x0000000000000000000000000000000000000000"
	}
	return addr
}

func encodeAnchorString(s string) []byte {
	out := make([]byte, 4+len(s))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(s)))
	copy(out[4:], []byte(s))
	return out
}

func u64ToLE(v uint64) []byte {
	out := make([]byte, 8)
	binary.LittleEndian.PutUint64(out, v)
	return out
}

func decimalStringToUint64(value string) uint64 {
	n := new(big.Int)
	if _, ok := n.SetString(value, 10); !ok || n.Sign() < 0 {
		return 0
	}
	if n.BitLen() > 64 {
		return math.MaxUint64
	}
	return n.Uint64()
}

func addressToBytes32(addr string) [32]byte {
	var out [32]byte
	if addr == "" {
		return out
	}

	if strings.HasPrefix(addr, "0x") {
		if b, err := hex.DecodeString(strings.TrimPrefix(addr, "0x")); err == nil {
			if len(b) > 32 {
				b = b[len(b)-32:]
			}
			copy(out[32-len(b):], b)
			return out
		}
	}

	if decoded := base58Decode(addr); len(decoded) > 0 {
		if len(decoded) > 32 {
			decoded = decoded[len(decoded)-32:]
		}
		copy(out[32-len(decoded):], decoded)
		return out
	}

	raw := []byte(addr)
	if len(raw) > 32 {
		raw = raw[len(raw)-32:]
	}
	copy(out[32-len(raw):], raw)
	return out
}
