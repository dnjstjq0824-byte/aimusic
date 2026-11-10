// Command worker polls the Redis transcode queue and runs the audio processor.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/pkg/audio"
	"avibe-platform/backend/internal/pkg/generator"
	"avibe-platform/backend/internal/pkg/licensegate"
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
)

const maxAttempts = 3

// maxStagingBytes caps a single staged download (500 MiB). Partners uploading
// larger masters must split or compress first; the cap bounds worker disk use.
const maxStagingBytes = 500 << 20

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := repository.OpenDB(cfg.DB.DSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	rdb := repository.NewRedisClient(cfg.Redis.Addr)
	defer rdb.Close()
	st, buckets, err := storage.NewFromEnv()
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	repos := repository.New(db, rdb)
	genBackend := generator.SelectBackend(cfg.Generation.BackendURL, time.Duration(cfg.Generation.TimeoutSeconds)*time.Second)
	w := &worker{repos: repos, storage: st, hlsBucket: buckets.PublicHLS, stagingBucket: buckets.Staging,
		http: &http.Client{Timeout: 5 * time.Minute}, genBackend: genBackend}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go runDisputeSLASweeper(ctx, repos)
	go runJobReaper(ctx, repos, buckets.Staging)
	go runGenerationLoop(ctx, w)
	log.Printf("worker polling queue %q (generation backend: %q)", repository.TranscodeQueueKey, cfg.Generation.BackendURL)
	for {
		select {
		case <-ctx.Done():
			log.Printf("worker shutting down")
			return
		default:
		}
		payload, err := repos.Queue.DequeueTranscode(ctx, 10*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("dequeue: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if payload == nil {
			continue
		}
		if err := w.process(ctx, payload); err != nil {
			log.Printf("process job=%s track=%s: %v", payload.JobID, payload.TrackID, err)
		}
	}
}

type worker struct {
	repos         *repository.Repos
	storage       storage.Storage
	hlsBucket     string
	stagingBucket string
	http          *http.Client
	genBackend    generator.Backend
}

// runGenerationLoop polls the generation queue (WI-03 setup-only). With no
// GENERATION_BACKEND_URL configured, jobs fail with the backend's message.
func runGenerationLoop(ctx context.Context, w *worker) {
	log.Printf("generation loop polling queue %q", repository.GenerationQueueKey)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		payload, err := w.repos.Queue.DequeueGeneration(ctx, 10*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("generation dequeue: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if payload == nil {
			continue
		}
		if err := w.processGeneration(ctx, payload); err != nil {
			log.Printf("process generation job=%s: %v", payload.JobID, err)
		}
	}
}

// processGeneration forwards one job to the external GPU backend and stages
// the returned audio for the normal PreCheck/Finalize flow.
func (w *worker) processGeneration(ctx context.Context, p *repository.GenerationPayload) error {
	job, err := w.repos.GenJobs.GetByID(ctx, p.JobID)
	if err != nil {
		return err
	}
	if err := w.repos.GenJobs.MarkRunning(ctx, job.ID); err != nil {
		return err
	}
	fail := func(msg string) error {
		log.Printf("generation job %s failed: %s", job.ID, msg)
		if err := w.repos.GenJobs.MarkFailed(ctx, job.ID, msg); err != nil {
			return err
		}
		updated, err := w.repos.GenJobs.GetByID(ctx, job.ID)
		if err != nil {
			return err
		}
		if updated.Attempts < maxAttempts {
			return w.repos.Queue.EnqueueGeneration(ctx, *p)
		}
		return nil
	}
	var params map[string]any
	if strings.TrimSpace(job.ParamsJSON) != "" && job.ParamsJSON != "{}" {
		if err := json.Unmarshal([]byte(job.ParamsJSON), &params); err != nil {
			return fail(fmt.Sprintf("저장된 생성 파라미터 파싱 실패: %v", err))
		}
	}
	lyrics := ""
	if job.Lyrics != nil {
		lyrics = *job.Lyrics
	}
	res, err := w.genBackend.Generate(ctx, generator.Request{
		JobID: job.ID, Prompt: job.Prompt, Lyrics: lyrics, Model: job.Model, Params: params,
	})
	if err != nil {
		return fail(err.Error())
	}
	sum := sha256.Sum256(res.Audio)
	ext := ".wav"
	switch strings.ToLower(strings.TrimSpace(res.MimeType)) {
	case "audio/mpeg", "audio/mp3":
		ext = ".mp3"
	case "audio/flac":
		ext = ".flac"
	}
	key := licensegate.BuildStagingKey(job.PartnerID, hex.EncodeToString(sum[:]), "generated-"+job.ID+ext)
	if err := w.uploadBytes(ctx, w.stagingBucket, key, res.Audio); err != nil {
		return fail(fmt.Sprintf("stage generated audio: %v", err))
	}
	if err := w.repos.GenJobs.MarkSucceeded(ctx, job.ID, key, res.DurationMs); err != nil {
		return err
	}
	log.Printf("generation job %s staged to %s (%d bytes)", job.ID, key, len(res.Audio))
	return nil
}

// uploadBytes stores one object via presigned PUT.
func (w *worker) uploadBytes(ctx context.Context, bucket, key string, data []byte) error {
	signed, err := w.storage.PresignPut(ctx, bucket, key, 15*time.Minute)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, signed, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(data))
	resp, err := w.http.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("upload %s: status %d", key, resp.StatusCode)
	}
	return nil
}

