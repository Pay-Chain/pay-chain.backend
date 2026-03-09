package usecases

import "strings"

// isQuoteSchemaMismatchReason returns true when an error/reason suggests
// ABI/signature mismatch between caller and on-chain contract methods.
func isQuoteSchemaMismatchReason(reason string) bool {
	r := strings.ToLower(strings.TrimSpace(reason))
	if r == "" {
		return false
	}
	keywords := []string{
		"function selector was not recognized",
		"selector was not recognized",
		"unknown selector",
		"no fallback function",
		"no method with id",
		"method not found",
		"function does not exist",
		"abi mismatch",
	}
	for _, kw := range keywords {
		if strings.Contains(r, kw) {
			return true
		}
	}
	return false
}

