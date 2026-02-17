package entities

import "testing"

func TestChain_GetCAIP2ID(t *testing.T) {
	evm := &Chain{Type: ChainTypeEVM, ChainID: "8453"}
	if got := evm.GetCAIP2ID(); got != "eip155:8453" {
		t.Fatalf("expected eip155:8453 got %s", got)
	}

	svm := &Chain{Type: ChainTypeSVM, ChainID: "devnet"}
	if got := svm.GetCAIP2ID(); got != "solana:devnet" {
		t.Fatalf("expected solana:devnet got %s", got)
	}

	caip2 := &Chain{Type: ChainTypeEVM, ChainID: "eip155:42161"}
	if got := caip2.GetCAIP2ID(); got != "eip155:42161" {
		t.Fatalf("expected eip155:42161 got %s", got)
	}

	trimmed := &Chain{Type: ChainTypeEVM, ChainID: "  10  "}
	if got := trimmed.GetCAIP2ID(); got != "eip155:10" {
		t.Fatalf("expected eip155:10 got %s", got)
	}

	substrate := &Chain{Type: ChainTypeSubstrate, ChainID: "polkadot"}
	if got := substrate.GetCAIP2ID(); got != "polkadot" {
		t.Fatalf("expected polkadot got %s", got)
	}
}
