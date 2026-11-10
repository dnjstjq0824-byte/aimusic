// Package generator forwards music-generation jobs to an external backend.
//
// WI-03 is setup-only: no model runs in this repo. Configure
// GENERATION_BACKEND_URL to point at a GPU server implementing POST /generate
// (see generator/server.py for the reference ACE-Step server); with the URL
// unset, SelectBackend returns DisabledBackend and jobs fail with a clear
// message instead of fake audio.
package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MaxAudioBytes caps a single backend response (200 MiB).
const MaxAudioBytes = 200 << 20

// Request is one generation call.
type Request struct {
	JobID  string
	Prompt string
	Lyrics string
	Model  string
	Params map[string]any
}

// Result is decoded backend audio.
type Result struct {
	Audio      []byte
	MimeType   string
	DurationMs int
}

// Backend generates audio for a request.
type Backend interface {
	Generate(ctx context.Context, req Request) (*Result, error)
}

// DisabledBackend fails closed when no generation server is configured.
type DisabledBackend struct{}

// Generate always returns an error naming the missing configuration.
func (DisabledBackend) Generate(_ context.Context, _ Request) (*Result, error) {
	return nil, fmt.Errorf("음악 생성 백엔드가 설정되지 않았습니다 (GENERATION_BACKEND_URL 미설정). GPU 서버를 연결한 뒤 다시 시도하세요")
}

// RemoteBackend POSTs jobs to a GPU server (POST {base}/generate).
type RemoteBackend struct {
	base   string
	client *http.Client
}

// NewRemoteBackend builds a backend for base (trailing slashes trimmed).
func NewRemoteBackend(base string, timeout time.Duration) (*RemoteBackend, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil, fmt.Errorf("generation backend URL is empty")
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return &RemoteBackend{base: base, client: &http.Client{Timeout: timeout}}, nil
}

type remoteRequest struct {
	JobID  string         `json:"jobId"`
	Prompt string         `json:"prompt"`
	Lyrics string         `json:"lyrics,omitempty"`
	Model  string         `json:"model"`
	Params map[string]any `json:"params,omitempty"`
}

type remoteResponse struct {
	AudioBase64 string `json:"audioBase64"`
	MimeType    string `json:"mimeType"`
	DurationMs  int    `json:"durationMs"`
}

// Generate calls POST {base}/generate and decodes the audio payload.
func (b *RemoteBackend) Generate(ctx context.Context, req Request) (*Result, error) {
	body, err := json.Marshal(remoteRequest{
		JobID: req.JobID, Prompt: req.Prompt, Lyrics: req.Lyrics,
		Model: req.Model, Params: req.Params,
	})
	if err != nil {
		return nil, fmt.Errorf("encode generation request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.base+"/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build generation request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("generation backend unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxAudioBytes+4096))
	if err != nil {
		return nil, fmt.Errorf("read generation response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generation backend %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var decoded remoteResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode generation response: %w", err)
	}
	audio, err := base64.StdEncoding.DecodeString(decoded.AudioBase64)
	if err != nil {
		return nil, fmt.Errorf("decode generation audio: %w", err)
	}
	if len(audio) == 0 || len(audio) > MaxAudioBytes {
		return nil, fmt.Errorf("generation audio size %d out of bounds", len(audio))
	}
	if decoded.MimeType == "" {
		decoded.MimeType = "audio/wav"
	}
	return &Result{Audio: audio, MimeType: decoded.MimeType, DurationMs: decoded.DurationMs}, nil
}

// SelectBackend returns a remote backend when url is set, else DisabledBackend.
func SelectBackend(url string, timeout time.Duration) Backend {
	if b, err := NewRemoteBackend(url, timeout); err == nil {
		return b
	}
	return DisabledBackend{}
}
