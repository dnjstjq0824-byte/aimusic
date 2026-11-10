package main

import (
	"context"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

type fakeReapJobs struct {
	stuck  []*domain.TranscodingJob
	reset  map[string]bool
	failed map[string]string
}

func (f *fakeReapJobs) ListStuckRunning(_ context.Context, _ time.Time, _ int) ([]*domain.TranscodingJob, error) {
	return f.stuck, nil
}

func (f *fakeReapJobs) RequeueStuck(_ context.Context, id string) (bool, error) {
	if f.reset == nil {
		f.reset = map[string]bool{}
	}
	if f.reset[id] {
		return false, nil
	}
	f.reset[id] = true
	return true, nil
}

func (f *fakeReapJobs) MarkFailed(_ context.Context, id, msg string) error {
	if f.failed == nil {
		f.failed = map[string]string{}
	}
	f.failed[id] = msg
	return nil
}

type fakeReapTracks struct{ rejected map[string]string }

func (f *fakeReapTracks) UpdateStatus(_ context.Context, trackID, status string, reason *string) error {
	if f.rejected == nil {
		f.rejected = map[string]string{}
	}
	r := ""
	if reason != nil {
		r = *reason
	}
	f.rejected[trackID] = status + "|" + r
	return nil
}

type fakeReapQueue struct{ queued []repository.TranscodePayload }

func (f *fakeReapQueue) EnqueueTranscode(_ context.Context, p repository.TranscodePayload) error {
	f.queued = append(f.queued, p)
	return nil
}

func stuckJob(id string, attempts int) *domain.TranscodingJob {
	prefix := "hls/" + id
	return &domain.TranscodingJob{ID: id, TrackID: "track-" + id, Status: "RUNNING",
		InputObjectKey: "staging/" + id + ".wav", OutputPrefix: &prefix, Attempts: attempts}
}

func TestSweepRequeuesStuckJobs(t *testing.T) {
	j := &fakeReapJobs{stuck: []*domain.TranscodingJob{stuckJob("j-1", 0), stuckJob("j-2", 2)}}
	tr := &fakeReapTracks{}
	q := &fakeReapQueue{}
	if got := sweepStuckJobs(context.Background(), j, tr, q, "stage-bkt", 30*time.Minute); got != 2 {
		t.Fatalf("requeued = %d, want 2", got)
	}
	if len(q.queued) != 2 || q.queued[0].JobID != "j-1" || q.queued[0].InputBucket != "stage-bkt" {
		t.Fatalf("queued = %+v", q.queued)
	}
	if q.queued[1].OutputPrefix != "hls/j-2" || q.queued[1].TrackID != "track-j-2" {
		t.Fatalf("payload not rebuilt: %+v", q.queued[1])
	}
	// Second sweep: resets already taken, nothing requeued.
	q2 := &fakeReapQueue{}
	if got := sweepStuckJobs(context.Background(), j, tr, q2, "stage-bkt", 30*time.Minute); got != 0 {
		t.Fatalf("resweep = %d, want 0", got)
	}
}

func TestSweepFailsExhaustedJobs(t *testing.T) {
	j := &fakeReapJobs{stuck: []*domain.TranscodingJob{stuckJob("j-9", maxAttempts)}}
	tr := &fakeReapTracks{}
	q := &fakeReapQueue{}
	if got := sweepStuckJobs(context.Background(), j, tr, q, "stage-bkt", 30*time.Minute); got != 0 {
		t.Fatalf("requeued = %d, want 0", got)
	}
	if _, ok := j.failed["j-9"]; !ok {
		t.Fatalf("exhausted job not failed")
	}
	if tr.rejected["track-j-9"] == "" {
		t.Fatalf("track not rejected")
	}
	if len(q.queued) != 0 {
		t.Fatalf("exhausted job requeued")
	}
}

func TestSweepEmptyIsSilent(t *testing.T) {
	q := &fakeReapQueue{}
	if got := sweepStuckJobs(context.Background(), &fakeReapJobs{}, &fakeReapTracks{}, q, "stage-bkt", 30*time.Minute); got != 0 {
		t.Fatalf("requeued = %d", got)
	}
}
