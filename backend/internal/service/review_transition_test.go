package service

import (
	"testing"

	"avibe-platform/backend/internal/domain"
)

// P1: approving any track to READY resurrects purged takedowns and publishes
// HLS-less uploads. Only transcode-complete review states may be approved.

func TestCanReviewApprove(t *testing.T) {
	for _, st := range []string{domain.TrackLicenseGatePending, domain.TrackLicenseGateRejected, domain.TrackReady} {
		if !canReviewApprove(st) {
			t.Fatalf("%s must be approvable", st)
		}
	}
	for _, st := range []string{domain.TrackUploaded, domain.TrackTranscoding, domain.TrackTakenDown, "", "READY "} {
		if canReviewApprove(st) {
			t.Fatalf("%q must not be approvable", st)
		}
	}
}

func TestCanReviewReject(t *testing.T) {
	if canReviewReject(domain.TrackTakenDown) {
		t.Fatalf("TAKEN_DOWN must not be rejectable (bytes already purged)")
	}
	for _, st := range []string{domain.TrackLicenseGatePending, domain.TrackLicenseGateRejected, domain.TrackReady, domain.TrackUploaded} {
		if !canReviewReject(st) {
			t.Fatalf("%s must be rejectable", st)
		}
	}
}
