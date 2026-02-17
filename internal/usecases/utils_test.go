package usecases

import (
	"encoding/hex"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPadHelpers(t *testing.T) {
	assert.Equal(t, "000abc", padLeft("abc", 6))
	assert.Equal(t, "abc000", padRight("abc", 6))
	assert.Equal(t, "abcdef", padLeft("abcdef", 3))
	assert.Equal(t, "abcdef", padRight("abcdef", 3))
}

func TestChainTypeHelpers(t *testing.T) {
	assert.Equal(t, "eip155", getChainTypeFromCAIP2("eip155:8453"))
	assert.Equal(t, "eip155", getChainTypeFromCAIP2("  eip155:8453  "))
	assert.Equal(t, "", getChainTypeFromCAIP2(""))
	assert.True(t, isEVMChain("eip155:8453"))
	assert.False(t, isEVMChain("solana:devnet"))
	assert.True(t, isSolanaChain("solana:devnet"))
	assert.False(t, isSolanaChain("eip155:8453"))
}

func TestConvertToSmallestUnit(t *testing.T) {
	tests := []struct {
		name     string
		amount   string
		decimals int
		want     string
		wantErr  bool
	}{
		{"integer", "1", 6, "1000000", false},
		{"fraction", "1.23", 6, "1230000", false},
		{"leading plus", "+2.5", 6, "2500000", false},
		{"zero", "0", 6, "0", false},
		{"blank", "", 6, "", true},
		{"negative", "-1", 6, "", true},
		{"non numeric", "a1", 6, "", true},
		{"too many decimals", "1.1234567", 6, "", true},
		{"bad decimals", "1", -1, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToSmallestUnit(tt.amount, tt.decimals)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAddDecimalStrings(t *testing.T) {
	got, err := addDecimalStrings("100", "25")
	assert.NoError(t, err)
	assert.Equal(t, "125", got)

	_, err = addDecimalStrings("x", "1")
	assert.Error(t, err)

	_, err = addDecimalStrings("1", "y")
	assert.Error(t, err)
}

func TestBase58EncodeDecode(t *testing.T) {
	raw := []byte{0, 0, 1, 2, 3, 4, 5}
	encoded := base58Encode(raw)
	decoded := base58Decode(encoded)
	assert.Equal(t, raw, decoded)

	assert.Nil(t, base58Decode("0OIl")) // invalid alphabet chars
	assert.Equal(t, "", base58Encode(nil))
}

func TestNormalizeEvmAddress(t *testing.T) {
	assert.Equal(t, "0x0000000000000000000000000000000000000000", normalizeEvmAddress(""))
	assert.Equal(t, "0x0000000000000000000000000000000000000000", normalizeEvmAddress("native"))
	assert.Equal(t, "0x0000000000000000000000000000000000000000", normalizeEvmAddress("abc"))
	assert.Equal(t, "0xabc", normalizeEvmAddress("0xabc"))
}

func TestBinaryHelpers(t *testing.T) {
	encoded := encodeAnchorString("abc")
	assert.Len(t, encoded, 7)
	assert.Equal(t, []byte("abc"), encoded[4:])

	le := u64ToLE(1024)
	assert.Len(t, le, 8)
}

func TestDecimalStringToUint64(t *testing.T) {
	assert.Equal(t, uint64(123), decimalStringToUint64("123"))
	assert.Equal(t, uint64(0), decimalStringToUint64("-1"))
	assert.Equal(t, uint64(0), decimalStringToUint64("not-num"))
	assert.Equal(t, uint64(math.MaxUint64), decimalStringToUint64("1844674407370955161600"))
}

func TestAddressToBytes32(t *testing.T) {
	empty := addressToBytes32("")
	assert.Equal(t, [32]byte{}, empty)

	evm := addressToBytes32("0x000000000000000000000000000000000000dEaD")
	assert.Equal(t, "000000000000000000000000000000000000000000000000000000000000dead", hex.EncodeToString(evm[:]))

	tooLongHex := addressToBytes32("0x111111111111111111111111111111111111111111111111111111111111111111")
	assert.NotEqual(t, [32]byte{}, tooLongHex)

	solRaw := addressToBytes32("2")
	assert.NotEqual(t, [32]byte{}, solRaw)

	longASCII := addressToBytes32("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	assert.NotEqual(t, [32]byte{}, longASCII)

	textRaw := addressToBytes32("plain-address")
	assert.NotEqual(t, [32]byte{}, textRaw)

	// "0x" prefix with invalid hex should fall back to base58/ascii handling.
	invalidHexPrefixed := addressToBytes32("0xzzzz")
	assert.NotEqual(t, [32]byte{}, invalidHexPrefixed)

	// Base58 decoded payload > 32 bytes should be right-trimmed.
	longRaw := make([]byte, 40)
	for i := range longRaw {
		longRaw[i] = byte(i + 1)
	}
	longBase58 := base58Encode(longRaw)
	longDecoded := addressToBytes32(longBase58)
	assert.NotEqual(t, [32]byte{}, longDecoded)
}

func TestUUIDAndAnchorHelpers(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	bytes32 := uuidToBytes32(id)
	assert.Equal(t, byte(1), bytes32[31])

	hexed := uuidToBytes32Hex(id)
	assert.Len(t, hexed, 64)

	d := anchorDiscriminator("create_payment")
	assert.Len(t, d, 8)
}
