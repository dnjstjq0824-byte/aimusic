// Package transcode provides transcoder resource watchdog monitoring,
// memory leak (RSS) caps, execution timeout termination, and OOM prevention guards.
package transcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrMemoryLimitExceeded = errors.New("watchdog aborted transcode: memory limit (RSS) exceeded")
	ErrExecutionTimeout    = errors.New("watchdog aborted transcode: max execution duration exceeded")
)

// WatchdogConfig defines resource enforcement limits.
type WatchdogConfig struct {
	MaxMemoryBytes int64         `json:"max_memory_bytes"` // Max RSS memory before abort (default: 512MB)
	MaxDuration    time.Duration `json:"max_duration"`     // Max duration per job (default: 60s)
	CheckInterval  time.Duration `json:"check_interval"`   // Watchdog polling period (default: 20ms)
}

// DefaultWatchdogConfig provides production container safety limits.
func DefaultWatchdogConfig() WatchdogConfig {
	return WatchdogConfig{
		MaxMemoryBytes: 512 * 1024 * 1024, // 512 MB
		MaxDuration:    60 * time.Second,
		CheckInterval:  20 * time.Millisecond,
	}
}

// ProcessMetrics reports instantaneous resource telemetry.
type ProcessMetrics struct {
	JobID           string        `json:"job_id"`
	PID             int           `json:"pid"`
	CurrentMemory   int64         `json:"current_memory_bytes"`
	PeakMemory      int64         `json:"peak_memory_bytes"`
	ElapsedDuration time.Duration `json:"elapsed_duration"`
	Aborted         bool          `json:"aborted"`
	AbortReason     string        `json:"abort_reason,omitempty"`
}

// ResourceWatchdog enforces memory caps and execution limits on worker jobs.
type ResourceWatchdog struct {
	cfg WatchdogConfig
	mu  sync.RWMutex
}

// NewResourceWatchdog creates a watchdog monitor.
func NewResourceWatchdog(cfg WatchdogConfig) *ResourceWatchdog {
	if cfg.MaxMemoryBytes <= 0 {
		cfg.MaxMemoryBytes = 512 * 1024 * 1024
	}
	if cfg.MaxDuration <= 0 {
		cfg.MaxDuration = 60 * time.Second
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 20 * time.Millisecond
	}
	return &ResourceWatchdog{cfg: cfg}
}

// MonitorExecution executes an encoding routine while enforcing watchdog resource constraints.
// memSampler is a function that returns current RSS memory bytes for the job.
func (rw *ResourceWatchdog) MonitorExecution(
	parentCtx context.Context,
	jobID string,
	memSampler func() int64,
	taskFn func(ctx context.Context) error,
) (*ProcessMetrics, error) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	start := time.Now()
	metrics := &ProcessMetrics{
		JobID: jobID,
	}

	done := make(chan error, 1)
	go func() {
		done <- taskFn(ctx)
	}()

	ticker := time.NewTicker(rw.cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			metrics.ElapsedDuration = time.Since(start)
			if err != nil {
				return metrics, err
			}
			return metrics, nil

		case <-parentCtx.Done():
			metrics.ElapsedDuration = time.Since(start)
			metrics.Aborted = true
			metrics.AbortReason = "parent context cancelled"
			return metrics, parentCtx.Err()

		case <-ticker.C:
			elapsed := time.Since(start)
			metrics.ElapsedDuration = elapsed

			if elapsed > rw.cfg.MaxDuration {
				metrics.Aborted = true
				metrics.AbortReason = fmt.Sprintf("execution exceeded max duration of %v", rw.cfg.MaxDuration)
				cancel()
				<-done // wait for task to terminate
				return metrics, ErrExecutionTimeout
			}

			if memSampler != nil {
				currMem := memSampler()
				metrics.CurrentMemory = currMem
				if currMem > metrics.PeakMemory {
					metrics.PeakMemory = currMem
				}

				if currMem > rw.cfg.MaxMemoryBytes {
					metrics.Aborted = true
					metrics.AbortReason = fmt.Sprintf("memory usage %d bytes exceeded limit of %d bytes", currMem, rw.cfg.MaxMemoryBytes)
					cancel()
					<-done
					return metrics, ErrMemoryLimitExceeded
				}
			}
		}
	}
}

// ToJSON serializes process metrics.
func (pm *ProcessMetrics) ToJSON() (string, error) {
	b, err := json.MarshalIndent(pm, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
