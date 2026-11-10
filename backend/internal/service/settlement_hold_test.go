package service

import (
	"context"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
)

// P2-2 (#12): confirmed-fraud settlement hold. Only a PENDING settlement
// can be held; only a HELD one released. Approve/pay never move a HELD row.

func TestSettlementHoldGuards(t *testing.T) {
	if !canHold(domain.SettlePending) {
		t.Fatalf("PENDING must be holdable")
	}
	for _, st := range []string{domain.SettleApproved, domain.SettlePaid, domain.SettleForfeited, domain.SettleHeld} {
		if canHold(st) {
			t.Fatalf("%s must not be holdable", st)
		}
	}
	if !canRelease(domain.SettleHeld) {
		t.Fatalf("HELD must be releasable")
	}
	for _, st := range []string{domain.SettlePending, domain.SettleApproved, domain.SettlePaid, domain.SettleForfeited} {
		if canRelease(st) {
			t.Fatalf("%s must not be releasable", st)
		}
	}
}

func TestSettlementHoldRejectsEmptyReason(t *testing.T) {
	s := &SettlementService{}
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	// Reason is validated before any repository use, so nil repos suffice.
	if _, err := s.Hold(context.Background(), "p", start, end, "  ", "admin"); err == nil {
		t.Fatalf("empty hold reason must be rejected")
	}
	if _, err := s.Hold(context.Background(), "", start, end, "fraud confirmed", "admin"); err == nil {
		t.Fatalf("empty partner must be rejected")
	}
}
