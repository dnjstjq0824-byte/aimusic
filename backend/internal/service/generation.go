package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

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

// BannedVoicePhrases extends the voice-mimic filter to all 5 locales (G5).
// Substring match over normalized text (phrases are long enough to be safe).
var BannedVoicePhrases = []string{
	"clone the voice", "mimic the voice", "copy her voice", "copy his voice",
	"목소리를 복제", "목소리 복제", "목소리를 흉내", "음성을 복제",
	"声真似", "声まね", "ものまね", "歌声を真似", "なりすまし", "声を複製",
	"模仿声音", "声音克隆", "克隆声音", "冒充歌手", "模仿歌声",
	"เลียนแบบเสียง", "โคลนเสียง", "แอบอ้าง",
}

// BannedStylePatterns blocks "~ in the style of X" imitation constructions
// in all 5 locales (G5). The artist name itself need not be enumerated: the
// construction is the signal. Substring match over normalized text.
var BannedStylePatterns = []string{
	"in the style of", "in style of",
	"스타일로", "스타일으로", "풍으로", "풍의 노래", "창법으로",
	"風に", "風の", "っぽく", "スタイルで",
	"风格唱", "风格演", "模仿唱",
	"สไตล์", "แนวเพลงของ",
}

// BannedArtistNames is the curation seed for real-artist-name blocking (G5,
// extended WI-N13): unmistakable names in canonical + local spellings,
// matched with word boundaries so short names never hit inside other words
// ("IU" must not match "triumph"). Deliberately EXCLUDED (over-blocking):
// common nouns doubling as stage names (queen/정국/태양/대성/윈터/슈가),
// single-syllable names (뷔), and song titles (ordinary words — block via
// GENERATION_BANNED_KEYWORDS per case instead). Extend via
// GENERATION_BANNED_KEYWORDS or a curation PR (PO/legal review: names
// change and over-blocking harms legit prompts).
var BannedArtistNames = []string{
	"bts", "blackpink", "newjeans", "bigbang", "twice",
	"taylor swift", "drake", "weeknd", "ed sheeran", "adele", "iu",
	"beatles", "michael jackson", "madonna", "beyonce",
	"coldplay", "rihanna", "lady gaga", "justin bieber", "ariana grande",
	"bruno mars", "billie eilish", "eminem", "sia", "sam smith",
	"dua lipa", "olivia rodrigo", "yoasobi",
	"방탄소년단", "방탄", "블랙핑크", "뉴진스", "빅뱅", "트와이스",
	"테일러 스위프트", "아이유",
	"세븐틴", "스트레이키즈", "에스파", "아이브", "르세라핌", "엔시티",
	"엑소", "샤이니", "슈퍼주니어", "소녀시대", "지드래곤", "지민",
	"제이홉", "카리나", "원영", "리사", "지수", "로제",
	"米津玄師", "あいみょん", "宇多田ヒカル",
	"周杰伦",
}

// normalizeGuardText folds full-width ASCII (U+FF01–U+FF5E), ideographic
// spaces, case, and whitespace runs so spacing/trick variants of a banned
// phrase still match (G5).
func normalizeGuardText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E:
			r -= 0xFEE0
		case r == 0x3000:
			r = ' '
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !space && b.Len() > 0 {
				b.WriteByte(' ')
				space = true
			}
			continue
		}
		space = false
		b.WriteRune(r)
	}
	return strings.ToLower(strings.TrimSpace(b.String()))
}

// isCJKName reports whether name is all CJK (Han/Hiragana/Katakana/
// Hangul). CJK names match by substring: CJK text has no word boundaries
// ("米津玄師みたいな" must match, "지민처럼" must match despite the
// particle). Curators must keep 1-2 rune CJK names unmistakable (no
// common nouns like 정국/태양 — see BannedArtistNames).
func isCJKName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !(unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
			unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r)) {
			return false
		}
	}
	return true
}

