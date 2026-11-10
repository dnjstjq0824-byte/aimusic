package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

type fakeGenStore struct {
	jobs       []*domain.GenerationJob
	violations []time.Time
	now        time.Time
}

func (f *fakeGenStore) GetByID(_ context.Context, id string) (*domain.GenerationJob, error) {
	for _, j := range f.jobs {
		if j.ID == id {
			return j, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeGenStore) ListByPartner(_ context.Context, partnerID string, _ int) ([]*domain.GenerationJob, error) {
	out := []*domain.GenerationJob{}
	for _, j := range f.jobs {
		if j.PartnerID == partnerID {
			out = append(out, j)
		}
	}
	return out, nil
}

func (f *fakeGenStore) CreateFull(_ context.Context, partnerID, prompt string, lyrics *string, model, licenseRef, paramsJSON string) (*domain.GenerationJob, error) {
	g := &domain.GenerationJob{ID: "gen-1", PartnerID: partnerID, Prompt: prompt,
		Lyrics: lyrics, Model: model, LicenseRef: licenseRef, ParamsJSON: paramsJSON, Status: domain.JobQueued}
	f.jobs = append(f.jobs, g)
	return g, nil
}

func (f *fakeGenStore) CountTodayByPartner(_ context.Context, partnerID string, _ time.Time) (int64, error) {
	var n int64
	for _, j := range f.jobs {
		if j.PartnerID == partnerID {
			n++
		}
	}
	return n, nil
}

func (f *fakeGenStore) RecordViolation(_ context.Context, _ string, _ string) error {
	f.violations = append(f.violations, f.now)
	return nil
}

func (f *fakeGenStore) CountRecentViolations(_ context.Context, _ string, since time.Time) (int64, error) {
	var n int64
	for _, v := range f.violations {
		if !v.Before(since) {
			n++
		}
	}
	return n, nil
}

type fakeGenQueue struct{ n int }

func (f *fakeGenQueue) EnqueueGeneration(_ context.Context, _ repository.GenerationPayload) error {
	f.n++
	return nil
}

func testGenerationService(f *fakeGenStore, dailyCap int, extra []string) *GenerationService {
	return &GenerationService{store: f, queue: &fakeGenQueue{}, dailyCap: dailyCap, bannedExtra: extra, now: func() time.Time { return f.now }}
}

func TestGenerationSubmitHappy(t *testing.T) {
	f := &fakeGenStore{now: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)}
	svc := testGenerationService(f, 5, nil)
	lyrics := "la la"
	job, err := svc.Submit(context.Background(), GenerationSubmitInput{
		PartnerID: "partner-1", Prompt: "upbeat piano loop", Lyrics: &lyrics,
		Model: "ace-step-v1", LicenseRef: "ACE-Step release LICENSE (Apache-2.0, ops-verified)",
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if job.LicenseRef != "ACE-Step release LICENSE (Apache-2.0, ops-verified)" {
		t.Errorf("licenseRef = %q", job.LicenseRef)
	}
	if job.Prompt != "upbeat piano loop" {
		t.Errorf("prompt = %q", job.Prompt)
	}
}

func TestGenerationDailyCap(t *testing.T) {
	f := &fakeGenStore{now: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)}
	f.jobs = append(f.jobs, &domain.GenerationJob{ID: "old", PartnerID: "partner-1"})
	svc := testGenerationService(f, 1, nil)
	if _, err := svc.Submit(context.Background(), GenerationSubmitInput{PartnerID: "partner-1", Prompt: "one more"}); err == nil {
		t.Error("over-cap submit must fail")
	}
	// Other partners unaffected.
	if _, err := svc.Submit(context.Background(), GenerationSubmitInput{PartnerID: "partner-2", Prompt: "fresh"}); err != nil {
		t.Errorf("other partner: %v", err)
	}
}

func TestGenerationBannedPrompt(t *testing.T) {
	f := &fakeGenStore{now: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)}
	svc := testGenerationService(f, 100, []string{"forbidden-singer"})
	if _, err := svc.Submit(context.Background(), GenerationSubmitInput{PartnerID: "p", Prompt: "sing like forbidden-singer"}); err == nil {
		t.Error("extra keyword must be blocked")
	}
	if _, err := svc.Submit(context.Background(), GenerationSubmitInput{PartnerID: "p", Prompt: "voice clone of a famous singer"}); err == nil {
		t.Error("built-in keyword must be blocked")
	}
	if len(f.violations) != 2 {
		t.Fatalf("violations = %d", len(f.violations))
	}
	// Third strike within 24h suspends generation.
	if _, err := svc.Submit(context.Background(), GenerationSubmitInput{PartnerID: "p", Prompt: "voice clone again"}); err == nil {
		t.Fatal("third strike must fail")
	} else if got := err.Error(); !strings.Contains(got, "24시간") {
		t.Errorf("third strike must mention 24h suspension, got %q", got)
	}
	// Clean prompt still works after unrelated partner's strikes.
	if _, err := svc.Submit(context.Background(), GenerationSubmitInput{PartnerID: "other", Prompt: "calm guitar"}); err != nil {
		t.Errorf("clean prompt: %v", err)
	}
}
