package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := os.Getenv("CONVERGE_JWT_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}

	userID := os.Args[1]
	displayName := ""
	if len(os.Args) > 2 {
		displayName = os.Args[2]
	} else {
		displayName = userID
	}

	claims := jwt.MapClaims{
		"user_id":      userID,
		"display_name": displayName,
		"sub":          userID,
		"iat":          time.Now().Unix(),
		"exp":          time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(tokenString)
}
