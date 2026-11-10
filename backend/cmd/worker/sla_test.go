package main

import (
	"context"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
)

type fakeSLADisputes struct {
	stale   []*domain.CopyrightDispute
	stamped map[string]bool
}

func (f *fakeSLADisputes) ListSLABreached(_ context.Context, _ int, _ int) ([]*domain.CopyrightDispute, error) {
	return f.stale, nil
}

func (f *fakeSLADisputes) StampSLANotified(_ context.Context, id string) (bool, error) {
	if f.stamped == nil {
		f.stamped = map[string]bool{}
	}
	if f.stamped[id] {
		return false, nil // another sweeper won the race
	}
	f.stamped[id] = true
	return true, nil
}

type fakeSLAUsers struct{ admins []string }

func (f *fakeSLAUsers) ListIDsByRole(_ context.Context, role string) ([]string, error) {
	if role != domain.RoleAdmin {
		return nil, nil
	}
	return f.admins, nil
}

type fakeSLANotify struct {
	sent []string // userID per alert
}

func (f *fakeSLANotify) Create(_ context.Context, userID, kind, _, _, _ string) error {
	if kind != "NOTICE" {
		panic("unexpected kind " + kind)
	}
	f.sent = append(f.sent, userID)
	return nil
}

func staleDispute(id string, age time.Duration) *domain.CopyrightDispute {
	return &domain.CopyrightDispute{ID: id, TrackID: "t-1", Reason: "VOICE_CLONE", Status: "OPEN", CreatedAt: time.Now().Add(-age)}
}

func TestSweepAlertsOncePerDispute(t *testing.T) {
	d := &fakeSLADisputes{stale: []*domain.CopyrightDispute{staleDispute("d-1", 80*time.Hour), staleDispute("d-2", 100*time.Hour)}}
	u := &fakeSLAUsers{admins: []string{"a-1", "a-2"}}
	n := &fakeSLANotify{}
	if got := sweepDisputeSLA(context.Background(), d, u, n, 72); got != 2 {
		t.Fatalf("swept = %d, want 2", got)
	}
	if len(n.sent) != 4 {
		t.Fatalf("alerts = %d, want 4 (2 disputes x 2 admins)", len(n.sent))
	}
	// Second sweep: stamps already taken, no re-alert.
	n2 := &fakeSLANotify{}
	if got := sweepDisputeSLA(context.Background(), d, u, n2, 72); got != 0 {
		t.Fatalf("resweep = %d, want 0", got)
	}
	if len(n2.sent) != 0 {
		t.Fatalf("re-alerts = %d, want 0", len(n2.sent))
	}
}

func TestSweepSilentWithoutStaleOrAdmins(t *testing.T) {
	u := &fakeSLAUsers{admins: []string{"a-1"}}
	n := &fakeSLANotify{}
	if got := sweepDisputeSLA(context.Background(), &fakeSLADisputes{}, u, n, 72); got != 0 {
		t.Fatalf("empty sweep = %d", got)
	}
	d := &fakeSLADisputes{stale: []*domain.CopyrightDispute{staleDispute("d-9", 80*time.Hour)}}
	if got := sweepDisputeSLA(context.Background(), d, &fakeSLAUsers{}, n, 72); got != 0 {
		t.Fatalf("no-admin sweep = %d", got)
	}
	if d.stamped["d-9"] {
		t.Fatalf("dispute stamped without recipients")
	}
}
