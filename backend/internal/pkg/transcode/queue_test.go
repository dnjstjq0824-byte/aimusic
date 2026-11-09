package transcode

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestParallelQueue_EnqueueAndValidation(t *testing.T) {
	pq := NewParallelQueue(4, 10)
	defer pq.Close()

	// Invalid job
	err := pq.Enqueue(&TranscodeJob{})
	if err != ErrInvalidJob {
		t.Fatalf("expected ErrInvalidJob, got %v", err)
	}

	job := &TranscodeJob{
		ID:          "job-001",
		TrackID:     "track-001",
		SourceURI:   "s3://bucket/audio.wav",
		DurationSec: 180.0,
		Priority:    PriorityHigh,
	}

	err = pq.Enqueue(job)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Profiles should have defaulted to 5 profiles
	if len(job.Profiles) != 5 {
		t.Errorf("expected 5 default profiles, got %d", len(job.Profiles))
	}
}

func TestExecuteParallelProfiles_5ProfilesUnder8Seconds(t *testing.T) {
	job := &TranscodeJob{
		ID:          "job-test-bench",
		TrackID:     "track-test-bench",
		SourceURI:   "s3://bucket/test.wav",
		DurationSec: 180.0, // 3-minute audio
		Profiles:    DefaultTranscodeProfiles(),
	}

	// Mock high-performance encoding worker taking ~50ms per profile
	mockEncoder := func(ctx context.Context, j *TranscodeJob, p TranscodeProfile) (ProfileResult, error) {
		time.Sleep(50 * time.Millisecond) // Simulated encoding latency
		return ProfileResult{
			ProfileName: p.Name,
			Duration:    50 * time.Millisecond,
			OutputURI:   fmt.Sprintf("s3://bucket/hls/%s/%s.m3u8", j.TrackID, p.Name),
			ByteSize:    1024 * 1024 * 2,
			Success:     true,
		}, nil
	}

	start := time.Now()
	res := ExecuteParallelProfiles(context.Background(), job, mockEncoder)
	elapsed := time.Since(start)

	// Verification check against Definition of Done: < 8.0 seconds
	if elapsed >= 8*time.Second {
		t.Fatalf("DoD violated: parallel transcode took %v (limit: 8.0s)", elapsed)
	}
	t.Logf("Parallel 5-profile transcode completed in %v (< 8.0s limit)", elapsed)

	if !res.AllSucceeded {
		t.Errorf("expected all 5 profiles to succeed")
	}
	if len(res.ProfileResults) != 5 {
		t.Errorf("expected 5 profile results, got %d", len(res.ProfileResults))
	}

	// Ensure each profile is accounted for
	expectedNames := []string{"opus_96k", "opus_160k", "opus_256k", "aac_128k", "aac_320k"}
	for _, name := range expectedNames {
		r, ok := res.ProfileResults[name]
		if !ok || !r.Success {
			t.Errorf("profile %s missing or failed", name)
		}
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}
