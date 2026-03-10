package validator

import (
	"errors"
	"regexp"
)

// ValidatePassword validates the given password.
func ValidatePassword(password string) error {
	var (
		uppercase = regexp.MustCompile(`[A-Z]`)
		lowercase = regexp.MustCompile(`[a-z]`)
		number    = regexp.MustCompile(`[0-9]`)
		special   = regexp.MustCompile(`[!@#~$%^&*()_+=|<>?{}\[\]\-]`)
	)
	switch {
	case len(password) < 8:
		return errors.New("password must be at least 8 characters long")
	case !uppercase.MatchString(password):
		return errors.New("password must include at least one uppercase letter")
	case !lowercase.MatchString(password):
		return errors.New("password must include at least one lowercase letter")
	case !number.MatchString(password):
		return errors.New("password must include at least one number")
	case !special.MatchString(password):
		return errors.New("password must include at least one special character")
	default:
		return nil
	}
}
