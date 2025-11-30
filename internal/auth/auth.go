package auth

import (
	"fmt"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("Need at least one character in the password")
	}

	hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)	
	if err != nil {
		return "", fmt.Errorf("Need at least one character in the password: error %w", err)
	}

	return hashedPassword, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, fmt.Errorf("Couild not compare the password and the hash: error %w", err)
	}

	return match, nil

}
