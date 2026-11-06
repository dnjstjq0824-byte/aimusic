package service

import (
	"context"
	"time"

	"ai-music-platform/backend/internal/domain"
	"ai-music-platform/backend/internal/repository"
)

// ReviewService handles admin moderation queues.
type ReviewService struct {
	repos *repository.Repos
}

// PendingQueues is the admin review inbox.
type PendingQueues struct {
	LicenseGateTracks []*domain.Track                 `json:"licenseGateTracks"`
	Disputes          []*domain.CopyrightDispute      `json:"disputes"`
	Partners          []*domain.PartnerProfile        `json:"partners"`
	Subscriptions     []*domain.PartnerAISubscription `json:"subscriptions"`
}

// Pending collects items awaiting admin review.
func (s *ReviewService) Pending(ctx context.Context) (*PendingQueues, error) {
	tracks, err := s.repos.Tracks.ListByStatus(ctx, domain.TrackLicenseGatePending, 50, 0)
	if err != nil {
		return nil, err
	}
	disputes, err := s.repos.Disputes.ListOpen(ctx, 50, 0)
	if err != nil {
		return nil, err
	}
	partners, err := s.repos.Partners.ListByStatus(ctx, domain.PartnerPending, 50, 0)
	if err != nil {
		return nil, err
	}
	subs, err := s.repos.Subscriptions.ListPending(ctx, 50, 0)
	if err != nil {
		return nil, err
	}
	return &PendingQueues{
		LicenseGateTracks: tracks,
		Disputes:          disputes,
		Partners:          partners,
		Subscriptions:     subs,
	}, nil
}

// ReviewTrack approves a track to READY or rejects it with a Korean reason.
func (s *ReviewService) ReviewTrack(ctx context.Context, trackID string, approve bool, reason string) (*domain.Track, error) {
	if _, err := s.repos.Tracks.GetByID(ctx, trackID); err != nil {
		return nil, err
	}
	if approve {
		if err := s.repos.Tracks.UpdateStatus(ctx, trackID, domain.TrackReady, nil); err != nil {
			return nil, err
		}
	} else {
		if reason == "" {
			reason = "라이선스 검토 결과 배포가 거부되었습니다."
		}
		if err := s.repos.Tracks.UpdateStatus(ctx, trackID, domain.TrackLicenseGateRejected, &reason); err != nil {
			return nil, err
		}
	}
	return s.repos.Tracks.GetByID(ctx, trackID)
}

// ReviewPartner approves or suspends a partner profile.
func (s *ReviewService) ReviewPartner(ctx context.Context, partnerID string, approve bool) (*domain.PartnerProfile, error) {
	status := domain.PartnerSuspended
	if approve {
		status = domain.PartnerApproved
	}
	if err := s.repos.Partners.UpdateStatus(ctx, partnerID, status, approve); err != nil {
		return nil, err
	}
	return s.repos.Partners.GetByID(ctx, partnerID)
}

// ReviewSubscription verifies or rejects a subscription proof.
func (s *ReviewService) ReviewSubscription(ctx context.Context, subscriptionID, verifierID string, approve bool, expiresAt *time.Time) (*domain.PartnerAISubscription, error) {
	status := domain.SubRejected
	if approve {
		status = domain.SubVerified
	}
	if err := s.repos.Subscriptions.UpdateVerification(ctx, subscriptionID, status, verifierID, expiresAt); err != nil {
		return nil, err
	}
	return s.repos.Subscriptions.GetByID(ctx, subscriptionID)
}

// DisputeService handles reports, counter-claims, takedowns, and strikes.
type DisputeService struct {
	repos *repository.Repos
}

// Report files a copyright dispute on a track.
func (s *DisputeService) Report(ctx context.Context, trackID string, reporterID *string, reason, desc string) (*domain.CopyrightDispute, error) {
	if !domain.ValidDisputeReason(reason) {
		return nil, domain.ErrInvalidInput
	}
	if _, err := s.repos.Tracks.GetByID(ctx, trackID); err != nil {
		return nil, err
	}
	return s.repos.Disputes.Create(ctx, trackID, reporterID, reason, desc)
}

// Counter records a partner counter-claim; only the track owner may counter.
func (s *DisputeService) Counter(ctx context.Context, disputeID, callerUserID, note string) (*domain.CopyrightDispute, error) {
	d, err := s.repos.Disputes.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	track, err := s.repos.Tracks.GetByID(ctx, d.TrackID)
	if err != nil {
		return nil, err
	}
	profile, err := s.repos.Partners.GetByID(ctx, track.PartnerID)
	if err != nil {
		return nil, err
	}
	if profile.UserID != callerUserID {
		return nil, domain.ErrForbidden
	}
	if err := s.repos.Disputes.AppendCounter(ctx, disputeID, note); err != nil {
		return nil, err
	}
	return s.repos.Disputes.GetByID(ctx, disputeID)
}

// Resolve upholds (takedown + strike) or dismisses a dispute.
func (s *DisputeService) Resolve(ctx context.Context, disputeID, resolverID string, uphold bool) (*domain.CopyrightDispute, error) {
	d, err := s.repos.Disputes.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	if !uphold {
		if err := s.repos.Disputes.Resolve(ctx, disputeID, domain.DisputeDismissed, resolverID, false); err != nil {
			return nil, err
		}
		return s.repos.Disputes.GetByID(ctx, disputeID)
	}
	track, err := s.repos.Tracks.GetByID(ctx, d.TrackID)
	if err != nil {
		return nil, err
	}
	if err := s.repos.Tracks.UpdateStatus(ctx, track.ID, domain.TrackTakenDown, nil); err != nil {
		return nil, err
	}
	if err := s.repos.Disputes.Resolve(ctx, disputeID, domain.DisputeUpheld, resolverID, true); err != nil {
		return nil, err
	}
	profile, err := s.repos.Partners.GetByID(ctx, track.PartnerID)
	if err != nil {
		return nil, err
	}
	strikes, err := s.repos.Blocks.CountStrikes(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}
	next := strikes + 1
	if next > 3 {
		next = 3
	}
	var expires *time.Time
	now := time.Now()
	switch next {
	case 1:
		e := now.Add(7 * 24 * time.Hour)
		expires = &e
	case 2:
		e := now.Add(30 * 24 * time.Hour)
		expires = &e
	case 3:
		if err := s.repos.Partners.UpdateStatus(ctx, profile.ID, domain.PartnerSuspended, false); err != nil {
			return nil, err
		}
	}
	if _, err := s.repos.Blocks.Create(ctx, profile.UserID, &resolverID, "copyright strike "+d.Reason, next, expires); err != nil {
		return nil, err
	}
	return s.repos.Disputes.GetByID(ctx, disputeID)
}

// Takedown immediately takes down a track.
func (s *DisputeService) Takedown(ctx context.Context, trackID, reason string) (*domain.Track, error) {
	if _, err := s.repos.Tracks.GetByID(ctx, trackID); err != nil {
		return nil, err
	}
	if err := s.repos.Tracks.UpdateStatus(ctx, trackID, domain.TrackTakenDown, &reason); err != nil {
		return nil, err
	}
	return s.repos.Tracks.GetByID(ctx, trackID)
}
