package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
)

func validateInputs(mode string, hexLen int) error {
	if mode != "live" && mode != "test" {
		return fmt.Errorf("invalid mode: %s (allowed: live, test)", mode)
	}
	if hexLen <= 0 || hexLen%2 != 0 {
		return fmt.Errorf("invalid hex-len: %d (must be positive and even)", hexLen)
	}
	return nil
}

func buildCredentials(mode string, hexLen int) (string, string, error) {
	if err := validateInputs(mode, hexLen); err != nil {
		return "", "", err
	}

	apiKeyRaw, err := generateRandomHex(hexLen)
	if err != nil {
		return "", "", err
	}
	secretKeyRaw, err := generateRandomHex(hexLen)
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf("pk_%s_%s", mode, apiKeyRaw), fmt.Sprintf("sk_%s_%s", mode, secretKeyRaw), nil
}

func main() {
	mode := flag.String("mode", "live", "key mode: live or test")
	hexLen := flag.Int("hex-len", 32, "random hex length (must be even)")
	flag.Parse()

	apiKey, secretKey, err := buildCredentials(*mode, *hexLen)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Generated API credentials")
	fmt.Printf("API_KEY=%s\n", apiKey)
	fmt.Printf("SECRET_KEY=%s\n", secretKey)
}

func generateRandomHex(n int) (string, error) {
	b := make([]byte, n/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
