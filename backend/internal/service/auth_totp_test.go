package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
)

func TestCheckAge(t *testing.T) {
	if err := checkAge(nil, false); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("unconfirmed age err = %v", err)
	}
	if err := checkAge(nil, true); err != nil {
		t.Fatalf("confirmed without birth year: %v", err)
	}
	now := time.Now().Year()
	young := now - 14 // year-gap 14 still admits 13-year-olds: must fail closed
	if err := checkAge(&young, true); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("year-gap 14 err = %v", err)
	}
	old := now - 15
	if err := checkAge(&old, true); err != nil {
		t.Fatalf("year-gap 15: %v", err)
	}
	future := now + 1
	if err := checkAge(&future, true); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("future birth year err = %v", err)
	}
}

func TestEnrollTOTPRefusesWhenEnabled(t *testing.T) {
	s := testAuthService()
	s.getUser = func(_ context.Context, id string) (*domain.User, error) {
		return &domain.User{ID: id, Email: "e@x.io", TOTPEnabled: true, TOTPSecret: "LIVESECRET"}, nil
	}
	if _, err := s.EnrollTOTP(context.Background(), "u-1"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("re-enroll over live 2FA err = %v", err)
	}
}