func (w *worker) process(ctx context.Context, p *repository.TranscodePayload) error {
	job, err := w.repos.Jobs.GetByID(ctx, p.JobID)
	if err != nil {
		return err
	}
	track, err := w.repos.Tracks.GetByID(ctx, p.TrackID)
	if err != nil {
		return err
	}
	if err := w.repos.Jobs.MarkRunning(ctx, job.ID); err != nil {
		return err
	}
	if err := w.repos.Tracks.UpdateStatus(ctx, track.ID, domain.TrackTranscoding, nil); err != nil {
		return err
	}
	fail := func(msg string, licenseReject bool) error {
		log.Printf("job %s failed: %s", job.ID, msg)
		if err := w.repos.Jobs.MarkFailed(ctx, job.ID, msg); err != nil {
			return err
		}
		if licenseReject {
			if p.InputBucket != "" && p.InputKey != "" {
				if delErr := w.storage.Delete(ctx, p.InputBucket, p.InputKey); delErr != nil {
					log.Printf("warn: delete rejected staging file %s/%s: %v", p.InputBucket, p.InputKey, delErr)
				} else {
					log.Printf("deleted rejected staging file %s/%s", p.InputBucket, p.InputKey)
				}
			}
			return w.repos.Tracks.UpdateStatus(ctx, track.ID, domain.TrackLicenseGateRejected, &msg)
		}
		updated, err := w.repos.Jobs.GetByID(ctx, job.ID)
		if err != nil {
			return err
		}
		if updated.Attempts < maxAttempts {
			return w.repos.Queue.EnqueueTranscode(ctx, *p)
		}
		msg2 := msg + " (attempts exhausted)"
		return w.repos.Tracks.UpdateStatus(ctx, track.ID, domain.TrackLicenseGateRejected, &msg2)
	}

	policy, err := w.repos.Policies.Get(ctx, track.AIProvider)
	if err != nil {
		return fail(fmt.Sprintf("policy lookup for %s: %v", track.AIProvider, err), true)
	}
	planTier, err := w.recheckLicense(ctx, track, policy)
	if err != nil {
		_ = w.repos.Rights.SetGate3Report(ctx, track.ID, false, audio.FormatFFprobeReportJSON(nil, false, err.Error()))
		return fail(err.Error(), true)
	}

	tmpDir, err := os.MkdirTemp("", "transcode-")
	if err != nil {
		return fail(fmt.Sprintf("temp dir: %v", err), false)
	}
	defer os.RemoveAll(tmpDir)

	ext := strings.ToLower(filepath.Ext(p.InputKey))
	if ext == "" {
		ext = ".bin"
	}
	localInput := filepath.Join(tmpDir, "input"+ext)
	if err := w.download(ctx, p.InputBucket, p.InputKey, localInput); err != nil {
		return fail(fmt.Sprintf("download input: %v", err), false)
	}
	// Integrity: the staged bytes must match the client-declared SHA-256
	// embedded in the staging key, otherwise duplicate detection is bypassable.
	if _, keySHA, _, ok := licensegate.ParseStagingKey(p.InputKey); ok && keySHA != "" {
		actual, err := sha256File(localInput)
		if err != nil {
			return fail(fmt.Sprintf("hash input: %v", err), false)
		}
		if !strings.EqualFold(actual, keySHA) {
			return fail(fmt.Sprintf(
				"무결성 검증 실패: 업로드 파일이 신고된 SHA-256과 다릅니다 (staging=%s actual=%s)",
				keySHA, actual), true)
		}
	}

	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	probe, err := audio.RunFFprobe(probeCtx, localInput)
	cancel()
	if err != nil {
		return fail(fmt.Sprintf("ffprobe: %v", err), false)
	}
	if keyword, blocked := audio.CheckBlockedMetadata(probe.Tags, policy.BlockedMetadataKeywords); blocked {
		reason := fmt.Sprintf("차단된 메타데이터 키워드 감지: %s", keyword)
		_ = w.repos.Rights.SetGate3Report(ctx, track.ID, false, audio.FormatFFprobeReportJSON(probe, false, reason))
		return fail(reason, true)
	}
	if err := audio.CheckBitrate(probe.Ext, probe.CodecName, probe.BitrateKbps, policy.MinBitrateKbps); err != nil {
		reason := fmt.Sprintf("비트레이트 기준 미달: %v", err)
		_ = w.repos.Rights.SetGate3Report(ctx, track.ID, false, audio.FormatFFprobeReportJSON(probe, false, reason))
		return fail(reason, true)
	}
	_ = w.repos.Rights.SetGate3Report(ctx, track.ID, true, audio.FormatFFprobeReportJSON(probe, true, ""))

	outDir := filepath.Join(tmpDir, "hls")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fail(fmt.Sprintf("output dir: %v", err), false)
	}
	if planTier == "" {
		planTier = "commercial"
	}
	encodeCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	encodeErr := w.runEncode(encodeCtx, localInput, outDir, audio.AIWatermarkComment, track.AIProvider, planTier)
	cancel()
	if encodeErr != nil {
		return fail(fmt.Sprintf("ffmpeg hls: %v", encodeErr), false)
	}

	var wavePtr, fpPtr *string
	waveCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	peaks, waveErr := audio.ExtractWaveform(waveCtx, localInput, 100)
	cancel()
	if waveErr != nil {
		log.Printf("waveform skipped for track %s: %v", track.ID, waveErr)
	} else {
		if fp := audio.ComputeAudioFingerprint(peaks); fp != "" {
			fpPtr = &fp
		}
		if waveJSON, err := audio.WaveformJSON(peaks); err == nil {
			wavePtr = &waveJSON
			_ = os.WriteFile(filepath.Join(outDir, "peaks.json"), []byte(waveJSON), 0o644)
		}
	}

	if err := w.uploadDir(ctx, outDir, p.OutputPrefix); err != nil {
		return fail(fmt.Sprintf("upload hls: %v", err), false)
	}

	bitrate := probe.BitrateKbps
	if bitrate <= 0 {
		bitrate = 256
	}
	var durationMs *int
	if probe.DurationSec > 0 {
		d := int(probe.DurationSec * 1000)
		durationMs = &d
	}
	if err := w.repos.Jobs.MarkSucceeded(ctx, job.ID); err != nil {
		return err
	}
	if err := w.repos.Tracks.SetTranscoded(ctx, track.ID, p.OutputPrefix, bitrate, durationMs, wavePtr, fpPtr); err != nil {
		return err
	}
	// Best-effort distribution metadata (WI-05): tempo/key detection must
	// never fail an otherwise good transcode.
	analysisCtx, analysisCancel := context.WithTimeout(ctx, 10*time.Minute)
	analysisPCM, analysisErr := audio.DecodeMonoForAnalysis(analysisCtx, localInput, 0)
	analysisCancel()
	if analysisErr != nil {
		log.Printf("analysis skipped for track %s: %v", track.ID, analysisErr)
	} else if analysis, err := audio.AnalyzeTrack(analysisPCM); err != nil {
		log.Printf("analysis skipped for track %s: %v", track.ID, err)
	} else if err := w.repos.Tracks.SetAnalysis(ctx, track.ID, &analysis.BPM, &analysis.MusicalKey); err != nil {
		log.Printf("analysis store failed for track %s: %v", track.ID, err)
	} else {
		log.Printf("analysis for track %s: %.1f BPM, %s", track.ID, analysis.BPM, analysis.MusicalKey)
	}
	log.Printf("job %s track %s transcoded to %s (LICENSE_GATE_PENDING awaiting Admin Gate 4)", job.ID, track.ID, p.OutputPrefix)
	return nil
}

