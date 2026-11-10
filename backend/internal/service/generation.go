package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

// Generation limits (WI-03 setup-only + WI-08 guards).
const (
	MaxGenerationPromptChars = 2000
	MaxGenerationLyricsChars = 10000
	MaxGenerationLicenseRef  = 256
	DefaultGenerationModel   = "ace-step-v1"
	DefaultGenerationCap     = 5
	// GenerationStrikesToSuspend blocks generation for 24h after this many
	// banned-prompt strikes in the window (warn -> suspend ladder; account
	// termination stays a manual admin suspension).
	GenerationStrikesToSuspend = 3
	GenerationStrikeWindow     = 24 * time.Hour
)

// BannedPromptKeywords are built-in impersonation/voice-mimic signals
// (E-4). Matching is case-insensitive substring over prompt+lyrics.
// Operator extras append via GENERATION_BANNED_KEYWORDS (comma-separated).
var BannedPromptKeywords = []string{
	"voice clone", "voiceclone", "voice cloning",
	"impersonat", "sound-alike", "soundalike",
	"목소리 모방", "성대모사", "흉내", "닮은 목소리", "사칭",
}

// generationStore is the generation persistence surface.
type generationStore interface {
	CreateFull(ctx context.Context, partnerID, prompt string, lyrics *string, model, licenseRef, paramsJSON string) (*domain.GenerationJob, error)
	GetByID(ctx context.Context, id string) (*domain.GenerationJob, error)
	ListByPartner(ctx context.Context, partnerID string, limit int) ([]*domain.GenerationJob, error)
	CountTodayByPartner(ctx context.Context, partnerID string, day time.Time) (int64, error)
	RecordViolation(ctx context.Context, partnerID, keyword string) error
	CountRecentViolations(ctx context.Context, partnerID string, since time.Time) (int64, error)
}

// generationQueue is the generation enqueue surface.
type generationQueue interface {
	EnqueueGeneration(ctx context.Context, p repository.GenerationPayload) error
}

// GenerationService submits and reads music-generation jobs. The worker
// forwards QUEUED jobs to GENERATION_BACKEND_URL; nothing generates in-process.
type GenerationService struct {
	store       generationStore
	queue       generationQueue
	dailyCap    int
	bannedExtra []string
	now         func() time.Time
}

// NewGenerationService wires the service over live repositories.
func NewGenerationService(repos *repository.Repos, dailyCap int, bannedExtra []string) *GenerationService {
	if dailyCap <= 0 {
		dailyCap = DefaultGenerationCap
	}
	return &GenerationService{store: repos.GenJobs, queue: repos.Queue, dailyCap: dailyCap, bannedExtra: bannedExtra, now: time.Now}
}

func (s *GenerationService) clock() time.Time {
	if s.now == nil {
		return time.Now()
	}
	return s.now()
}

// GenerationSubmitInput is a validated generation request.
type GenerationSubmitInput struct {
	PartnerID  string
	Prompt     string
	Lyrics     *string
	Model      string
	LicenseRef string
	Params     map[string]any
}

// Submit validates, quota/filters, persists, and enqueues a generation job.
func (s *GenerationService) Submit(ctx context.Context, in GenerationSubmitInput) (*domain.GenerationJob, error) {
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" || len([]rune(prompt)) > MaxGenerationPromptChars {
		return nil, fmt.Errorf("%w: prompt 1..2000자", domain.ErrInvalidInput)
	}
	var lyrics *string
	lyricsText := ""
	if in.Lyrics != nil {
		l := strings.TrimSpace(*in.Lyrics)
		if len([]rune(l)) > MaxGenerationLyricsChars {
			return nil, fmt.Errorf("%w: lyrics 최대 10000자", domain.ErrInvalidInput)
		}
		if l != "" {
			lyrics = &l
			lyricsText = l
		}
	}
	model := strings.TrimSpace(in.Model)
	if model == "" {
		model = DefaultGenerationModel
	}
	if len(model) > 64 {
		return nil, fmt.Errorf("%w: model 최대 64자", domain.ErrInvalidInput)
	}
	licenseRef := strings.TrimSpace(in.LicenseRef)
	if len(licenseRef) > MaxGenerationLicenseRef {
		return nil, fmt.Errorf("%w: licenseRef 최대 256자", domain.ErrInvalidInput)
	}
	now := s.clock()
	// Banned-prompt filter first: strikes accrue even when quota is open.
	if keyword := matchBannedKeyword(prompt+"\n"+lyricsText, s.bannedExtra); keyword != "" {
		_ = s.store.RecordViolation(ctx, in.PartnerID, keyword)
		recent, _ := s.store.CountRecentViolations(ctx, in.PartnerID, now.Add(-GenerationStrikeWindow))
		if recent >= GenerationStrikesToSuspend {
			return nil, fmt.Errorf("%w: 금지 프롬프트 누적 %d회로 24시간 생성 제한됩니다 (실존 인물 사칭·목소리 모방 금지)", domain.ErrForbidden, recent)
		}
		return nil, fmt.Errorf("%w: 금지 프롬프트 감지(%s, 경고 %d/%d — 누적 시 24시간 제한)", domain.ErrInvalidInput, keyword, recent, GenerationStrikesToSuspend)
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	used, err := s.store.CountTodayByPartner(ctx, in.PartnerID, dayStart)
	if err != nil {
		return nil, err
	}
	if used >= int64(s.dailyCap) {
		return nil, fmt.Errorf("%w: 일일 생성 한도 초과(%d건/일)", domain.ErrForbidden, s.dailyCap)
	}
	paramsJSON := "{}"
	if len(in.Params) > 0 {
		raw, err := json.Marshal(in.Params)
		if err != nil || len(raw) > 4096 {
			return nil, fmt.Errorf("%w: params", domain.ErrInvalidInput)
		}
		paramsJSON = string(raw)
	}
	job, err := s.store.CreateFull(ctx, in.PartnerID, prompt, lyrics, model, licenseRef, paramsJSON)
	if err != nil {
		return nil, err
	}
	if err := s.queue.EnqueueGeneration(ctx, repository.GenerationPayload{JobID: job.ID}); err != nil {
		return nil, err
	}
	return job, nil
}

// matchBannedKeyword returns the first blocked signal found ("" when clean).
func matchBannedKeyword(text string, extra []string) string {
	lowered := strings.ToLower(text)
	for _, kw := range BannedPromptKeywords {
		if strings.Contains(lowered, strings.ToLower(kw)) {
			return kw
		}
	}
	for _, kw := range extra {
		kw = strings.TrimSpace(kw)
		if kw != "" && strings.Contains(lowered, strings.ToLower(kw)) {
			return kw
		}
	}
	return ""
}

// Get returns a job owned by the partner.
func (s *GenerationService) Get(ctx context.Context, partnerID, jobID string) (*domain.GenerationJob, error) {
	job, err := s.store.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.PartnerID != partnerID {
		return nil, domain.ErrForbidden
	}
	return job, nil
}

// List returns the partner's jobs, newest first.
func (s *GenerationService) List(ctx context.Context, partnerID string, limit int) ([]*domain.GenerationJob, error) {
	return s.store.ListByPartner(ctx, partnerID, limit)
}
