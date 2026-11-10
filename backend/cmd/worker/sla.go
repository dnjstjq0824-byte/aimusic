package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

// Dispute SLA sweeper: §103 takedown requests must be acted on promptly.
// Every DISPUTE_SLA_INTERVAL_SECONDS (default 15 min) the worker finds
// OPEN/UNDER_REVIEW disputes older than DISPUTE_SLA_HOURS (default 72h),
// notifies every active admin once, and stamps the dispute so concurrent
// sweepers and later sweeps stay silent.

// slaDisputes is the dispute surface the sweeper needs (seam for tests).
type slaDisputes interface {
	ListSLABreached(ctx context.Context, slaHours int, limit int) ([]*domain.CopyrightDispute, error)
	StampSLANotified(ctx context.Context, id string) (bool, error)
}

// slaUsers resolves alert recipients.
type slaUsers interface {
	ListIDsByRole(ctx context.Context, role string) ([]string, error)
}

// slaNotify delivers one alert.
type slaNotify interface {
	Create(ctx context.Context, userID, kind, titleKo, bodyKo, linkPath string) error
}

func disputeSLAConfig() (interval time.Duration, hours int) {
	interval = 15 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("DISPUTE_SLA_INTERVAL_SECONDS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			interval = time.Duration(n) * time.Second
		}
	}
	hours = 72
	if raw := strings.TrimSpace(os.Getenv("DISPUTE_SLA_HOURS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			hours = n
		}
	}
	return interval, hours
}

func runDisputeSLASweeper(ctx context.Context, repos *repository.Repos) {
	interval, hours := disputeSLAConfig()
	log.Printf("dispute SLA sweeper every %s (breach after %dh un-notified)", interval, hours)
	// Sweep once at boot so deploys/restarts surface stale disputes now.
	sweepDisputeSLA(ctx, repos.Disputes, repos.Users, repos.Notify, hours)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			sweepDisputeSLA(ctx, repos.Disputes, repos.Users, repos.Notify, hours)
		}
	}
}

// sweepDisputeSLA alerts admins about breached disputes, stamping each exactly
// once. Stamp-first ordering (before fan-out) keeps concurrent sweepers from
// double-alerting; a failed fan-out leaves the stamp, so the dispute will not
// re-alert — operators see the stamp via the admin console.
func sweepDisputeSLA(ctx context.Context, disputes slaDisputes, users slaUsers, notify slaNotify, slaHours int) int {
	stale, err := disputes.ListSLABreached(ctx, slaHours, 50)
	if err != nil {
		log.Printf("dispute SLA sweep: list: %v", err)
		return 0
	}
	if len(stale) == 0 {
		return 0
	}
	admins, err := users.ListIDsByRole(ctx, domain.RoleAdmin)
	if err != nil {
		log.Printf("dispute SLA sweep: admins: %v", err)
		return 0
	}
	if len(admins) == 0 {
		log.Printf("dispute SLA sweep: %d breached but no active admins", len(stale))
		return 0
	}
	done := 0
	for _, d := range stale {
		stamped, err := disputes.StampSLANotified(ctx, d.ID)
		if err != nil || !stamped {
			continue // raced with another sweeper; skip silently
		}
		age := time.Since(d.CreatedAt).Truncate(time.Hour)
		title := "분쟁 SLA 초과: " + d.ID
		body := fmt.Sprintf("신고 %s(%s)가 %s째 미처리입니다. 트랙 %s", d.ID, d.Reason, age, d.TrackID)
		for _, adminID := range admins {
			if err := notify.Create(ctx, adminID, "NOTICE", title, body, "/admin/disputes"); err != nil {
				log.Printf("dispute SLA sweep: notify %s: %v", adminID, err)
			}
		}
		done++
	}
	if done > 0 {
		log.Printf("dispute SLA sweep: alerted %d dispute(s) to %d admin(s)", done, len(admins))
	}
	return done
}
