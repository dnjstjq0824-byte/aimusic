package service

import (
	"context"
	"errors"
	"testing"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

func TestExportDataEmptyUser(t *testing.T) {
	s := NewAuthService(&repository.Repos{}, &config.Config{})
	if _, err := s.ExportData(context.Background(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("empty userID must be invalid input, got %v", err)
	}
}
