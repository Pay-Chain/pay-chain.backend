package main

import (
	"fmt"
	"log"
	"os"

	"pay-chain.backend/pkg/crypto"
)

func main() {
	// Default password
	password := "The.Conqueror-45"

	// Check if argument provided
	if len(os.Args) > 1 {
		password = os.Args[1]
	}

	fmt.Printf("Generating hash for password: %s\n", password)

	hash, err := crypto.HashPassword(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	fmt.Printf("Bcrypt Hash: %s\n", hash)
}
