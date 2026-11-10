package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// fakeS3 serves ListObjectsV2 pages and records DELETEs.
type fakeS3 struct {
	keys     []string
	pages    int // pages served (pagination drill)
	deleted  []string
	failList bool
}

func (f *fakeS3) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			f.deleted = append(f.deleted, strings.TrimPrefix(r.URL.Path, "/bkt/"))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Query().Get("list-type") != "2" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if f.failList {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		f.pages++
		prefix := r.URL.Query().Get("prefix")
		cont := r.URL.Query().Get("continuation-token")
		// One key per page to force pagination.
		start := 0
		if cont != "" {
			fmt.Sscanf(cont, "tok-%d", &start)
		}
		var sb strings.Builder
		sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		sb.WriteString(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
		matched := []string{}
		for _, k := range f.keys {
			if strings.HasPrefix(k, prefix) {
				matched = append(matched, k)
			}
		}
		if start < len(matched) {
			sb.WriteString("<Contents><Key>" + matched[start] + "</Key></Contents>")
		}
		if start+1 < len(matched) {
			sb.WriteString("<IsTruncated>true</IsTruncated>")
			sb.WriteString(fmt.Sprintf("<NextContinuationToken>tok-%d</NextContinuationToken>", start+1))
		} else {
			sb.WriteString("<IsTruncated>false</IsTruncated>")
		}
		sb.WriteString("</ListBucketResult>")
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, sb.String())
	}
}

func TestListPrefixPaginatesAndScopesToDirectory(t *testing.T) {
	f := &fakeS3{keys: []string{
		"hls/track-1/master.m3u8",
		"hls/track-1/seg0.ts",
		"hls/track-10/master.m3u8", // sibling: must NOT match hls/track-1
		"staging/other.mp3",
	}}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	s := New(srv.URL, "us-east-1", "ak", "sk", false)

	keys, err := s.ListPrefix(context.Background(), "bkt", "hls/track-1")
	if err != nil {
		t.Fatalf("ListPrefix: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("keys = %v, want 2 under hls/track-1/", keys)
	}
	if f.pages != 2 {
		t.Fatalf("pages = %d, want 2 (pagination)", f.pages)
	}
}

func TestListPrefixRejectsEmpty(t *testing.T) {
	s := New("http://localhost:9000", "us-east-1", "ak", "sk", false)
	if _, err := s.ListPrefix(context.Background(), "bkt", ""); err == nil {
		t.Fatalf("empty prefix accepted")
	}
	if _, err := s.ListPrefix(context.Background(), "bkt", "///"); err == nil {
		t.Fatalf("blank prefix accepted")
	}
	if _, err := s.ListPrefix(context.Background(), "", "hls/a"); err == nil {
		t.Fatalf("empty bucket accepted")
	}
}

func TestDeletePrefixRemovesListedKeys(t *testing.T) {
	f := &fakeS3{keys: []string{"hls/t9/master.m3u8", "hls/t9/seg0.ts"}}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	s := New(srv.URL, "us-east-1", "ak", "sk", false)

	if err := s.DeletePrefix(context.Background(), "bkt", "hls/t9"); err != nil {
		t.Fatalf("DeletePrefix: %v", err)
	}
	if len(f.deleted) != 2 {
		t.Fatalf("deleted = %v, want 2 keys", f.deleted)
	}
}

func TestDeletePrefixEmptyIsSuccessListFailureIsError(t *testing.T) {
	f := &fakeS3{}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	s := New(srv.URL, "us-east-1", "ak", "sk", false)

	if err := s.DeletePrefix(context.Background(), "bkt", "hls/nothing"); err != nil {
		t.Fatalf("empty prefix: %v", err)
	}
	f.failList = true
	if err := s.DeletePrefix(context.Background(), "bkt", "hls/nothing"); err == nil {
		t.Fatalf("list failure accepted")
	}
}
