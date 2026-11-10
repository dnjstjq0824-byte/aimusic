// Package storage provides S3-compatible presigned URLs for LocalStack (local) and R2.
package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Storage issues presigned URLs and deletes objects.
type Storage interface {
	PresignPut(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	// Get fetches a whole object (at most maxBytes+1 probed; error when over).
	Get(ctx context.Context, bucket, key string, maxBytes int64) ([]byte, error)
	Delete(ctx context.Context, bucket, key string) error
	// ListPrefix returns every key under prefix (directory-style listing).
	ListPrefix(ctx context.Context, bucket, prefix string) ([]string, error)
	// DeletePrefix removes every object under prefix; empty result is success.
	DeletePrefix(ctx context.Context, bucket, prefix string) error
}

// Buckets holds the staging (private) and HLS (public) bucket names.
type Buckets struct {
	Staging   string
	PublicHLS string
}

// S3Compat is an S3-compatible implementation (LocalStack local, R2 in staging/prod)
// using AWS Signature Version 4 query-string authentication.
type S3Compat struct {
	endpoint  string
	region    string
	accessKey string
	secretKey string
	useSSL    bool
	client    *http.Client
}

// New builds an S3Compat client.
func New(endpoint, region, accessKey, secretKey string, useSSL bool) *S3Compat {
	return &S3Compat{
		endpoint:  strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		region:    region,
		accessKey: accessKey,
		secretKey: secretKey,
		useSSL:    useSSL,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// NewFromEnv builds a Storage from the environment, switching to R2 when
// STORAGE_DRIVER=r2.
func NewFromEnv() (Storage, Buckets, error) {
	driver := strings.TrimSpace(os.Getenv("STORAGE_DRIVER"))
	if driver == "r2" {
		endpoint := strings.TrimSpace(os.Getenv("R2_ENDPOINT"))
		ak := strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID"))
		sk := os.Getenv("R2_SECRET_ACCESS_KEY")
		if endpoint == "" || ak == "" || sk == "" {
			return nil, Buckets{}, fmt.Errorf("r2 storage misconfigured: R2_ENDPOINT/R2_ACCESS_KEY_ID/R2_SECRET_ACCESS_KEY required")
		}
		buckets := Buckets{
			Staging:   firstNonEmpty(os.Getenv("R2_STAGING_BUCKET"), "staging-private"),
			PublicHLS: firstNonEmpty(os.Getenv("R2_PUBLIC_HLS_BUCKET"), "public-hls"),
		}
		return New(endpoint, firstNonEmpty(os.Getenv("S3_REGION"), "auto"), ak, sk, true), buckets, nil
	}
	endpoint := firstNonEmpty(os.Getenv("S3_ENDPOINT"), "http://localhost:4566")
	ak := strings.TrimSpace(os.Getenv("S3_ACCESS_KEY"))
	sk := os.Getenv("S3_SECRET_KEY")
	if ak == "" || sk == "" {
		return nil, Buckets{}, fmt.Errorf("local s3 storage misconfigured: S3_ACCESS_KEY/S3_SECRET_KEY required")
	}
	buckets := Buckets{
		Staging:   firstNonEmpty(os.Getenv("S3_STAGING_BUCKET"), "local-staging"),
		PublicHLS: firstNonEmpty(os.Getenv("S3_PUBLIC_HLS_BUCKET"), "local-hls"),
	}
	return New(endpoint, firstNonEmpty(os.Getenv("S3_REGION"), "us-east-1"), ak, sk, os.Getenv("S3_USE_SSL") == "true"), buckets, nil
}

// Endpoint returns the configured S3 endpoint.
func (s *S3Compat) Endpoint() string { return s.endpoint }

// PresignPut returns a presigned URL for uploading key via HTTP PUT.
func (s *S3Compat) PresignPut(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	return s.presign("PUT", bucket, key, ttl)
}

// PresignGet returns a presigned URL for downloading key via HTTP GET.
func (s *S3Compat) PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	return s.presign("GET", bucket, key, ttl)
}

// Get fetches a whole object via a presigned URL, refusing bodies over
// maxBytes so a corrupt key cannot OOM the API.
func (s *S3Compat) Get(ctx context.Context, bucket, key string, maxBytes int64) ([]byte, error) {
	signed, err := s.presign("GET", bucket, key, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, signed, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("get %s/%s: unexpected status %d", bucket, key, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("get %s/%s: object exceeds %d bytes", bucket, key, maxBytes)
	}
	return body, nil
}

// Delete removes an object; missing keys are treated as success.
func (s *S3Compat) Delete(ctx context.Context, bucket, key string) error {
	signed, err := s.presign("DELETE", bucket, key, 5*time.Minute)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, signed, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("delete %s/%s: unexpected status %d", bucket, key, resp.StatusCode)
}

// listBucketResult mirrors the S3 ListObjectsV2 XML envelope.
type listBucketResult struct {
	XMLName  xml.Name `xml:"ListBucketResult"`
	Contents []struct {
		Key string `xml:"Key"`
	} `xml:"Contents"`
	IsTruncated           bool   `xml:"IsTruncated"`
	NextContinuationToken string `xml:"NextContinuationToken"`
}

// ListPrefix returns every key under prefix via ListObjectsV2 (paginated).
// The prefix is normalized to directory form ("hls/<id>" -> "hls/<id>/") so
// sibling tracks can never match.
func (s *S3Compat) ListPrefix(ctx context.Context, bucket, prefix string) ([]string, error) {
	if bucket == "" || strings.Trim(prefix, "/") == "" {
		return nil, fmt.Errorf("bucket and prefix are required")
	}
	prefix = strings.Trim(prefix, "/") + "/"
	var out []string
	token := ""
	for {
		extra := url.Values{}
		extra.Set("list-type", "2")
		extra.Set("prefix", prefix)
		extra.Set("max-keys", "1000")
		if token != "" {
			extra.Set("continuation-token", token)
		}
		signed, err := s.presignQuery("GET", bucket, "", 5*time.Minute, extra)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, signed, nil)
		if err != nil {
			return nil, err
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("list %s/%s: unexpected status %d", bucket, prefix, resp.StatusCode)
		}
		var res listBucketResult
		if err := xml.Unmarshal(body, &res); err != nil {
			return nil, fmt.Errorf("list %s/%s: bad xml: %w", bucket, prefix, err)
		}
		for _, c := range res.Contents {
			if strings.HasPrefix(c.Key, prefix) {
				out = append(out, c.Key)
			}
		}
		if !res.IsTruncated || res.NextContinuationToken == "" {
			break
		}
		token = res.NextContinuationToken
	}
	return out, nil
}

// DeletePrefix removes every object under prefix. A prefix with no objects
// is success; the first delete failure aborts with an error.
func (s *S3Compat) DeletePrefix(ctx context.Context, bucket, prefix string) error {
	keys, err := s.ListPrefix(ctx, bucket, prefix)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := s.Delete(ctx, bucket, k); err != nil {
			return err
		}
	}
	return nil
}

func (s *S3Compat) presign(method, bucket, key string, ttl time.Duration) (string, error) {
	return s.presignQuery(method, bucket, key, ttl, nil)
}

// presignQuery signs extra query parameters (e.g. ListObjectsV2) into the URL.
func (s *S3Compat) presignQuery(method, bucket, key string, ttl time.Duration, extra url.Values) (string, error) {
	if bucket == "" {
		return "", fmt.Errorf("bucket and key are required")
	}
	if extra == nil && key == "" {
		return "", fmt.Errorf("bucket and key are required")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if ttl > 7*24*time.Hour {
		ttl = 7 * 24 * time.Hour
	}
	u, err := url.Parse(s.endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	credentialScope := dateStamp + "/" + s.region + "/s3/aws4_request"

	canonicalURI := "/" + bucket + "/"
	if key != "" {
		canonicalURI += escapeKey(key)
	}
	q := url.Values{}
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	q.Set("X-Amz-Credential", s.accessKey+"/"+credentialScope)
	q.Set("X-Amz-Date", amzDate)
	q.Set("X-Amz-Expires", strconv.Itoa(int(ttl.Seconds())))
	q.Set("X-Amz-SignedHeaders", "host")
	canonicalQuery := q.Encode()

	canonicalHeaders := "host:" + u.Host + "\n"
	canonicalRequest := method + "\n" + canonicalURI + "\n" + canonicalQuery + "\n" +
		canonicalHeaders + "\n" + "host" + "\n" + "UNSIGNED-PAYLOAD"
	hashedCanonical := sha256Hex([]byte(canonicalRequest))
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + credentialScope + "\n" + hashedCanonical
	signingKey := deriveSigningKey(s.secretKey, dateStamp, s.region, "s3")
	signature := hmacHex(signingKey, stringToSign)
	return s.endpoint + canonicalURI + "?" + canonicalQuery + "&X-Amz-Signature=" + signature, nil
}

func deriveSigningKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func hmacHex(key []byte, data string) string {
	return hex.EncodeToString(hmacSHA256(key, data))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func escapeKey(key string) string {
	parts := strings.Split(strings.TrimPrefix(key, "/"), "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
