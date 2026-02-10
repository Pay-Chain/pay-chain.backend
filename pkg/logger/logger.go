package logger

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log  *zap.Logger
	once sync.Once
	atom zap.AtomicLevel
)

type ContextKey string

const (
	RequestIDKey     ContextKey = "request_id"
	CorrelationIDKey ContextKey = "correlation_id"
)

// Init initializes the logger
func Init(env string) {
	once.Do(func() {
		config := zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		if env == "development" {
			config = zap.NewDevelopmentConfig()
			config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}

		var err error
		log, err = config.Build(zap.AddCallerSkip(1))
		if err != nil {
			panic(err)
		}
		atom = config.Level
	})
}

// GetLogger returns the underlying zap logger
func GetLogger() *zap.Logger {
	return log
}

// WithContext adds context fields (request_id) to the logger
func WithContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return log
	}

	var fields []zap.Field
	if reqID, ok := ctx.Value("request_id").(string); ok { // Using string key for compatibility with Gin
		fields = append(fields, zap.String("request_id", reqID))
	}
	// Also check for our typed key if used elsewhere
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		fields = append(fields, zap.String("request_id", reqID))
	}

	if len(fields) > 0 {
		return log.With(fields...)
	}
	return log
}

// Info logs a message at InfoLevel
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Info(msg, fields...)
}

// Error logs a message at ErrorLevel
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Error(msg, fields...)
}

// Debug logs a message at DebugLevel
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Debug(msg, fields...)
}

// Warn logs a message at WarnLevel
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Warn(msg, fields...)
}

// LogRequest logs an HTTP request details
func LogRequest(ctx context.Context, method, path string, status int, latency time.Duration, clientIP string) {
	WithContext(ctx).Info("HTTP Request",
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status", status),
		zap.Duration("latency", latency),
		zap.String("client_ip", clientIP),
	)
}
