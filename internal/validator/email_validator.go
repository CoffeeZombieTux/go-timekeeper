package validator

import (
	"net/mail"
)

// ValidateEmail validates the given email address.
func ValidateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	return err
}
