package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestGenerateRandomHex(t *testing.T) {
	v, err := generateRandomHex(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v) != 32 {
		t.Fatalf("expected len 32 got %d", len(v))
	}

	v2, err := generateRandomHex(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v2) != 2 {
		t.Fatalf("expected len 2 got %d", len(v2))
	}
}

func TestValidateInputs(t *testing.T) {
	if err := validateInputs("live", 32); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateInputs("bad", 32); err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if err := validateInputs("test", 3); err == nil {
		t.Fatal("expected error for odd hex len")
	}
}

func TestBuildCredentials(t *testing.T) {
	apiKey, secretKey, err := buildCredentials("test", 32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(apiKey, "pk_test_") {
		t.Fatalf("unexpected api key format: %s", apiKey)
	}
	if !strings.HasPrefix(secretKey, "sk_test_") {
		t.Fatalf("unexpected secret key format: %s", secretKey)
	}

	if _, _, err := buildCredentials("invalid", 32); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestBuildCredentials_RngFailureBranches(t *testing.T) {
	orig := randRead
	defer func() { randRead = orig }()

	// first random read fails
	randRead = func([]byte) (int, error) {
		return 0, errors.New("rng fail first")
	}
	_, _, err := buildCredentials("live", 32)
	if err == nil || !strings.Contains(err.Error(), "rng fail first") {
		t.Fatalf("expected first rng error, got: %v", err)
	}

	// second random read fails
	count := 0
	randRead = func(b []byte) (int, error) {
		count++
		if count == 2 {
			return 0, errors.New("rng fail second")
		}
		for i := range b {
			b[i] = byte(i + 1)
		}
		return len(b), nil
	}
	_, _, err = buildCredentials("test", 32)
	if err == nil || !strings.Contains(err.Error(), "rng fail second") {
		t.Fatalf("expected second rng error, got: %v", err)
	}
}

func TestMain_Success(t *testing.T) {
	origArgs := os.Args
	origStdout := os.Stdout
	origCommandLine := flag.CommandLine
	defer func() {
		os.Args = origArgs
		os.Stdout = origStdout
		flag.CommandLine = origCommandLine
	}()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"apikey-gen", "-mode", "test", "-hex-len", "16"}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	main()

	_ = w.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(r)
	if !strings.Contains(out.String(), "Generated API credentials") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestGenerateRandomHex_ReadError(t *testing.T) {
	orig := randRead
	defer func() { randRead = orig }()

	randRead = func([]byte) (int, error) {
		return 0, errors.New("rng error")
	}
	_, err := generateRandomHex(32)
	if err == nil {
		t.Fatal("expected rng error")
	}
}

func TestMain_ExitsOnInvalidInput(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_APIKEY_GEN_MAIN") == "1" {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"apikey-gen", "-mode", "invalid-mode", "-hex-len", "16"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMain_ExitsOnInvalidInput")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_APIKEY_GEN_MAIN=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected subprocess to exit with error")
	}
}
