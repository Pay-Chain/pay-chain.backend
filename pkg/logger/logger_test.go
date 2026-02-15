package logger

import (
	"context"
	"testing"
	"time"
)

func TestInitAndContextLogging(t *testing.T) {
	Init("development")
	if GetLogger() == nil {
		t.Fatal("expected logger initialized")
	}

	ctx := context.WithValue(context.Background(), "request_id", "req-1")
	l := WithContext(ctx)
	if l == nil {
		t.Fatal("expected contextual logger")
	}

	Info(ctx, "info")
	Debug(ctx, "debug")
	Warn(ctx, "warn")
	Error(ctx, "error")
	LogRequest(ctx, "GET", "/health", 200, 10*time.Millisecond, "127.0.0.1")
}

func TestWithContextNil(t *testing.T) {
	Init("development")
	if WithContext(nil) == nil {
		t.Fatal("expected base logger for nil context")
	}
}
