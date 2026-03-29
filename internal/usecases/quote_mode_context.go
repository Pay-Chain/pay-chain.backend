package usecases

import "context"

type quoteModeKeyType struct{}

var quoteModeKey = quoteModeKeyType{}

type quoteModeOptions struct {
	preferDryRunQuote bool
}

func withPreferDryRunQuote(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	current, _ := ctx.Value(quoteModeKey).(*quoteModeOptions)
	if current != nil && current.preferDryRunQuote {
		return ctx
	}
	next := &quoteModeOptions{
		preferDryRunQuote: true,
	}
	return context.WithValue(ctx, quoteModeKey, next)
}

func preferDryRunQuote(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	mode, _ := ctx.Value(quoteModeKey).(*quoteModeOptions)
	return mode != nil && mode.preferDryRunQuote
}

