package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestResolvePassword(t *testing.T) {
	if got := resolvePassword(nil); got != "The.Conqueror-45" {
		t.Fatalf("unexpected default password: %s", got)
	}
	if got := resolvePassword([]string{"abc"}); got != "abc" {
		t.Fatalf("unexpected arg password: %s", got)
	}
}

func TestGenerateHash(t *testing.T) {
	hash, err := generateHash("my-pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestMain_PrintsHash(t *testing.T) {
	origArgs := os.Args
	origStdout := os.Stdout
	origPrintf := printfFn
	origGenerate := generateHashFn
	origFatalf := fatalfFn
	defer func() {
		os.Args = origArgs
		os.Stdout = origStdout
		printfFn = origPrintf
		generateHashFn = origGenerate
		fatalfFn = origFatalf
	}()

	os.Args = []string{"hash-gen", "my-pass"}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	main()

	_ = w.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(r)
	text := out.String()
	if !strings.Contains(text, "Generating hash for password: my-pass") {
		t.Fatalf("unexpected output: %s", text)
	}
	if !strings.Contains(text, "Bcrypt Hash: ") {
		t.Fatalf("hash output missing: %s", text)
	}
}

func TestMain_DefaultPasswordAndFatalBranch(t *testing.T) {
	origArgs := os.Args
	origPrintf := printfFn
	origGenerate := generateHashFn
	origFatalf := fatalfFn
	defer func() {
		os.Args = origArgs
		printfFn = origPrintf
		generateHashFn = origGenerate
		fatalfFn = origFatalf
	}()

	var printed []string
	printfFn = func(format string, a ...interface{}) (int, error) {
		printed = append(printed, format)
		return 0, nil
	}

	type fatalSignal struct{}
	fatalfFn = func(string, ...interface{}) { panic(fatalSignal{}) }
	generateHashFn = func(string) (string, error) {
		return "", errors.New("boom")
	}

	os.Args = []string{"hash-gen"}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected fatal panic")
		}
	}()
	main()

	if len(printed) == 0 || !strings.Contains(printed[0], "Generating hash for password: %s") {
		t.Fatalf("unexpected print sequence: %#v", printed)
	}
}
