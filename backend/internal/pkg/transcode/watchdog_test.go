package transcode

import (
	"context"
	"testing"
	"time"
)

func TestResourceWatchdog_NormalTaskPass(t *testing.T) {
	cfg := WatchdogConfig{
		MaxMemoryBytes: 100 * 1024 * 1024, // 100MB
		MaxDuration:    2 * time.Second,
		CheckInterval:  10 * time.Millisecond,
	}
	wd := NewResourceWatchdog(cfg)

	metrics, err := wd.MonitorExecution(context.Background(), "job-normal", func() int64 {
		return 10 * 1024 * 1024 // 10MB steady
	}, func(ctx context.Context) error {
		time.Sleep(30 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.Aborted {
		t.Errorf("expected clean completion without abort")
	}
	if metrics.PeakMemory != 10*1024*1024 {
		t.Errorf("expected 10MB peak memory, got %d", metrics.PeakMemory)
	}
}

func TestResourceWatchdog_MemoryLimitAbort(t *testing.T) {
	cfg := WatchdogConfig{
		MaxMemoryBytes: 20 * 1024 * 1024, // 20MB limit
		MaxDuration:    2 * time.Second,
		CheckInterval:  10 * time.Millisecond,
	}
	wd := NewResourceWatchdog(cfg)

	memUsage := int64(5 * 1024 * 1024)

	metrics, err := wd.MonitorExecution(context.Background(), "job-oom-runaway", func() int64 {
		return memUsage
	}, func(ctx context.Context) error {
		// Simulate runaway memory leak after 30ms
		time.Sleep(30 * time.Millisecond)
		memUsage = 50 * 1024 * 1024 // Jumps to 50MB!

		// Wait until aborted by context
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return nil
		}
	})

	if err != ErrMemoryLimitExceeded {
		t.Fatalf("expected ErrMemoryLimitExceeded, got %v", err)
	}
	if !metrics.Aborted {
		t.Errorf("expected metrics.Aborted to be true")
	}
	t.Logf("Watchdog abort reason: %s", metrics.AbortReason)

	jsonStr, err := metrics.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}

func TestResourceWatchdog_TimeoutAbort(t *testing.T) {
	cfg := WatchdogConfig{
		MaxMemoryBytes: 100 * 1024 * 1024,
		MaxDuration:    50 * time.Millisecond, // 50ms timeout limit
		CheckInterval:  10 * time.Millisecond,
	}
	wd := NewResourceWatchdog(cfg)

	metrics, err := wd.MonitorExecution(context.Background(), "job-hanging-zombie", func() int64 {
		return 5 * 1024 * 1024
	}, func(ctx context.Context) error {
		// Hangs indefinitely until context cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			return nil
		}
	})

	if err != ErrExecutionTimeout {
		t.Fatalf("expected ErrExecutionTimeout, got %v", err)
	}
	if !metrics.Aborted {
		t.Errorf("expected metrics.Aborted to be true")
	}
}
