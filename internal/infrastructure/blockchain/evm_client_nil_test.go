package blockchain

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
)

func expectPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic, got nil")
		}
	}()
	fn()
}

func TestEVMClient_Methods_PanicWhenClientNil(t *testing.T) {
	c := &EVMClient{client: nil, chainID: big.NewInt(1), rpcURL: "http://unused"}
	ctx := context.Background()

	expectPanic(t, func() { _, _ = c.GetBalance(ctx, "0x1111111111111111111111111111111111111111") })
	expectPanic(t, func() {
		_, _ = c.GetTokenBalance(ctx, "0x2222222222222222222222222222222222222222", "0x1111111111111111111111111111111111111111")
	})
	expectPanic(t, func() {
		_, _, _ = c.GetTransaction(ctx, "0x1111111111111111111111111111111111111111111111111111111111111111")
	})
	expectPanic(t, func() {
		_, _ = c.GetTransactionReceipt(ctx, "0x1111111111111111111111111111111111111111111111111111111111111111")
	})
	expectPanic(t, func() { _, _ = c.GetBlockNumber(ctx) })
	expectPanic(t, func() { _, _ = c.EstimateGas(ctx, ethereum.CallMsg{}) })
	expectPanic(t, func() { _, _ = c.CallView(ctx, "0x3333333333333333333333333333333333333333", []byte{0x12, 0x34}) })

	// Close is intentionally no-op when underlying client is nil.
	c.Close()
}
