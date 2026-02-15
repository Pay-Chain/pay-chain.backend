package main

import (
	"fmt"
	"log"
	"os"

	"pay-chain.backend/pkg/crypto"
)

func resolvePassword(args []string) string {
	password := "The.Conqueror-45"
	if len(args) > 0 {
		return args[0]
	}
	return password
}

func generateHash(password string) (string, error) {
	return crypto.HashPassword(password)
}

func main() {
	password := resolvePassword(os.Args[1:])

	fmt.Printf("Generating hash for password: %s\n", password)

	hash, err := generateHash(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	fmt.Printf("Bcrypt Hash: %s\n", hash)
}
