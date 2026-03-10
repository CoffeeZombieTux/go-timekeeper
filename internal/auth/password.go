package auth

import (
	"go-timekeeper/internal/apperror"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes the given password using bcrypt.
func HashPassword(password string) (string, error) {
	res, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// ComparePasswords compares the given password with the hashed password.
func ComparePasswords(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "Invalid password")
	}
	return nil
}