// recheckLicense re-validates the commercial gate at transcode time so a
// track that bypassed finalize (or whose policy/subscription changed since)
// can never reach HLS packaging.
func (w *worker) recheckLicense(ctx context.Context, track *domain.Track, policy *domain.AIProviderPolicy) (string, error) {
	if !policy.AllowsStandaloneStreaming {
		msg := policy.RejectionMessageKo
		if msg == "" {
			msg = "provider license does not allow standalone streaming"
		}
		return "", fmt.Errorf("상업 라이선스 재검증 실패: %s", msg)
	}
	rights, err := w.repos.Rights.GetByTrack(ctx, track.ID)
	if err != nil {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 권리선언 누락: %v", err)
	}
	if rights.SourceModel != nil {
		if err := licensegate.ValidateSourceModel(*rights.SourceModel); err != nil {
			return "", fmt.Errorf("상업 라이선스 재검증 실패: %v", err)
		}
	}
	if rights.SubscriptionID == nil || *rights.SubscriptionID == "" {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 구독 증빙 누락")
	}
	if rights.GeneratedAt == nil || rights.GeneratedAt.IsZero() {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 음원 생성일 누락")
	}
	if policy.MaxGeneratedAt != nil && !policy.MaxGeneratedAt.IsZero() && rights.GeneratedAt.After(*policy.MaxGeneratedAt) {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: %s 컷오프(%s) 이후 생성", track.AIProvider, policy.MaxGeneratedAt.Format("2006-01-02"))
	}
	sub, err := w.repos.Gate.GetSubscription(ctx, track.PartnerID, track.AIProvider, *rights.SubscriptionID)
	if err != nil {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 구독 조회 오류: %v", err)
	}
	if sub == nil || sub.Status != domain.SubVerified {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 검증된 유료 구독이 아님")
	}
	start := sub.CreatedAt
	if sub.VerifiedAt != nil && !sub.VerifiedAt.IsZero() {
		start = *sub.VerifiedAt
	}
	if rights.GeneratedAt.Before(start) {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 구독 기간 이전 생성(소급 불가)")
	}
	if sub.ExpiresAt != nil && !sub.ExpiresAt.IsZero() && rights.GeneratedAt.After(*sub.ExpiresAt) {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 구독 기간 외 생성")
	}
	tier := rights.PlanTierUsed
	if tier == nil || *tier == "" {
		tier = &sub.PlanName
	}
	snap, err := licensegate.BuildCertSnapshot(licensegate.CertSnapshot{
		Provider:              track.AIProvider,
		PlanTierUsed:          tier,
		SubscriptionID:        rights.SubscriptionID,
		GeneratedAt:           rights.GeneratedAt,
		SourceModel:           rights.SourceModel,
		CollectiveTrustStatus: rights.CollectiveTrustStatus,
		Gate1PolicyPassed:     true,
		Gate2DatePassed:       true,
	})
	if err != nil {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 인증 스냅샷 생성 오류: %v", err)
	}
	if err := w.repos.Rights.SetCertSnapshot(ctx, track.ID, snap, time.Now().UTC()); err != nil {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 인증 스냅샷 저장 오류: %v", err)
	}
	return sub.PlanName, nil
}

