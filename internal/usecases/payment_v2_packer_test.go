package usecases

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"math/big"
)

func sampleV2Req() PaymentRequestV2Args {
	return PaymentRequestV2Args{
		DestChainIDBytes:   []byte("eip155:137"),
		ReceiverBytes:      common.HexToAddress("0xE6A7d99011257AEc28Ad60EFED58A256c4d5Fea3").Bytes(),
		SourceToken:        common.HexToAddress("0x4200000000000000000000000000000000000006"),
		BridgeTokenSource:  common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
		DestToken:          common.HexToAddress("0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"),
		AmountInSource:     big.NewInt(100000),
		MinBridgeAmountOut: big.NewInt(99000),
		MinDestAmountOut:   big.NewInt(98000),
		Mode:               0,
		BridgeOption:       BridgeOptionDefaultSentinel,
	}
}

func TestPackCreatePaymentV2Calldata(t *testing.T) {
	calldata, err := packCreatePaymentV2Calldata(sampleV2Req())
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(calldata, CreatePaymentSelector))
	require.Greater(t, len(calldata), len(CreatePaymentSelector))
}

func TestPackCreatePaymentPrivateV2Calldata(t *testing.T) {
	req := sampleV2Req()
	privacy := PrivateRoutingArgs{
		IntentID:        [32]byte{1, 2, 3},
		StealthReceiver: common.HexToAddress("0x0000000000000000000000000000000000001234"),
	}

	calldata, err := packCreatePaymentPrivateV2Calldata(req, privacy)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(calldata, CreatePaymentPrivateSelector))
	require.Greater(t, len(calldata), len(CreatePaymentPrivateSelector))
}

func TestPackCreatePaymentDefaultBridgeV2Calldata(t *testing.T) {
	calldata, err := packCreatePaymentDefaultBridgeV2Calldata(sampleV2Req())
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(calldata, CreatePaymentDefaultBridgeSelector))
	require.Greater(t, len(calldata), len(CreatePaymentDefaultBridgeSelector))
}

func TestPackCreatePaymentV2Calldata_NilAmount(t *testing.T) {
	req := sampleV2Req()
	req.AmountInSource = nil
	_, err := packCreatePaymentV2Calldata(req)
	require.Error(t, err)
}
