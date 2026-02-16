package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestMainProcess_ExitsOnRedisInitFailure(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainProcess_ExitsOnRedisInitFailure")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"SERVER_ENV=development",
		"REDIS_URL=redis://127.0.0.1:0",
	)

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected helper process to exit with error")
	}
}

func TestMainProcess_ExitsOnInvalidServerPortAfterSetup(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "2" {
		main()
		return
	}

	redisSrv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis not available in this environment: %v", err)
	}
	defer redisSrv.Close()

	cmd := exec.Command(os.Args[0], "-test.run=TestMainProcess_ExitsOnInvalidServerPortAfterSetup")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=2",
		"SERVER_ENV=development",
		"SERVER_PORT=invalid-port",
		"REDIS_URL=redis://"+redisSrv.Addr(),
		// Force DB ping to fail quickly but allow boot to continue.
		"DB_HOST=127.0.0.1",
		"DB_PORT=1",
		"DB_USER=postgres",
		"DB_PASSWORD=postgres",
		"DB_NAME=paychain",
		"DB_SSLMODE=disable",
	)

	err = cmd.Run()
	if err == nil {
		t.Fatalf("expected helper process to exit with error on invalid port")
	}
}
