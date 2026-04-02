package api_model

import (
	"testing"
	"time"
)

func TestValidateRequestHelpers(t *testing.T) {
	r := RegisterRequest{Email: "user@example.com", Password: "Valid@123"}
	if err := r.ValidateRegisterRequest(); err != nil {
		t.Fatalf("register validation should pass: %v", err)
	}

	cp := ChangePasswordRequest{NewPassword: "Valid@123"}
	if err := cp.ValidateChangePasswordRequest(); err != nil {
		t.Fatalf("change password validation should pass: %v", err)
	}
}

func TestNewPaginationParams(t *testing.T) {
	p := NewPaginationParams(0, -10)
	if p.Limit <= 0 || p.Offset != 0 {
		t.Fatalf("unexpected pagination defaults: %+v", p)
	}
	p = NewPaginationParams(20000, 10)
	if p.Limit != 10000 {
		t.Fatalf("max limit clamp failed: %+v", p)
	}
}

func TestValidateTimeRangeParams(t *testing.T) {
	valid := &TimeRangeParams{
		FromDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
	}
	if err := valid.ValidateTimeRangeParams(); err != nil {
		t.Fatalf("expected valid range: %v", err)
	}

	invalid := &TimeRangeParams{
		FromDate: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
	}
	if err := invalid.ValidateTimeRangeParams(); err == nil {
		t.Fatal("expected invalid range error")
	}
}
