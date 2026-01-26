package usecases

import (
	"math/big"
	"strings"
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
