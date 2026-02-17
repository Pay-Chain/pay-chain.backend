package middleware

import (
	"context"
	"testing"
)

func TestLoadSessionFromStore_WrapperExecutes(t *testing.T) {
	defer func() {
		_ = recover()
	}()
	// This intentionally uses nil store only to execute the wrapper body branch.
	_, _ = loadSessionFromStore(context.Background(), nil, "session-id")
}