func (w *worker) runEncode(ctx context.Context, input, outDir, comment, provider, tier string) error {
	cmd := audio.BuildHLSCommand(ctx, input, outDir, comment, provider, tier)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (w *worker) download(ctx context.Context, bucket, key, dest string) error {
	signed, err := w.storage.PresignGet(ctx, bucket, key, 15*time.Minute)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, signed, nil)
	if err != nil {
		return err
	}
	resp, err := w.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download status %d", resp.StatusCode)
	}
	if resp.ContentLength > maxStagingBytes {
		return fmt.Errorf("staged object too large: %d bytes (max %d)", resp.ContentLength, maxStagingBytes)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	// Bound the copy even when Content-Length is missing or lies.
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxStagingBytes+1))
	if err != nil {
		return err
	}
	if n > maxStagingBytes {
		_ = os.Remove(dest)
		return fmt.Errorf("staged object too large (max %d bytes)", maxStagingBytes)
	}
	return nil
}

// sha256File streams a file through SHA-256 without loading it into memory.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (w *worker) uploadDir(ctx context.Context, dir, prefix string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		key := strings.Trim(prefix, "/") + "/" + e.Name()
		signed, err := w.storage.PresignPut(ctx, w.hlsBucket, key, 15*time.Minute)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, signed, bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.ContentLength = int64(len(data))
		resp, err := w.http.Do(req)
		if err != nil {
			return err
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("upload %s: status %d", key, resp.StatusCode)
		}
	}
	return nil
}
