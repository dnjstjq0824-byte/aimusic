package transcode

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type mockS3Client struct {
	mu       sync.Mutex
	uploaded map[string][]byte
	delay    time.Duration
}

func (m *mockS3Client) Upload(ctx context.Context, key, contentType string, data []byte) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	m.uploaded[key] = data
	m.mu.Unlock()
	return nil
}

func TestS3ParallelOptimizer_BulkSegmentsLatencyReduction(t *testing.T) {
	mockS3 := &mockS3Client{
		uploaded: make(map[string][]byte),
		delay:    5 * time.Millisecond, // 5ms simulated network per upload
	}

	optimizer := NewS3ParallelOptimizer(mockS3, 10) // 10 parallel workers

	// Create 50 segment tasks
	tasks := make([]UploadTask, 50)
	for i := 0; i < 50; i++ {
		tasks[i] = UploadTask{
			Key:  fmt.Sprintf("hls/track-001/stream0/seg-%04d.ts", i),
			Data: []byte("mock-ts-packet-data-188-bytes"),
		}
	}

	start := time.Now()
	res, err := optimizer.UploadBatch(context.Background(), tasks)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("UploadBatch failed: %v", err)
	}

	if res.SucceededFiles != 50 {
		t.Fatalf("expected 50 succeeded files, got %d", res.SucceededFiles)
	}
	if res.FailedFiles != 0 {
		t.Errorf("expected 0 failed files, got %d", res.FailedFiles)
	}

	// Sequential theoretical time: 50 * 5ms = 250ms
	// Parallel time with 10 workers: 50/10 * 5ms = ~25ms
	// Latency reduction should be >= 70%
	if res.LatencyReduction < 70.0 {
		t.Errorf("DoD violated: expected >= 70%% latency reduction, got %f%%", res.LatencyReduction)
	}
	t.Logf("Uploaded 50 segments in %v with %f%% latency reduction", elapsed, res.LatencyReduction)
}

func TestDetectContentType(t *testing.T) {
	cases := []struct {
		key      string
		expected string
	}{
		{"index.m3u8", "application/vnd.apple.mpegurl"},
		{"segment0.ts", "video/MP2T"},
		{"audio.m4s", "video/iso.segment"},
		{"enc.key", "application/octet-stream"},
	}

	for _, c := range cases {
		ct := DetectContentType(c.key)
		if ct != c.expected {
			t.Errorf("expected %s for %s, got %s", c.expected, c.key, ct)
		}
	}
}
