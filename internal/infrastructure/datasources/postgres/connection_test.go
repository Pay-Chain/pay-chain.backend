package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/config"
)

func TestNewConnection_PingFailure(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     1,
		User:     "x",
		Password: "x",
		DBName:   "x",
		SSLMode:  "disable",
	}

	db, err := NewConnection(cfg)
	require.Error(t, err)
	require.Nil(t, db)
	require.Contains(t, err.Error(), "failed to ping database")
}
