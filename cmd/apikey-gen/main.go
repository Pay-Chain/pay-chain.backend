package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
)

func main() {
	mode := flag.String("mode", "live", "key mode: live or test")
	hexLen := flag.Int("hex-len", 32, "random hex length (must be even)")
	flag.Parse()

	if *mode != "live" && *mode != "test" {
		log.Fatalf("invalid mode: %s (allowed: live, test)", *mode)
	}
	if *hexLen <= 0 || *hexLen%2 != 0 {
		log.Fatalf("invalid hex-len: %d (must be positive and even)", *hexLen)
	}

	apiKeyRaw, err := generateRandomHex(*hexLen)
	if err != nil {
		log.Fatalf("failed to generate api key: %v", err)
	}
	secretKeyRaw, err := generateRandomHex(*hexLen)
	if err != nil {
		log.Fatalf("failed to generate secret key: %v", err)
	}

	apiKey := fmt.Sprintf("pk_%s_%s", *mode, apiKeyRaw)
	secretKey := fmt.Sprintf("sk_%s_%s", *mode, secretKeyRaw)

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

