package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDisabledBackendFailsClosed(t *testing.T) {
	b := DisabledBackend{}
	_, err := b.Generate(context.Background(), Request{JobID: "j1", Prompt: "piano ballad", Model: "ace-step-v1"})
	if err == nil {
		t.Fatal("disabled backend must fail, not fake audio")
	}
}

func TestRemoteBackendRoundTrip(t *testing.T) {
	var got struct {
		JobID  string `json:"jobId"`
		Prompt string `json:"prompt"`
		Model  string `json:"model"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"audioBase64": base64.StdEncoding.EncodeToString([]byte("RIFFfake-wav")),
			"mimeType":    "audio/wav",
			"durationMs":  30000,
		})
	}))
	defer srv.Close()

	b, err := NewRemoteBackend(srv.URL, 10*time.Second)
	if err != nil {
		t.Fatalf("NewRemoteBackend: %v", err)
	}
	res, err := b.Generate(context.Background(), Request{JobID: "j1", Prompt: "piano ballad", Model: "ace-step-v1"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.JobID != "j1" || got.Prompt != "piano ballad" || got.Model != "ace-step-v1" {
		t.Errorf("backend got %+v", got)
	}
	if string(res.Audio) != "RIFFfake-wav" || res.MimeType != "audio/wav" || res.DurationMs != 30000 {
		t.Errorf("unexpected result %+v", res)
	}
}

func TestRemoteBackendRequiresURL(t *testing.T) {
	if _, err := NewRemoteBackend("  ", time.Second); err == nil {
		t.Error("empty URL must fail")
	}
}

func TestRemoteBackendSurfacesBackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "model loading"})
	}))
	defer srv.Close()
	b, err := NewRemoteBackend(srv.URL, 10*time.Second)
	if err != nil {
		t.Fatalf("NewRemoteBackend: %v", err)
	}
	if _, err := b.Generate(context.Background(), Request{JobID: "j1", Prompt: "x", Model: "m"}); err == nil {
		t.Error("backend 503 must surface as error")
	}
}

func TestRemoteBackendV2RoundTrip(t *testing.T) {
	var got struct {
		ContractVersion int    `json:"contractVersion"`
		ModelVersion    string `json:"modelVersion"`
		Seed            *int64 `json:"seed"`
		Watermark       bool   `json:"watermark"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"contractVersion": 2,
			"audioBase64":     base64.StdEncoding.EncodeToString([]byte("RIFFv2-wav")),
			"mimeType":        "audio/wav",
			"durationMs":      45000,
			"modelVersion":    "ace-step-v1.2",
			"seed":            42,
			"weightsHash":     "sha256:abc123",
			"safety":          map[string]any{"passed": true, "checks": []string{"prompt", "similarity", "watermark"}},
			"watermarkId":     "wm-42",
		})
	}))
	defer srv.Close()

	b, err := NewRemoteBackend(srv.URL, 10*time.Second)
	if err != nil {
		t.Fatalf("NewRemoteBackend: %v", err)
	}
	seed := int64(42)
	res, err := b.Generate(context.Background(), Request{JobID: "j1", Prompt: "piano", Model: "ace-step-v1", ModelVersion: "ace-step-v1.2", Seed: &seed})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.ContractVersion != 2 || !got.Watermark {
		t.Errorf("v2 request must carry contractVersion 2 + watermark:true, got %+v", got)
	}
	if got.ModelVersion != "ace-step-v1.2" || got.Seed == nil || *got.Seed != 42 {
		t.Errorf("v2 request must echo pins, got %+v", got)
	}
	if string(res.Audio) != "RIFFv2-wav" || res.DurationMs != 45000 {
		t.Errorf("unexpected audio %+v", res)
	}
	if res.ModelVersion != "ace-step-v1.2" || res.Seed == nil || *res.Seed != 42 {
		t.Errorf("provenance pins lost: %+v", res)
	}
	if res.WeightsHash != "sha256:abc123" || res.WatermarkID != "wm-42" {
		t.Errorf("provenance evidence lost: %+v", res)
	}
	if !res.Safety.Passed || len(res.Safety.Checks) != 3 {
		t.Errorf("safety verdict lost: %+v", res.Safety)
	}
}

func TestRemoteBackendV1Compat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"audioBase64": base64.StdEncoding.EncodeToString([]byte("RIFFv1-wav")),
			"mimeType":    "audio/wav",
			"durationMs":  30000,
		})
	}))
	defer srv.Close()

	b, err := NewRemoteBackend(srv.URL, 10*time.Second)
	if err != nil {
		t.Fatalf("NewRemoteBackend: %v", err)
	}
	res, err := b.Generate(context.Background(), Request{JobID: "j1", Prompt: "x", Model: "m"})
	if err != nil {
		t.Fatalf("v1 response must still decode: %v", err)
	}
	if string(res.Audio) != "RIFFv1-wav" {
		t.Errorf("unexpected audio %+v", res)
	}
	if res.ModelVersion != "" || res.Seed != nil || res.WeightsHash != "" || res.WatermarkID != "" || res.Safety.Passed {
		t.Errorf("v1 response must yield empty provenance: %+v", res)
	}
}

func TestRemoteBackendSafetyFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"contractVersion": 2,
			"audioBase64":     base64.StdEncoding.EncodeToString([]byte("RIFFx")),
			"safety":          map[string]any{"passed": false, "checks": []string{"similarity"}},
		})
	}))
	defer srv.Close()

	b, err := NewRemoteBackend(srv.URL, 10*time.Second)
	if err != nil {
		t.Fatalf("NewRemoteBackend: %v", err)
	}
	if _, err := b.Generate(context.Background(), Request{JobID: "j1", Prompt: "x", Model: "m"}); err == nil {
		t.Error("safety passed:false must fail the job")
	}
}

func TestRemoteBackendAudioURL(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"contractVersion": 2,
				"audioUrl":        srv.URL + "/audio/j1.wav",
				"mimeType":        "audio/wav",
				"durationMs":      60000,
				"watermarkId":     "wm-9",
			})
		case "/audio/j1.wav":
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write([]byte("RIFFurl-wav"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	b, err := NewRemoteBackend(srv.URL, 10*time.Second)
	if err != nil {
		t.Fatalf("NewRemoteBackend: %v", err)
	}
	res, err := b.Generate(context.Background(), Request{JobID: "j1", Prompt: "x", Model: "m"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if string(res.Audio) != "RIFFurl-wav" || res.WatermarkID != "wm-9" {
		t.Errorf("audioUrl payload not fetched: %+v", res)
	}
}

func TestSelectBackend(t *testing.T) {
	if _, ok := SelectBackend("", time.Second).(DisabledBackend); !ok {
		t.Error("empty URL must select DisabledBackend")
	}
	b := SelectBackend("http://gpu:8000", time.Second)
	if _, ok := b.(*RemoteBackend); !ok {
		t.Errorf("URL must select RemoteBackend, got %T", b)
	}
}
