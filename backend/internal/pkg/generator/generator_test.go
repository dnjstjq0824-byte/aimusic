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

func TestSelectBackend(t *testing.T) {
	if _, ok := SelectBackend("", time.Second).(DisabledBackend); !ok {
		t.Error("empty URL must select DisabledBackend")
	}
	b := SelectBackend("http://gpu:8000", time.Second)
	if _, ok := b.(*RemoteBackend); !ok {
		t.Errorf("URL must select RemoteBackend, got %T", b)
	}
}
