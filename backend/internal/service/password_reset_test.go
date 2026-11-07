package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ai-music-platform/backend/internal/domain"
)

type fakeResetUsers struct {
	byEmail map[string]*domain.User
	hashes  map[string]string
}

func (f *fakeResetUsers) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeResetUsers) UpdatePasswordHash(_ context.Context, id, hash string) error {
	if f.hashes == nil {
		f.hashes = map[string]string{}
	}
	f.hashes[id] = hash
	return nil
}

type fakeResetTokens struct {
	byHash  map[string]*domain.PasswordResetToken
	created int
	deleted []string
}

func (f *fakeResetTokens) Create(_ context.Context, userID, hash string, exp time.Time) (*domain.PasswordResetToken, error) {
	if f.byHash == nil {
		f.byHash = map[string]*domain.PasswordResetToken{}
	}
	t := &domain.PasswordResetToken{ID: "t-" + hash[:8], UserID: userID, TokenHash: hash, ExpiresAt: exp}
	f.byHash[hash] = t
	f.created++
	return t, nil
}

func (f *fakeResetTokens) GetByHash(_ context.Context, hash string) (*domain.PasswordResetToken, error) {
	t, ok := f.byHash[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (f *fakeResetTokens) MarkUsed(_ context.Context, id string) error {
	for _, t := range f.byHash {
		if t.ID == id {
			if t.UsedAt != nil {
				return domain.ErrNotFound
			}
			now := time.Now()
			t.UsedAt = &now
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeResetTokens) DeleteForUser(_ context.Context, userID string) error {
	f.deleted = append(f.deleted, userID)
	for h, t := range f.byHash {
		if t.UserID == userID && t.UsedAt == nil {
			delete(f.byHash, h)
		}
	}
	return nil
}

type fakeMailer struct {
	to      []string
	subject []string
	bodies  []string
}

func (f *fakeMailer) Send(_ context.Context, to, subject, body string) error {
	f.to = append(f.to, to)
	f.subject = append(f.subject, body)
	f.bodies = append(f.bodies, subject+body)
	return nil
}

func resetTestService(users *fakeResetUsers, tokens *fakeResetTokens, mailer *fakeMailer) *PasswordResetService {
	return &PasswordResetService{
		users: users, tokens: tokens, mailer: mailer,
		ttl: 30 * time.Minute, linkBase: "https://app.example.test", now: time.Now,
	}
}

func TestHashResetTokenStable(t *testing.T) {
	a, b := HashResetToken("abc"), HashResetToken("abc")
	if a != b || len(a) != 64 {
		t.Fatalf("hash unstable: %q", a)
	}
	if HashResetToken("abd") == a {
		t.Fatalf("hash collision across inputs")
	}
}

func TestBuildResetLink(t *testing.T) {
	got := BuildResetLink("https://app.example.test/", "tok-1")
	if got != "https://app.example.test/reset-password?token=tok-1" {
		t.Fatalf("link = %q", got)
	}
}

func TestGenerateResetTokenRandom(t *testing.T) {
	a, err := GenerateResetToken()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	b, _ := GenerateResetToken()
	if len(a) != 64 || a == b {
		t.Fatalf("weak token: %q", a)
	}
}

func TestRequestIssuesLinkForActiveUser(t *testing.T) {
	users := &fakeResetUsers{byEmail: map[string]*domain.User{
		"u@example.com": {ID: "u-1", Email: "u@example.com", IsActive: true},
	}}
	tokens := &fakeResetTokens{}
	mailer := &fakeMailer{}
	s := resetTestService(users, tokens, mailer)
	if err := s.Request(context.Background(), " U@Example.com "); err != nil {
		t.Fatalf("request: %v", err)
	}
	if tokens.created != 1 || len(mailer.to) != 1 || mailer.to[0] != "u@example.com" {
		t.Fatalf("side effects: created=%d mailed=%v", tokens.created, mailer.to)
	}
	if !strings.Contains(strings.Join(mailer.bodies, "\n"), "/reset-password?token=") {
		t.Fatalf("mail missing reset link")
	}
}

func TestRequestDoesNotEnumerate(t *testing.T) {
	users := &fakeResetUsers{byEmail: map[string]*domain.User{
		"off@example.com": {ID: "u-2", Email: "off@example.com", IsActive: false},
	}}
	tokens := &fakeResetTokens{}
	mailer := &fakeMailer{}
	s := resetTestService(users, tokens, mailer)
	for _, email := range []string{"missing@example.com", "off@example.com"} {
		if err := s.Request(context.Background(), email); err != nil {
			t.Fatalf("request(%s): %v", email, err)
		}
	}
	if tokens.created != 0 || len(mailer.to) != 0 {
		t.Fatalf("enumeration leak: created=%d mailed=%v", tokens.created, mailer.to)
	}
}

func TestConfirmResetsPasswordOnce(t *testing.T) {
	users := &fakeResetUsers{hashes: map[string]string{}}
	tokens := &fakeResetTokens{}
	mailer := &fakeMailer{}
	s := resetTestService(users, tokens, mailer)
	raw := "raw-token-1"
	if _, err := tokens.Create(context.Background(), "u-1", HashResetToken(raw), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := s.Confirm(context.Background(), raw, "new-password-1"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	hash := users.hashes["u-1"]
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("new-password-1")); err != nil {
		t.Fatalf("hash not updated: %v", err)
	}
	// Replay of a consumed token must fail.
	if err := s.Confirm(context.Background(), raw, "another-password-1"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("replay err = %v", err)
	}
}

func TestConfirmRejectsBadTokens(t *testing.T) {
	s := resetTestService(&fakeResetUsers{}, &fakeResetTokens{}, &fakeMailer{})
	ctx := context.Background()
	if err := s.Confirm(ctx, "unknown", "valid-password-1"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("unknown token err = %v", err)
	}
	tokens := &fakeResetTokens{}
	s2 := resetTestService(&fakeResetUsers{hashes: map[string]string{}}, tokens, &fakeMailer{})
	if _, err := tokens.Create(ctx, "u-1", HashResetToken("expired"), time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := s2.Confirm(ctx, "expired", "valid-password-1"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expired token err = %v", err)
	}
	if err := s2.Confirm(ctx, "", "short"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("bad input err = %v", err)
	}
}
