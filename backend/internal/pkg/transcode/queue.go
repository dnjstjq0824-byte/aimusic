// Package transcode provides high-speed parallel audio transcoding job distribution,
// multi-profile concurrent encoding pipelines, and worker pool scaling.
package transcode

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var (
	ErrQueueClosed = errors.New("transcode queue is closed")
	ErrJobTimeout  = errors.New("transcode job execution timed out")
	ErrInvalidJob  = errors.New("invalid transcode job specification")
)

// JobPriority defines the execution priority level.
type JobPriority int

const (
	PriorityLow      JobPriority = 0
	PriorityStandard JobPriority = 1
	PriorityHigh     JobPriority = 2
	PriorityUrgent   JobPriority = 3
)

// TranscodeProfile defines an encoding target profile.
type TranscodeProfile struct {
	Name        string `json:"name"`  // e.g. "opus_96k", "opus_160k", "opus_256k", "aac_128k", "aac_320k"
	Codec       string `json:"codec"` // "opus", "aac", "alac", "flac"
	BitrateKbps int    `json:"bitrate_kbps"`
	SampleRate  int    `json:"sample_rate"` // e.g. 48000
	Channels    int    `json:"channels"`    // e.g. 2
}

// DefaultTranscodeProfiles returns standard 5-target parallel distribution profiles.
func DefaultTranscodeProfiles() []TranscodeProfile {
	return []TranscodeProfile{
		{Name: "opus_96k", Codec: "opus", BitrateKbps: 96, SampleRate: 48000, Channels: 2},
		{Name: "opus_160k", Codec: "opus", BitrateKbps: 160, SampleRate: 48000, Channels: 2},
		{Name: "opus_256k", Codec: "opus", BitrateKbps: 256, SampleRate: 48000, Channels: 2},
		{Name: "aac_128k", Codec: "aac", BitrateKbps: 128, SampleRate: 48000, Channels: 2},
		{Name: "aac_320k", Codec: "aac", BitrateKbps: 320, SampleRate: 48000, Channels: 2},
	}
}

// TranscodeJob represents a discrete audio transcoding task.
type TranscodeJob struct {
	ID          string             `json:"id"`
	TrackID     string             `json:"track_id"`
	SourceURI   string             `json:"source_uri"`
	DurationSec float64            `json:"duration_sec"`
	Priority    JobPriority        `json:"priority"`
	Profiles    []TranscodeProfile `json:"profiles"`
	CreatedAt   time.Time          `json:"created_at"`
}

// ProfileResult represents completion status for an individual profile encoding.
type ProfileResult struct {
	ProfileName string        `json:"profile_name"`
	Duration    time.Duration `json:"duration"`
	OutputURI   string        `json:"output_uri"`
	ByteSize    int64         `json:"byte_size"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
}

// JobResult represents the overall multi-profile transcoding result.
type JobResult struct {
	JobID          string                   `json:"job_id"`
	TrackID        string                   `json:"track_id"`
	TotalDuration  time.Duration            `json:"total_duration"`
	ProfileResults map[string]ProfileResult `json:"profile_results"`
	AllSucceeded   bool                     `json:"all_succeeded"`
}

// ParallelQueue distributes transcoding jobs across concurrent worker pools.
type ParallelQueue struct {
	mu         sync.RWMutex
	jobs       chan *TranscodeJob
	workers    int
	closed     bool
	workerPool sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewParallelQueue creates a parallel queue with specified worker capacity.
func NewParallelQueue(workers int, queueCapacity int) *ParallelQueue {
	if workers <= 0 {
		workers = 4
	}
	if queueCapacity <= 0 {
		queueCapacity = 100
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &ParallelQueue{
		jobs:    make(chan *TranscodeJob, queueCapacity),
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Enqueue submits a job to the distribution queue.
func (pq *ParallelQueue) Enqueue(job *TranscodeJob) error {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.closed {
		return ErrQueueClosed
	}
	if job == nil || job.ID == "" || job.TrackID == "" {
		return ErrInvalidJob
	}

	if len(job.Profiles) == 0 {
		job.Profiles = DefaultTranscodeProfiles()
	}

	pq.jobs <- job
	return nil
}

// ExecuteParallelProfiles encodes all profiles of a job concurrently.
func ExecuteParallelProfiles(ctx context.Context, job *TranscodeJob, encoderFn func(ctx context.Context, job *TranscodeJob, p TranscodeProfile) (ProfileResult, error)) *JobResult {
	start := time.Now()
	profiles := job.Profiles
	numProfiles := len(profiles)

	results := make(map[string]ProfileResult, numProfiles)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, p := range profiles {
		wg.Add(1)
		go func(prof TranscodeProfile) {
			defer wg.Done()
			res, err := encoderFn(ctx, job, prof)
			mu.Lock()
			if err != nil {
				res.Success = false
				res.Error = err.Error()
			}
			results[prof.Name] = res
			mu.Unlock()
		}(p)
	}

	wg.Wait()

	allSucceeded := true
	for _, r := range results {
		if !r.Success {
			allSucceeded = false
			break
		}
	}

	return &JobResult{
		JobID:          job.ID,
		TrackID:        job.TrackID,
		TotalDuration:  time.Since(start),
		ProfileResults: results,
		AllSucceeded:   allSucceeded,
	}
}

// Close gracefully closes the queue.
func (pq *ParallelQueue) Close() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if !pq.closed {
		pq.closed = true
		pq.cancel()
		close(pq.jobs)
	}
}

// ToJSON serializes job result.
func (r *JobResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
