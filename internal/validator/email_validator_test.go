package validator

import "testing"

func TestValidateEmail(t *testing.T) {
	if err := ValidateEmail("user@example.com"); err != nil {
		t.Fatalf("expected valid email: %v", err)
	}
	if err := ValidateEmail("broken-email"); err == nil {
		t.Fatal("expected invalid email error")
	}
}
