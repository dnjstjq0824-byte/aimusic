package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// P1: concurrent creates race past the existence pre-check and hit a unique
// violation (23505). The detector below maps those to 409s instead of 500s.

func TestIsUniqueViolation(t *testing.T) {
	dup := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
	if !isUniqueViolation(dup) {
		t.Fatalf("23505 not detected")
	}
	if !isUniqueViolation(fmt.Errorf("insert: %w", dup)) {
		t.Fatalf("wrapped 23505 not detected")
	}
	other := &pgconn.PgError{Code: "22P02", Message: "invalid input syntax"}
	if isUniqueViolation(other) {
		t.Fatalf("22P02 misdetected as unique violation")
	}
	if isUniqueViolation(errors.New("boom")) {
		t.Fatalf("plain error misdetected as unique violation")
	}
	if isUniqueViolation(nil) {
		t.Fatalf("nil misdetected as unique violation")
	}
}

func TestIsLicenseLimitExceeded(t *testing.T) {
	hit := &pgconn.PgError{Code: "P0001", Message: "store_license_limit_exceeded"}
	if !isLicenseLimitExceeded(hit) {
		t.Fatalf("license trigger not detected")
	}
	if !isLicenseLimitExceeded(fmt.Errorf("insert: %w", hit)) {
		t.Fatalf("wrapped license trigger not detected")
	}
	other := &pgconn.PgError{Code: "P0001", Message: "some other raise"}
	if isLicenseLimitExceeded(other) {
		t.Fatalf("unrelated P0001 misdetected as license trigger")
	}
	if isLicenseLimitExceeded(errors.New("boom")) {
		t.Fatalf("plain error misdetected as license trigger")
	}
	if isLicenseLimitExceeded(nil) {
		t.Fatalf("nil misdetected as license trigger")
	}
}
