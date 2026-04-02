package auth

import "testing"

func TestHashAndComparePassword(t *testing.T) {
	hash, err := HashPassword("Valid@123")
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	if err := ComparePasswords(hash, "Valid@123"); err != nil {
		t.Fatalf("compare should succeed: %v", err)
	}
	if err := ComparePasswords(hash, "Wrong@123"); err == nil {
		t.Fatal("compare should fail for wrong password")
	}
}
