package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "AdminPayChain2026!"
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), 14)
	fmt.Println(string(bytes))
}
