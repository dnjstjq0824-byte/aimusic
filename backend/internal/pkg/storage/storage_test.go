package storage

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPresignGetShape(t *testing.T) {
	s := New("http://localhost:9000", "us-east-1", "ak", "sk", false)
	signed, err := s.PresignGet(context.Background(), "public-hls", "tracks/abc/index.m3u8", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("parse signed URL: %v", err)
	}
	if u.Scheme != "http" || u.Host != "localhost:9000" {
		t.Fatalf("unexpected base: %s", signed)
	}
	if u.Path != "/public-hls/tracks/abc/index.m3u8" {
		t.Fatalf("unexpected path: %s", u.Path)
	}
	q := u.Query()
	if q.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" {
		t.Fatalf("missing X-Amz-Algorithm: %s", signed)
	}
	if q.Get("X-Amz-Expires") != "900" {
		t.Fatalf("X-Amz-Expires = %q, want 900", q.Get("X-Amz-Expires"))
	}
	if !strings.Contains(q.Get("X-Amz-Credential"), "ak/") {
		t.Fatalf("missing access key in credential: %s", signed)
	}
	if len(q.Get("X-Amz-Signature")) != 64 {
		t.Fatalf("signature malformed: %s", signed)
	}
}

func TestPresignEscapesKeySegments(t *testing.T) {
	s := New("https://r2.example.com", "auto", "ak", "sk", true)
	signed, err := s.PresignPut(context.Background(), "staging", "uploads/한글 파일.mp3", time.Hour)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	if strings.Contains(signed, " ") || strings.Contains(signed, "한글") {
		t.Fatalf("key not escaped: %s", signed)
	}
	if !strings.HasPrefix(signed, "https://r2.example.com/staging/uploads/") {
		t.Fatalf("unexpected base: %s", signed)
	}
	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.Query().Get("X-Amz-Expires") != "3600" {
		t.Fatalf("X-Amz-Expires = %q, want 3600", u.Query().Get("X-Amz-Expires"))
	}
}

func TestPresignRequiresBucketAndKey(t *testing.T) {
	s := New("http://localhost:9000", "us-east-1", "ak", "sk", false)
	if _, err := s.PresignGet(context.Background(), "", "k", time.Minute); err == nil {
		t.Fatalf("empty bucket accepted")
	}
	if _, err := s.PresignGet(context.Background(), "b", "", time.Minute); err == nil {
		t.Fatalf("empty key accepted")
	}
}

func TestNewFromEnvLocalS3MissingKeys(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "")
	t.Setenv("S3_ACCESS_KEY", "")
	t.Setenv("S3_SECRET_KEY", "")
	if _, _, err := NewFromEnv(); err == nil {
		t.Fatalf("missing keys accepted")
	} else if !strings.Contains(err.Error(), "S3_ACCESS_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewFromEnvR2HappyPath(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "r2")
	t.Setenv("R2_ENDPOINT", "https://abc.r2.cloudflarestorage.com")
	t.Setenv("R2_ACCESS_KEY_ID", "r2ak")
	t.Setenv("R2_SECRET_ACCESS_KEY", "r2sk")
	t.Setenv("R2_STAGING_BUCKET", "stage-bkt")
	t.Setenv("R2_PUBLIC_HLS_BUCKET", "hls-bkt")
	st, buckets, err := NewFromEnv()
	if err != nil {
		t.Fatalf("NewFromEnv: %v", err)
	}
	if buckets.Staging != "stage-bkt" || buckets.PublicHLS != "hls-bkt" {
		t.Fatalf("buckets = %+v", buckets)
	}
	signed, err := st.PresignGet(context.Background(), buckets.PublicHLS, "t/index.m3u8", time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	if !strings.HasPrefix(signed, "https://abc.r2.cloudflarestorage.com/hls-bkt/") {
		t.Fatalf("unexpected signed URL: %s", signed)
	}
}

func TestNewFromEnvR2MissingConfig(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "r2")
	t.Setenv("R2_ENDPOINT", "")
	t.Setenv("R2_ACCESS_KEY_ID", "")
	t.Setenv("R2_SECRET_ACCESS_KEY", "")
	if _, _, err := NewFromEnv(); err == nil {
		t.Fatalf("missing r2 config accepted")
	}
}
