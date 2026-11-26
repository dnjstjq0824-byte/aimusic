package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

// Stuck-job reaper: a worker that dies mid-job leaves the job RUNNING with
// its Redis payload gone. Every JOB_REAP_INTERVAL_SECONDS (default 5 min)
// the reaper resets RUNNING jobs older than JOB_STUCK_AFTER_SECONDS
// (default 45 min) to QUEUED and re-enqueues them, up to maxAttempts —
// beyond that the job FAILs (and the transcode track is rejected like any
// exhausted job). Covers transcode and generation queues alike.

// reapJobs is the job surface the reaper needs (seam for tests).
type reapJobs interface {
	ListStuckRunning(ctx context.Context, cutoff time.Time, limit int) ([]*domain.TranscodingJob, error)
	RequeueStuck(ctx context.Context, id string) (bool, error)
	MarkFailed(ctx context.Context, id, errMsg string) error
}

// reapTracks rejects tracks whose jobs exhausted attempts.
type reapTracks interface {
	UpdateStatus(ctx context.Context, trackID, status string, reason *string) error
}

// reapQueue re-enqueues reset jobs.
type reapQueue interface {
	EnqueueTranscode(ctx context.Context, p repository.TranscodePayload) error
	EnqueueGeneration(ctx context.Context, p repository.GenerationPayload) error
}

// reapGenJobs is the generation-job surface the reaper needs.
type reapGenJobs interface {
	ListStuckRunning(ctx context.Context, cutoff time.Time, limit int) ([]*domain.GenerationJob, error)
	RequeueStuck(ctx context.Context, id string) (bool, error)
	MarkFailed(ctx context.Context, id, errMsg string) error
}

func jobReapConfig() (interval, stuckAfter time.Duration) {
	interval = 5 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("JOB_REAP_INTERVAL_SECONDS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			interval = time.Duration(n) * time.Second
		}
	}
	// 45 minutes exceeds the longest single stage (30-minute encode
	// timeout) with margin, so heartbeated live jobs are never reaped.
	stuckAfter = 45 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("JOB_STUCK_AFTER_SECONDS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			stuckAfter = time.Duration(n) * time.Second
		}
	}
	return interval, stuckAfter
}

func runJobReaper(ctx context.Context, repos *repository.Repos, stagingBucket string) {
	interval, stuckAfter := jobReapConfig()
	log.Printf("job reaper every %s (stuck after %s)", interval, stuckAfter)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			sweepStuckJobs(ctx, repos.Jobs, repos.Tracks, repos.Queue, stagingBucket, stuckAfter)
			sweepStuckGenerations(ctx, repos.GenJobs, repos.Queue, stuckAfter)
		}
	}
}

// sweepStuckJobs resets stale RUNNING jobs and re-enqueues them. Jobs at
// maxAttempts fail permanently. Returns the requeued count.
func sweepStuckJobs(ctx context.Context, jobs reapJobs, tracks reapTracks, queue reapQueue, stagingBucket string, stuckAfter time.Duration) int {
	stuck, err := jobs.ListStuckRunning(ctx, time.Now().Add(-stuckAfter), 50)
	if err != nil {
		log.Printf("job reaper: list: %v", err)
		return 0
	}
	done := 0
	for _, j := range stuck {
		if j.Attempts >= maxAttempts {
			msg := "worker died mid-transcode (attempts exhausted)"
			if err := jobs.MarkFailed(ctx, j.ID, msg); err != nil {
				log.Printf("job reaper: fail %s: %v", j.ID, err)
				continue
			}
			if err := tracks.UpdateStatus(ctx, j.TrackID, domain.TrackLicenseGateRejected, &msg); err != nil {
				log.Printf("job reaper: reject track %s: %v", j.TrackID, err)
			}
			continue
		}
		reset, err := jobs.RequeueStuck(ctx, j.ID)
		if err != nil || !reset {
			continue // finished or claimed concurrently; skip silently
		}
		prefix := ""
		if j.OutputPrefix != nil {
			prefix = *j.OutputPrefix
		}
		if err := queue.EnqueueTranscode(ctx, repository.TranscodePayload{
			JobID: j.ID, TrackID: j.TrackID,
			InputBucket: stagingBucket, InputKey: j.InputObjectKey,
			OutputPrefix: prefix,
		}); err != nil {
			log.Printf("job reaper: requeue %s: %v", j.ID, err)
			continue
		}
		done++
	}
	if done > 0 {
		log.Printf("job reaper: requeued %d stuck job(s)", done)
	}
	return done
}

// sweepStuckGenerations resets stale RUNNING generation jobs and re-enqueues
// them. Jobs at maxAttempts fail permanently. Returns the requeued count.
func sweepStuckGenerations(ctx context.Context, jobs reapGenJobs, queue reapQueue, stuckAfter time.Duration) int {
	stuck, err := jobs.ListStuckRunning(ctx, time.Now().Add(-stuckAfter), 50)
	if err != nil {
		log.Printf("job reaper: list generations: %v", err)
		return 0
	}
	done := 0
	for _, j := range stuck {
		if j.Attempts >= maxAttempts {
			if err := jobs.MarkFailed(ctx, j.ID, "worker died mid-generation (attempts exhausted)"); err != nil {
				log.Printf("job reaper: fail generation %s: %v", j.ID, err)
			}
			continue
		}
		reset, err := jobs.RequeueStuck(ctx, j.ID)
		if err != nil || !reset {
			continue // finished or claimed concurrently; skip silently
		}
		if err := queue.EnqueueGeneration(ctx, repository.GenerationPayload{JobID: j.ID}); err != nil {
			log.Printf("job reaper: requeue generation %s: %v", j.ID, err)
			continue
		}
		done++
	}
	if done > 0 {
		log.Printf("job reaper: requeued %d stuck generation job(s)", done)
	}
	return done
}
