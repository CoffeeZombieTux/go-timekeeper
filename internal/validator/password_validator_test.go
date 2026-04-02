package validator

import "testing"

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("Valid@123"); err != nil {
		t.Fatalf("expected valid password: %v", err)
	}

	cases := []string{
		"short",
		"nouppercase@1",
		"NOLOWERCASE@1",
		"NoNumber@",
		"NoSpecial1",
	}
	for _, c := range cases {
		if err := ValidatePassword(c); err == nil {
			t.Fatalf("expected validation error for %q", c)
		}
	}
}
