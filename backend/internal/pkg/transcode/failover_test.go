package transcode

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFailoverEngine_PrimarySuccess(t *testing.T) {
	fe := NewFailoverEngine()
	job := &TranscodeJob{ID: "job-primary-ok"}

	primary := TranscodeProfile{Name: "opus_256k", Codec: "opus"}
	plan := DefaultFailoverPlan(primary)

	res, report, err := fe.Execute(context.Background(), job, plan, func(ctx context.Context, p TranscodeProfile) (ProfileResult, error) {
		return ProfileResult{
			ProfileName: p.Name,
			Success:     true,
			OutputURI:   "s3://bucket/opus_256k.m3u8",
		}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success || report.FallbackOccurred {
		t.Errorf("expected clean primary execution without fallback")
	}
	if report.FinalProfile != "opus_256k" {
		t.Errorf("expected final profile opus_256k, got %s", report.FinalProfile)
	}
}

func TestFailoverEngine_CascadeToAAC(t *testing.T) {
	fe := NewFailoverEngine()
	job := &TranscodeJob{ID: "job-fallback-test"}

	primary := TranscodeProfile{Name: "opus_256k", Codec: "opus"}
	plan := DefaultFailoverPlan(primary)

	// Simulate failure on all Opus profiles, but success on AAC
	mockEncoder := func(ctx context.Context, p TranscodeProfile) (ProfileResult, error) {
		if p.Codec == "opus" {
			return ProfileResult{}, errors.New("simulated libopus encoder fatal error")
		}
		// AAC succeeds!
		return ProfileResult{
			ProfileName: p.Name,
			Success:     true,
			OutputURI:   "s3://bucket/aac_128k.m3u8",
		}, nil
	}

	res, report, err := fe.Execute(context.Background(), job, plan, mockEncoder)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected result to be successful")
	}
	if !report.FallbackOccurred {
		t.Errorf("expected FallbackOccurred to be true")
	}
	if report.ResolvedCodec != "aac" {
		t.Errorf("expected ResolvedCodec to be 'aac', got %s", report.ResolvedCodec)
	}
	t.Logf("Fallback history: %v", report.AttemptHistory)

	jsonStr, err := report.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}

func TestCircuitBreaker_TripsAndCooldown(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)

	if !cb.AllowExecution() {
		t.Fatalf("fresh circuit breaker should allow execution")
	}

	// 1st failure
	cb.RecordFailure()
	if !cb.AllowExecution() {
		t.Fatalf("1st failure should not trip breaker")
	}

	// 2nd failure -> trips to OPEN
	cb.RecordFailure()
	if cb.AllowExecution() {
		t.Fatalf("2nd failure should trip circuit to OPEN")
	}

	// Wait cooldown (50ms)
	time.Sleep(60 * time.Millisecond)

	// Now should transition to HALF_OPEN and allow probe
	if !cb.AllowExecution() {
		t.Fatalf("after cooldown, should allow execution in HALF_OPEN state")
	}

	// Probe succeeds -> resets to CLOSED
	cb.RecordSuccess()
	if !cb.AllowExecution() {
		t.Fatalf("success should reset circuit to CLOSED")
	}
}
