package main

import (
	"fmt"
	"log"
	"os"

	"pay-chain.backend/pkg/crypto"
)

var (
	printfFn       = fmt.Printf
	generateHashFn = generateHash
	fatalfFn       = log.Fatalf
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

	printfFn("Generating hash for password: %s\n", password)

	hash, err := generateHashFn(password)
	if err != nil {
		fatalfFn("Failed to hash password: %v", err)
	}

	printfFn("Bcrypt Hash: %s\n", hash)
}
