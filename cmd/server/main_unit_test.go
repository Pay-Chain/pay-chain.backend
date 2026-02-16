package main

import (
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"pay-chain.backend/internal/config"
	plog "pay-chain.backend/pkg/logger"
	"pay-chain.backend/pkg/redis"
)

func withMainHooks(t *testing.T) {
	t.Helper()
	origLoadDotenv := loadDotenv
	origLoadCfg := loadCfg
	origInitLog := initLog
	origInitRedis := initRedis
	origOpenDB := openDB
	origNewSessionStore := newSessionStore
	origRunServer := runServer

	t.Cleanup(func() {
		loadDotenv = origLoadDotenv
		loadCfg = origLoadCfg
		initLog = origInitLog
		initRedis = origInitRedis
		openDB = origOpenDB
		newSessionStore = origNewSessionStore
		runServer = origRunServer
	})
}

func baseTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: "18080",
			Env:  "development",
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			DBName:   "paychain",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			URL:      "redis://localhost:6379",
			PASSWORD: "",
		},
		JWT: config.JWTConfig{
			Secret:        "secret",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		},
		Blockchain: config.BlockchainConfig{
			OwnerPrivateKey: "",
		},
		Security: config.SecurityConfig{
			ApiKeyEncryptionKey:  "0000000000000000000000000000000000000000000000000000000000000000",
			SessionEncryptionKey: "0000000000000000000000000000000000000000000000000000000000000000",
		},
	}
}

func TestRunMainProcess_RedisInitError(t *testing.T) {
	withMainHooks(t)

	loadDotenv = func(...string) error { return nil }
	loadCfg = baseTestConfig
	initLog = plog.Init
	initRedis = func(string, string) error { return errors.New("redis down") }

	err := runMainProcess()
	if err == nil {
		t.Fatal("expected redis init error")
	}
}

func TestRunMainProcess_DBOpenError(t *testing.T) {
	withMainHooks(t)

	loadDotenv = func(...string) error { return nil }
	loadCfg = baseTestConfig
	initLog = plog.Init
	initRedis = func(string, string) error { return nil }
	openDB = func(string) (*gorm.DB, error) { return nil, errors.New("db open failed") }

	err := runMainProcess()
	if err == nil {
		t.Fatal("expected db open error")
	}
}

func TestRunMainProcess_SessionStoreError(t *testing.T) {
	withMainHooks(t)

	loadDotenv = func(...string) error { return nil }
	loadCfg = baseTestConfig
	initLog = plog.Init
	initRedis = func(string, string) error { return nil }
	openDB = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file:main_session_err?mode=memory&cache=shared"), &gorm.Config{})
	}
	newSessionStore = func(string) (*redis.SessionStore, error) { return nil, errors.New("bad session key") }

	err := runMainProcess()
	if err == nil {
		t.Fatal("expected session store error")
	}
}

func TestRunMainProcess_ServerRunError(t *testing.T) {
	withMainHooks(t)

	loadDotenv = func(...string) error { return nil }
	loadCfg = baseTestConfig
	initLog = plog.Init
	initRedis = func(string, string) error { return nil }
	openDB = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file:main_server_err?mode=memory&cache=shared"), &gorm.Config{})
	}
	newSessionStore = redis.NewSessionStore
	runServer = func(*gin.Engine, string) error { return errors.New("listen failed") }

	err := runMainProcess()
	if err == nil {
		t.Fatal("expected server run error")
	}
}

func TestRunMainProcess_SuccessPath(t *testing.T) {
	withMainHooks(t)

	loadDotenv = func(...string) error { return nil }
	loadCfg = baseTestConfig
	initLog = plog.Init
	initRedis = func(string, string) error { return nil }
	openDB = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file:main_success?mode=memory&cache=shared"), &gorm.Config{})
	}
	newSessionStore = redis.NewSessionStore
	runServer = func(*gin.Engine, string) error { return nil }

	if err := runMainProcess(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
