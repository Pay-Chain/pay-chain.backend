package logger

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
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

func TestWithContextTypedRequestID(t *testing.T) {
	Init("development")
	ctx := context.WithValue(context.Background(), RequestIDKey, "typed-req-id")
	if WithContext(ctx) == nil {
		t.Fatal("expected logger with typed request id context")
	}
}

func TestInit_ProductionAndWithContextWithoutFields(t *testing.T) {
	// reset package singleton to cover production init branch deterministically
	log = nil
	once = sync.Once{}

	Init("production")
	if GetLogger() == nil {
		t.Fatal("expected production logger initialized")
	}

	if WithContext(context.Background()) == nil {
		t.Fatal("expected logger without contextual fields")
	}
}

func TestInit_PanicWhenLoggerBuildFails(t *testing.T) {
	log = nil
	once = sync.Once{}
	origBuild := buildLogger
	t.Cleanup(func() {
		buildLogger = origBuild
		log = nil
		once = sync.Once{}
	})

	buildLogger = func(zap.Config) (*zap.Logger, error) {
		return nil, errors.New("build failed")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when logger builder fails")
		}
	}()
	Init("production")
}
