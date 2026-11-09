// Package transcode provides real-time Server-Sent Events (SSE) broadcasting
// for audio transcoding job progress, 1% increment updates, and stage state synchronization.
package transcode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// TranscodeStage defines current execution milestone.
type TranscodeStage string

const (
	StageQueued      TranscodeStage = "QUEUED"
	StageDownloading TranscodeStage = "DOWNLOADING"
	StageNormalizing TranscodeStage = "NORMALIZING_LUFS"
	StageEncoding    TranscodeStage = "ENCODING_MULTIBITRATE"
	StagePackaging   TranscodeStage = "PACKAGING_HLS"
	StageUploading   TranscodeStage = "UPLOADING_S3"
	StageCompleted   TranscodeStage = "COMPLETED"
	StageFailed      TranscodeStage = "FAILED"
)

// ProgressEvent represents an individual progress update sent to SSE clients.
type ProgressEvent struct {
	JobID     string         `json:"job_id"`
	TrackID   string         `json:"track_id,omitempty"`
	Stage     TranscodeStage `json:"stage"`
	Percent   int            `json:"percent"` // 0 to 100
	Message   string         `json:"message,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// SSEStreamer manages active subscribers and broadcasts real-time progress events.
type SSEStreamer struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan ProgressEvent]struct{} // JobID -> set of client channels
}

// NewSSEStreamer creates an SSE broadcaster.
func NewSSEStreamer() *SSEStreamer {
	return &SSEStreamer{
		subscribers: make(map[string]map[chan ProgressEvent]struct{}),
	}
}

// Subscribe attaches a listener channel to receive events for a target job.
func (s *SSEStreamer) Subscribe(jobID string) chan ProgressEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan ProgressEvent, 256) // large buffer to accommodate fast bursts
	if _, ok := s.subscribers[jobID]; !ok {
		s.subscribers[jobID] = make(map[chan ProgressEvent]struct{})
	}
	s.subscribers[jobID][ch] = struct{}{}
	return ch
}

// Unsubscribe detaches and closes a listener channel.
func (s *SSEStreamer) Unsubscribe(jobID string, ch chan ProgressEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if subs, ok := s.subscribers[jobID]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(s.subscribers, jobID)
		}
	}
}

// Broadcast pushes an event to all subscribers listening to the given job.
func (s *SSEStreamer) Broadcast(event ProgressEvent) {
	if event.Percent < 0 {
		event.Percent = 0
	} else if event.Percent > 100 {
		event.Percent = 100
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	subs, ok := s.subscribers[event.JobID]
	if !ok {
		return
	}

	for ch := range subs {
		select {
		case ch <- event:
		default:
			// Non-blocking drop if slow client buffer full
		}
	}
}

// FormatSSEMessage serializes an event into standard SSE text/event-stream wire format.
func FormatSSEMessage(event ProgressEvent) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Stage, string(data))
	return []byte(msg), nil
}

// ServeHTTP handles incoming SSE HTTP connections from frontend clients.
func (s *SSEStreamer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "missing job_id parameter", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := s.Subscribe(jobID)
	defer s.Unsubscribe(jobID, ch)

	// Send initial connected ping
	_, _ = io.WriteString(w, ": connected\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			msg, err := FormatSSEMessage(ev)
			if err != nil {
				continue
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()

			if ev.Stage == StageCompleted || ev.Stage == StageFailed {
				return
			}
		}
	}
}

// Stream1PercentIncrements helper simulates/pushes 1% step increments from startPct to endPct.
func Stream1PercentIncrements(ctx context.Context, streamer *SSEStreamer, jobID string, stage TranscodeStage, startPct, endPct int, stepDelay time.Duration) {
	for p := startPct; p <= endPct; p++ {
		select {
		case <-ctx.Done():
			return
		default:
			streamer.Broadcast(ProgressEvent{
				JobID:   jobID,
				Stage:   stage,
				Percent: p,
			})
			if stepDelay > 0 {
				time.Sleep(stepDelay)
			}
		}
	}
}
