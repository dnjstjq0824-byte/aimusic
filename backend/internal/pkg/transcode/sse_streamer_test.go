package transcode

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSSEStreamer_PubSub1PercentIncrements(t *testing.T) {
	streamer := NewSSEStreamer()
	jobID := "job-sse-001"

	ch := streamer.Subscribe(jobID)
	defer streamer.Unsubscribe(jobID, ch)

	receivedPercents := make([]int, 0, 101)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for ev := range ch {
			receivedPercents = append(receivedPercents, ev.Percent)
			if ev.Percent == 100 {
				break
			}
		}
	}()

	// Push 0% to 100% in 1% steps
	ctx := context.Background()
	Stream1PercentIncrements(ctx, streamer, jobID, StageEncoding, 0, 100, 0)

	wg.Wait()

	if len(receivedPercents) != 101 {
		t.Fatalf("expected 101 events (0..100%%), got %d", len(receivedPercents))
	}
	for i, p := range receivedPercents {
		if p != i {
			t.Errorf("expected percent %d at index %d, got %d", i, i, p)
		}
	}
}

func TestFormatSSEMessage(t *testing.T) {
	ev := ProgressEvent{
		JobID:     "job-999",
		TrackID:   "track-888",
		Stage:     StagePackaging,
		Percent:   75,
		Message:   "Creating HLS m3u8 playlists",
		Timestamp: time.Now().UTC(),
	}

	msgBytes, err := FormatSSEMessage(ev)
	if err != nil {
		t.Fatalf("FormatSSEMessage failed: %v", err)
	}

	msg := string(msgBytes)
	if !strings.HasPrefix(msg, "event: PACKAGING_HLS\n") {
		t.Errorf("missing event line in SSE msg: %s", msg)
	}
	if !strings.Contains(msg, `"percent":75`) {
		t.Errorf("missing percent in SSE msg: %s", msg)
	}
	if !strings.HasSuffix(msg, "\n\n") {
		t.Errorf("SSE message must terminate with double newline: %s", msg)
	}
}

func TestSSEStreamer_Unsubscribe(t *testing.T) {
	streamer := NewSSEStreamer()
	jobID := "job-unsub-test"

	ch := streamer.Subscribe(jobID)
	streamer.Unsubscribe(jobID, ch)

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Errorf("expected channel to be closed after Unsubscribe")
	}
}
