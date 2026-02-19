package main

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	sigs := []string{
		"NoRouteFound()",
		"SlippageExceeded()",
		"InvalidAddress()",
		"Unauthorized()",
		"SameToken()",
		"ZeroAmount()",
		"PoolNotActive()",
		"Gateway_InvalidChainId()",
		"ExecuteFailed(bytes)",
		"Panic(uint256)",
		"Error(string)",
	}

	for _, sig := range sigs {
		hash := crypto.Keccak256([]byte(sig))
		selector := hex.EncodeToString(hash[:4])
		fmt.Printf("%s: 0x%s\n", sig, selector)
	}
}
