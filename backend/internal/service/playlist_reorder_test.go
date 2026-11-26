package service

import (
	"errors"
	"testing"

	"avibe-platform/backend/internal/domain"
)

func TestValidateReorderIDs(t *testing.T) {
	if err := validateReorderIDs([]string{"a", "b"}, []string{"b", "a"}); err != nil {
		t.Fatalf("permutation must pass: %v", err)
	}
	if err := validateReorderIDs([]string{"a", "b"}, []string{"a", "a"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("duplicate IDs must be rejected, got %v", err)
	}
	if err := validateReorderIDs([]string{"a", "b"}, []string{"a", "c"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("unknown ID must be rejected, got %v", err)
	}
	if err := validateReorderIDs([]string{"a", "b"}, []string{"a"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("short list must be rejected, got %v", err)
	}
	if err := validateReorderIDs(nil, nil); err != nil {
		t.Fatalf("empty reorder must pass: %v", err)
	}
}
