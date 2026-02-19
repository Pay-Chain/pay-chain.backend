package usecases

import "testing"

func TestFormatDecodedRouteErrorForPreflight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		decoded RouteErrorDecoded
		want    string
	}{
		{
			name:    "prefer detailed message",
			decoded: RouteErrorDecoded{Name: "RouteNotConfigured", Message: "route not configured for destination eip155:42161"},
			want:    "route not configured for destination eip155:42161",
		},
		{
			name:    "fallback by known name",
			decoded: RouteErrorDecoded{Name: "ChainSelectorMissing", Message: "execution_reverted"},
			want:    "ccip chain selector missing",
		},
		{
			name:    "unknown name returns name",
			decoded: RouteErrorDecoded{Name: "CustomFailure", Message: "execution_reverted"},
			want:    "CustomFailure",
		},
		{
			name:    "selector fallback",
			decoded: RouteErrorDecoded{Selector: "0xdeadbeef", Message: "execution_reverted"},
			want:    "execution_reverted (0xdeadbeef)",
		},
		{
			name:    "generic fallback",
			decoded: RouteErrorDecoded{Message: "execution_reverted"},
			want:    "execution_reverted",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatDecodedRouteErrorForPreflight(tt.decoded)
			if got != tt.want {
				t.Fatalf("unexpected formatted message: got=%q want=%q", got, tt.want)
			}
		})
	}
}

