package usecases

import (
	"encoding/hex"
	"regexp"
	"strings"
)

// decodeRevertDataFromError attempts to parse hex-encoded revert bytes from RPC errors.
// It supports rpc.DataError payloads and fallback extraction from error strings.
func decodeRevertDataFromError(err error) (RouteErrorDecoded, bool) {
	if err == nil {
		return RouteErrorDecoded{}, false
	}

	if data, ok := extractRevertHexFromDataError(err); ok {
		decoded := decodeRouteErrorData(data)
		return decoded, true
	}

	if data, ok := extractRevertHexFromErrorString(err.Error()); ok {
		decoded := decodeRouteErrorData(data)
		return decoded, true
	}

	return RouteErrorDecoded{}, false
}

func extractRevertHexFromDataError(err error) ([]byte, bool) {
	type rpcDataError interface {
		ErrorData() interface{}
	}
	dataErr, ok := err.(rpcDataError)
	if !ok {
		return nil, false
	}
	return parseRevertBytesFromAny(dataErr.ErrorData())
}

func parseRevertBytesFromAny(value interface{}) ([]byte, bool) {
	switch v := value.(type) {
	case string:
		return parseHexBytes(v)
	case []byte:
		if len(v) == 0 {
			return nil, false
		}
		out := make([]byte, len(v))
		copy(out, v)
		return out, true
	case map[string]interface{}:
		if raw, ok := v["data"]; ok {
			return parseRevertBytesFromAny(raw)
		}
		if raw, ok := v["result"]; ok {
			return parseRevertBytesFromAny(raw)
		}
	case map[string]string:
		if raw, ok := v["data"]; ok {
			return parseHexBytes(raw)
		}
		if raw, ok := v["result"]; ok {
			return parseHexBytes(raw)
		}
	}
	return nil, false
}

func extractRevertHexFromErrorString(message string) ([]byte, bool) {
	pattern := regexp.MustCompile(`0x[0-9a-fA-F]{8,}`)
	hexValues := pattern.FindAllString(message, -1)
	for _, candidate := range hexValues {
		if data, ok := parseHexBytes(candidate); ok {
			return data, true
		}
	}
	return nil, false
}

func parseHexBytes(raw string) ([]byte, bool) {
	value := strings.TrimSpace(strings.TrimPrefix(raw, "0x"))
	if len(value) < 8 || len(value)%2 != 0 {
		return nil, false
	}
	data, err := hex.DecodeString(value)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}