// containsWord reports whether word occurs with word boundaries (letters and
// digits glue; anything else splits). Both inputs must be normalized.
func containsWord(haystack, word string) bool {
	if word == "" {
		return false
	}
	isWord := func(r rune) bool {
		return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	for i := 0; i+len(word) <= len(haystack); {
		j := strings.Index(haystack[i:], word)
		if j < 0 {
			return false
		}
		start := i + j
		end := start + len(word)
		leftOK := start == 0
		if !leftOK {
			r, _ := utf8.DecodeLastRuneInString(haystack[:start])
			leftOK = !isWord(r)
		}
		rightOK := end == len(haystack)
		if !rightOK {
			r, _ := utf8.DecodeRuneInString(haystack[end:])
			rightOK = !isWord(r)
		}
		if leftOK && rightOK {
			return true
		}
		i = start + 1
	}
	return false
}

// generationStore is the generation persistence surface.
type generationStore interface {
	CreateFull(ctx context.Context, partnerID, prompt string, lyrics *string, model, licenseRef, paramsJSON string, seed *int64, modelVersion string) (*domain.GenerationJob, error)
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
		return nil, fmt.Errorf("%w: prompt 1..2000 chars", domain.ErrInvalidInput)
	}
	var lyrics *string
	lyricsText := ""
	if in.Lyrics != nil {
		l := strings.TrimSpace(*in.Lyrics)
		if len([]rune(l)) > MaxGenerationLyricsChars {
			return nil, fmt.Errorf("%w: lyrics at most 10000 chars", domain.ErrInvalidInput)
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
		return nil, fmt.Errorf("%w: model at most 64 chars", domain.ErrInvalidInput)
	}
	licenseRef := strings.TrimSpace(in.LicenseRef)
	if len(licenseRef) > MaxGenerationLicenseRef {
		return nil, fmt.Errorf("%w: licenseRef at most 256 chars", domain.ErrInvalidInput)
	}
	now := s.clock()
	// Banned-prompt filter first: strikes accrue even when quota is open.
	if keyword := matchBannedKeyword(prompt+"\n"+lyricsText, s.bannedExtra); keyword != "" {
		_ = s.store.RecordViolation(ctx, in.PartnerID, keyword)
		recent, _ := s.store.CountRecentViolations(ctx, in.PartnerID, now.Add(-GenerationStrikeWindow))
		if recent >= GenerationStrikesToSuspend {
			return nil, fmt.Errorf("%w: %d banned-prompt strikes suspend generation for 24h (no impersonation or voice cloning)", domain.ErrForbidden, recent)
		}
		return nil, fmt.Errorf("%w: banned prompt detected (%s, warning %d/%d; strikes suspend for 24h)", domain.ErrInvalidInput, keyword, recent, GenerationStrikesToSuspend)
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	used, err := s.store.CountTodayByPartner(ctx, in.PartnerID, dayStart)
	if err != nil {
		return nil, err
	}
	if used >= int64(s.dailyCap) {
		return nil, fmt.Errorf("%w: daily generation cap exceeded (%d/day)", domain.ErrForbidden, s.dailyCap)
	}
	// Reproducibility pins ride in params (WI-G0): the worker forwards them
	// to the contract-v2 backend and records the served values on the job.
	var seed *int64
	modelVersion := ""
	if v, ok := in.Params["seed"]; ok {
		n, ok := paramsInt64(v)
		if !ok || n < 0 {
			return nil, fmt.Errorf("%w: params.seed must be an integer >= 0", domain.ErrInvalidInput)
		}
		seed = &n
	}
	if v, ok := in.Params["modelVersion"]; ok {
		mv, ok := v.(string)
		if !ok || len(mv) > 64 {
			return nil, fmt.Errorf("%w: params.modelVersion must be a string at most 64 chars", domain.ErrInvalidInput)
		}
		modelVersion = strings.TrimSpace(mv)
	}
	paramsJSON := "{}"
	if len(in.Params) > 0 {
		raw, err := json.Marshal(in.Params)
		if err != nil || len(raw) > 4096 {
			return nil, fmt.Errorf("%w: params", domain.ErrInvalidInput)
		}
		paramsJSON = string(raw)
	}
	job, err := s.store.CreateFull(ctx, in.PartnerID, prompt, lyrics, model, licenseRef, paramsJSON, seed, modelVersion)
	if err != nil {
		return nil, err
	}
	if err := s.queue.EnqueueGeneration(ctx, repository.GenerationPayload{JobID: job.ID}); err != nil {
		return nil, err
	}
	return job, nil
}

// matchBannedKeyword returns the first blocked signal found ("" when clean).
// Text is normalized once (full-width/case/space folding, G5); voice phrases
// and style constructions match by substring, artist names by word.
func matchBannedKeyword(text string, extra []string) string {
	norm := normalizeGuardText(text)
	for _, kw := range BannedPromptKeywords {
		if strings.Contains(norm, normalizeGuardText(kw)) {
			return kw
		}
	}
	for _, kw := range BannedVoicePhrases {
		if strings.Contains(norm, normalizeGuardText(kw)) {
			return kw
		}
	}
	for _, kw := range BannedStylePatterns {
		if strings.Contains(norm, normalizeGuardText(kw)) {
			return kw
		}
	}
	for _, name := range BannedArtistNames {
		nn := normalizeGuardText(name)
		if isCJKName(nn) {
			if strings.Contains(norm, nn) {
				return name
			}
			continue
		}
		if containsWord(norm, nn) {
			return name
		}
	}
	for _, kw := range extra {
		kw = strings.TrimSpace(kw)
		if kw != "" && strings.Contains(norm, normalizeGuardText(kw)) {
			return kw
		}
	}
	return ""
}

// paramsInt64 decodes a JSON-number-ish param value (float64 from encoding/json,
// plus int/int64/json.Number for direct callers).
func paramsInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		if n != float64(int64(n)) {
			return 0, false
		}
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		n64, err := n.Int64()
		return n64, err == nil
	}
	return 0, false
}

// SafetyVerdict is the worker-recorded contract-v2 safety outcome: the pass
// flag plus the check detail the AVIBE gate branch trusts (WI-G0/G1).
type SafetyVerdict struct {
	Passed bool     `json:"passed"`
	Checks []string `json:"checks,omitempty"`
}

// AvibeProvenanceOK reports whether a generation job carries the evidence the
// AVIBE finalize branch requires: SUCCEEDED, a recorded watermark, and a
// passing safety verdict whose checks include both "similarity" (catalog /
// training-set memorization screen) and "watermark" (insertion proof).
// Anything unparsable fails closed — and until the fingerprint screen lands
// in the worker (G2 PoC graduates it), no job can satisfy the similarity
// leg, so AVIBE finalize stays impossible by construction (G-MODEL).
func AvibeProvenanceOK(job *domain.GenerationJob) bool {
	if job == nil || job.Status != domain.JobSucceeded || strings.TrimSpace(job.WatermarkID) == "" {
		return false
	}
	var v SafetyVerdict
	if err := json.Unmarshal([]byte(job.SafetyResult), &v); err != nil || !v.Passed {
		return false
	}
	seen := map[string]bool{}
	for _, c := range v.Checks {
		seen[strings.ToLower(strings.TrimSpace(c))] = true
	}
	return seen["similarity"] && seen["watermark"]
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
