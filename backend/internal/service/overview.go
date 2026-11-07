package service

import (
	"context"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

// AdminOverview is the admin console landing snapshot.
type AdminOverview struct {
	Users struct {
		Total    int64 `json:"total"`
		Admins   int64 `json:"admins"`
		Partners int64 `json:"partners"`
		Users    int64 `json:"users"`
	} `json:"users"`
	Tracks struct {
		Ready       int64 `json:"ready"`
		PendingGate int64 `json:"pendingGate"`
		TakenDown   int64 `json:"takenDown"`
	} `json:"tracks"`
	Reviews struct {
		Tracks        int64 `json:"tracks"`
		Partners      int64 `json:"partners"`
		Subscriptions int64 `json:"subscriptions"`
	} `json:"reviews"`
	DisputesOpen int64 `json:"disputesOpen"`
	Plays        struct {
		Last24h int64 `json:"last24h"`
		Total   int64 `json:"total"`
	} `json:"plays"`
	QueueDepth         *int64 `json:"queueDepth"`
	JobsFailed24h      int64  `json:"jobsFailed24h"`
	SettlementsPending struct {
		Count  int64 `json:"count"`
		NetKRW int64 `json:"netKrw"`
	} `json:"settlementsPending"`
}

// OverviewService aggregates platform counts for the admin console.
type OverviewService struct {
	repos *repository.Repos
}

// NewOverviewService builds the service.
func NewOverviewService(repos *repository.Repos) *OverviewService {
	return &OverviewService{repos: repos}
}

// Overview collects the admin snapshot. Queue depth is best-effort: a Redis
// outage yields null instead of failing the whole dashboard.
func (s *OverviewService) Overview(ctx context.Context) (*AdminOverview, error) {
	var out AdminOverview
	var err error
	r := s.repos
	if out.Users.Total, err = r.Users.CountAll(ctx); err != nil {
		return nil, err
	}
	if out.Users.Admins, err = r.Users.CountByRole(ctx, domain.RoleAdmin); err != nil {
		return nil, err
	}
	if out.Users.Partners, err = r.Users.CountByRole(ctx, domain.RolePartner); err != nil {
		return nil, err
	}
	if out.Users.Users, err = r.Users.CountByRole(ctx, domain.RoleUser); err != nil {
		return nil, err
	}
	if out.Tracks.Ready, err = r.Tracks.CountByStatus(ctx, domain.TrackReady); err != nil {
		return nil, err
	}
	if out.Tracks.PendingGate, err = r.Tracks.CountByStatus(ctx, domain.TrackLicenseGatePending); err != nil {
		return nil, err
	}
	if out.Tracks.TakenDown, err = r.Tracks.CountByStatus(ctx, domain.TrackTakenDown); err != nil {
		return nil, err
	}
	out.Reviews.Tracks = out.Tracks.PendingGate
	if out.Reviews.Partners, err = r.Partners.CountByStatus(ctx, domain.PartnerPending); err != nil {
		return nil, err
	}
	if out.Reviews.Subscriptions, err = r.Subscriptions.CountByStatus(ctx, "PENDING"); err != nil {
		return nil, err
	}
	if out.DisputesOpen, err = r.Disputes.CountOpen(ctx); err != nil {
		return nil, err
	}
	since := time.Now().Add(-24 * time.Hour)
	if out.Plays.Last24h, err = r.Streaming.CountCountedSince(ctx, since); err != nil {
		return nil, err
	}
	if out.Plays.Total, err = r.Streaming.CountCountedTotal(ctx); err != nil {
		return nil, err
	}
	if depth, err := r.Queue.QueueDepth(ctx); err == nil {
		out.QueueDepth = &depth
	}
	if out.JobsFailed24h, err = r.Jobs.CountFailedSince(ctx, since); err != nil {
		return nil, err
	}
	if out.SettlementsPending.Count, out.SettlementsPending.NetKRW, err = r.Settlements.PendingTotals(ctx); err != nil {
		return nil, err
	}
	return &out, nil
}
