package postgres

import (
	"database/sql"
	"errors"
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

func TestNewConnection_OpenAndPingHooks(t *testing.T) {
	origOpen := sqlOpen
	origPing := dbPing
	t.Cleanup(func() {
		sqlOpen = origOpen
		dbPing = origPing
	})

	cfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, User: "u", Password: "p", DBName: "d", SSLMode: "disable",
	}

	sqlOpen = func(_, _ string) (*sql.DB, error) {
		return nil, errors.New("open failed")
	}
	db, err := NewConnection(cfg)
	require.Error(t, err)
	require.Nil(t, db)
	require.Contains(t, err.Error(), "failed to open database")

	realDB, openErr := origOpen("postgres", "host=127.0.0.1 port=1 user=x password=x dbname=x sslmode=disable")
	require.NoError(t, openErr)
	t.Cleanup(func() { _ = realDB.Close() })
	sqlOpen = func(_, _ string) (*sql.DB, error) { return realDB, nil }
	dbPing = func(*sql.DB) error { return nil }

	db, err = NewConnection(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)
}
