// Package generator forwards music-generation jobs to an external backend.
//
// WI-03 is setup-only: no model runs in this repo. Configure
// GENERATION_BACKEND_URL to point at a GPU server implementing POST /generate
// (see generator/server.py for the reference ACE-Step server); with the URL
// unset, SelectBackend returns DisabledBackend and jobs fail with a clear
// message instead of fake audio.
//
// Contract v2 (WI-G1) adds reproducibility + provenance: the client sends
// contractVersion 2 with modelId/modelVersion/seed/durationSec/variations/
// stems and watermark:true, and reads back weightsHash/safety/watermarkId.
// v1 servers (no contractVersion echo, bare audioBase64) keep working: v2
// fields are optional on the response path.
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

// ContractVersion is the generation HTTP contract this client speaks.
// Servers answering without a version are treated as v1 (WI-G1 compat).
const ContractVersion = 2

// Request is one generation call.
type Request struct {
	JobID  string
	Prompt string
	Lyrics string
	Model  string
	// ModelVersion pins the registry version ("" = server default).
	ModelVersion string
	// Seed pins inference randomness (nil = server-chosen).
	Seed   *int64
	Params map[string]any
}

// SafetyReport is the backend's safety verdict: prompt/similarity/watermark
// checks with an overall pass flag. Passed=false fails the job.
type SafetyReport struct {
	Passed bool     `json:"passed"`
	Checks []string `json:"checks,omitempty"`
}

// Result is decoded backend audio plus v2 provenance (empty on v1 servers).
type Result struct {
	Audio []byte
	// AudioURL is set when the backend staged audio out-of-band (large
	// payloads): the client must fetch it. Audio is set otherwise.
	AudioURL     string
	MimeType     string
	DurationMs   int
	ModelVersion string
	Seed         *int64
	WeightsHash  string
	Safety       SafetyReport
	WatermarkID  string
}

// Backend generates audio for a request.
type Backend interface {
	Generate(ctx context.Context, req Request) (*Result, error)
}

// DisabledBackend fails closed when no generation server is configured.
type DisabledBackend struct{}

// Generate always returns an error naming the missing configuration.
func (DisabledBackend) Generate(_ context.Context, _ Request) (*Result, error) {
	return nil, fmt.Errorf("music generation backend is not configured (GENERATION_BACKEND_URL unset); connect a GPU server and retry")
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
	ContractVersion int            `json:"contractVersion"`
	JobID           string         `json:"jobId"`
	Prompt          string         `json:"prompt"`
	Lyrics          string         `json:"lyrics,omitempty"`
	Model           string         `json:"model"`
	ModelVersion    string         `json:"modelVersion,omitempty"`
	Seed            *int64         `json:"seed,omitempty"`
	Watermark       bool           `json:"watermark"`
	Params          map[string]any `json:"params,omitempty"`
}

type remoteResponse struct {
	ContractVersion int           `json:"contractVersion"`
	AudioBase64     string        `json:"audioBase64,omitempty"`
	AudioURL        string        `json:"audioUrl,omitempty"`
	MimeType        string        `json:"mimeType"`
	DurationMs      int           `json:"durationMs"`
	ModelVersion    string        `json:"modelVersion,omitempty"`
	Seed            *int64        `json:"seed,omitempty"`
	WeightsHash     string        `json:"weightsHash,omitempty"`
	Safety          *SafetyReport `json:"safety,omitempty"`
	WatermarkID     string        `json:"watermarkId,omitempty"`
}

// Generate calls POST {base}/generate and decodes the audio payload.
//
// v2 responses carry provenance (modelVersion/seed/weightsHash/safety/
// watermarkId) and either inline audioBase64 or an audioUrl to fetch for
// large payloads. v1 responses (bare audioBase64, no version) decode with
// empty provenance. A v2 safety verdict of passed:false fails the job.
func (b *RemoteBackend) Generate(ctx context.Context, req Request) (*Result, error) {
	body, err := json.Marshal(remoteRequest{
		ContractVersion: ContractVersion,
		JobID:           req.JobID, Prompt: req.Prompt, Lyrics: req.Lyrics,
		Model: req.Model, ModelVersion: req.ModelVersion, Seed: req.Seed,
		Watermark: true, Params: req.Params,
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
	if decoded.Safety != nil && !decoded.Safety.Passed {
		return nil, fmt.Errorf("generation backend safety verdict: failed (%s)", strings.Join(decoded.Safety.Checks, ","))
	}
	res := &Result{
		AudioURL: decoded.AudioURL, MimeType: decoded.MimeType, DurationMs: decoded.DurationMs,
		ModelVersion: decoded.ModelVersion, Seed: decoded.Seed,
		WeightsHash: decoded.WeightsHash, WatermarkID: decoded.WatermarkID,
	}
	if decoded.Safety != nil {
		res.Safety = *decoded.Safety
	}
	if decoded.AudioURL != "" {
		return res, b.fetchAudioURL(ctx, res)
	}
	audio, err := base64.StdEncoding.DecodeString(decoded.AudioBase64)
	if err != nil {
		return nil, fmt.Errorf("decode generation audio: %w", err)
	}
	if len(audio) == 0 || len(audio) > MaxAudioBytes {
		return nil, fmt.Errorf("generation audio size %d out of bounds", len(audio))
	}
	if res.MimeType == "" {
		res.MimeType = "audio/wav"
	}
	res.Audio = audio
	return res, nil
}

// fetchAudioURL downloads an out-of-band v2 payload into res.Audio.
func (b *RemoteBackend) fetchAudioURL(ctx context.Context, res *Result) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, res.AudioURL, nil)
	if err != nil {
		return fmt.Errorf("build audio fetch request: %w", err)
	}
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("fetch generation audio: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch generation audio: %s", resp.Status)
	}
	audio, err := io.ReadAll(io.LimitReader(resp.Body, MaxAudioBytes+1))
	if err != nil {
		return fmt.Errorf("read generation audio: %w", err)
	}
	if len(audio) == 0 || len(audio) > MaxAudioBytes {
		return fmt.Errorf("generation audio size %d out of bounds", len(audio))
	}
	if res.MimeType == "" {
		res.MimeType = "audio/wav"
	}
	res.Audio = audio
	return nil
}

// SelectBackend returns a remote backend when url is set, else DisabledBackend.
func SelectBackend(url string, timeout time.Duration) Backend {
	if b, err := NewRemoteBackend(url, timeout); err == nil {
		return b
	}
	return DisabledBackend{}
}
