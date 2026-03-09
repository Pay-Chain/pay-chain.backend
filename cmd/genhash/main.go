package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func generatePasswordHash(password string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func main() {
	password := "AdminPaymentKita2026!"
	hash, _ := generatePasswordHash(password, 14)
	fmt.Println(hash)
}
