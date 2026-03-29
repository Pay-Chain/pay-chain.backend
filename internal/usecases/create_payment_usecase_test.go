package usecases

import (
	"errors"
	"testing"

	domainerrors "payment-kita.backend/internal/domain/errors"
)

func TestShouldTreatQuoteProbeAsZero(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantTreat  bool
		wantReason string
	}{
		{
			name:       "placeholder quote",
			err:        domainerrors.BadRequest("accurate quote unavailable for selected pair: swapper returned 1:1 placeholder quote"),
			wantTreat:  true,
			wantReason: "accurate quote unavailable for selected pair",
		},
		{
			name:       "no usable v3 fee tier",
			err:        domainerrors.BadRequest("no usable v3 fee tier found (amountIn=1, reasons=fee500:execution reverted)"),
			wantTreat:  true,
			wantReason: "selected token pair has no usable route on selected chain",
		},
		{
			name:       "amountOut less or equal zero",
			err:        domainerrors.BadRequest("amountOut <= 0 from router"),
			wantTreat:  true,
			wantReason: "selected token pair returned zero output for probed amounts",
		},
		{
			name:      "non-bad-request",
			err:       domainerrors.InternalServerError("rpc timeout"),
			wantTreat: false,
		},
		{
			name:      "plain error",
			err:       errors.New("network timeout"),
			wantTreat: false,
		},
		{
			name:      "nil",
			err:       nil,
			wantTreat: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotTreat, gotReason := shouldTreatQuoteProbeAsZero(tc.err)
			if gotTreat != tc.wantTreat {
				t.Fatalf("want treat=%v, got=%v", tc.wantTreat, gotTreat)
			}
			if gotReason != tc.wantReason {
				t.Fatalf("want reason=%q, got=%q", tc.wantReason, gotReason)
			}
		})
	}
}

func TestAnnotateCreatePaymentEstimateError(t *testing.T) {
	base := domainerrors.BadRequest("selected token pair has no usable route on selected chain")
	err := annotateCreatePaymentEstimateError("unable to estimate source amount from selected token to bridge token", base)

	appErr, ok := err.(*domainerrors.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.Status != 400 {
		t.Fatalf("expected status 400, got %d", appErr.Status)
	}
	if appErr.Code != base.Code {
		t.Fatalf("expected code %s, got %s", base.Code, appErr.Code)
	}
	if appErr.Message == "" {
		t.Fatalf("expected non-empty message")
	}
	if appErr.Message != "unable to estimate source amount from selected token to bridge token: selected token pair has no usable route on selected chain" {
		t.Fatalf("unexpected message: %s", appErr.Message)
	}
}
