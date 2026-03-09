package usecases

import "testing"

func TestIsQuoteSchemaMismatchReason(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want bool
	}{
		{"function selector was not recognized and there's no fallback function", true},
		{"no method with id 0x12345678", true},
		{"execution reverted", false},
		{"route not configured", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := isQuoteSchemaMismatchReason(tc.in); got != tc.want {
				t.Fatalf("unexpected match: got=%v want=%v", got, tc.want)
			}
		})
	}
}

