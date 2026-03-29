package usecases

import (
	"context"
	"os"
	"strings"

	"go.uber.org/zap"
	"payment-kita.backend/pkg/logger"
)

func createPaymentTraceEnabled() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("CREATE_PAYMENT_TRACE_ENABLED")))
	if raw == "" {
		return true
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func createPaymentTraceInfo(ctx context.Context, msg string, fields ...zap.Field) {
	if !createPaymentTraceEnabled() {
		return
	}
	logger.Info(ctx, msg, fields...)
}

func createPaymentTraceDebug(ctx context.Context, msg string, fields ...zap.Field) {
	if !createPaymentTraceEnabled() {
		return
	}
	logger.Debug(ctx, msg, fields...)
}

func createPaymentTraceWarn(ctx context.Context, msg string, fields ...zap.Field) {
	if !createPaymentTraceEnabled() {
		return
	}
	logger.Warn(ctx, msg, fields...)
}

