// Command worker polls the Redis transcode queue and runs the audio processor.
package main

import (
	"bytes"
	"context"
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
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
)

const maxAttempts = 3

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
	w := &worker{repos: repos, storage: st, hlsBucket: buckets.PublicHLS, http: &http.Client{Timeout: 5 * time.Minute}}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Printf("worker polling queue %q", repository.TranscodeQueueKey)
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
	repos     *repository.Repos
	storage   storage.Storage
	hlsBucket string
	http      *http.Client
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
	if rights.SubscriptionID == nil || *rights.SubscriptionID == "" {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 구독 증빙 누락")
	}
	if rights.GeneratedAt == nil || rights.GeneratedAt.IsZero() {
		return "", fmt.Errorf("상업 라이선스 재검증 실패: 음원 생성일 누락")
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
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
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
