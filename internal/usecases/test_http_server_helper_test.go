package usecases

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newSafeHTTPServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("skip: httptest server unavailable in this environment: %v", r)
		}
	}()
	return httptest.NewServer(handler)
}

