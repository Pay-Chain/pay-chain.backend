package main

import (
	"bytes"
	"flag"
	"os"
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
