package service

import (
	"testing"

	"avibe-platform/backend/internal/domain"
)

// P0: recalculating a settlement must never rewrite a decided payout. Only
// PENDING rows may be recalculated; APPROVED/PAID/HELD/FORFEITED rows are
// rejected. Approving must never move a PAID row back to APPROVED.

func TestCanRecalculate(t *testing.T) {
	if !canRecalculate(domain.SettlePending) {
		t.Fatalf("PENDING must be recalculable")
	}
	for _, st := range []string{domain.SettleApproved, domain.SettlePaid, domain.SettleForfeited, domain.SettleHeld, "", "PENDING "} {
		if canRecalculate(st) {
			t.Fatalf("%q must not be recalculable", st)
		}
	}
}

func TestCanApprove(t *testing.T) {
	for _, st := range []string{domain.SettlePending, domain.SettleApproved} {
		if !canApprove(st) {
			t.Fatalf("%s must be approvable", st)
		}
	}
	for _, st := range []string{domain.SettlePaid, domain.SettleForfeited, domain.SettleHeld, ""} {
		if canApprove(st) {
			t.Fatalf("%s must not be approvable", st)
		}
	}
}
