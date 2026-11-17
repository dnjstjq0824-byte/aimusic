package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
)

type fakeVerifyUsers struct {
	byID     map[string]*domain.User
	verified []string
}

func (f *fakeVerifyUsers) GetByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeVerifyUsers) MarkEmailVerified(_ context.Context, id string) error {
	f.verified = append(f.verified, id)
	if u, ok := f.byID[id]; ok {
		now := time.Now()
		u.EmailVerifiedAt = &now
	}
	return nil
}

type fakeVerifyTokens struct {
	byHash        map[string]*domain.EmailVerification
	created       int
	deletedUsers  []string
	deletedEmails []string
}

func (f *fakeVerifyTokens) Create(_ context.Context, userID, email, purpose, hash, ref string, exp time.Time) (*domain.EmailVerification, error) {
	if f.byHash == nil {
		f.byHash = map[string]*domain.EmailVerification{}
	}
	v := &domain.EmailVerification{ID: "v-" + hash[:8], Email: email, Purpose: purpose,
		TokenHash: hash, ExpiresAt: exp}
	if userID != "" {
		u := userID
		v.UserID = &u
	}
	if ref != "" {
		r := ref
		v.RefToken = &r
	}
	f.byHash[hash] = v
	f.created++
	return v, nil
}

func (f *fakeVerifyTokens) GetByHash(_ context.Context, hash string) (*domain.EmailVerification, error) {
	v, ok := f.byHash[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return v, nil
}

func (f *fakeVerifyTokens) MarkUsed(_ context.Context, id string) error {
	for _, v := range f.byHash {
		if v.ID == id {
			if v.UsedAt != nil {
				return domain.ErrNotFound
			}
			now := time.Now()
			v.UsedAt = &now
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeVerifyTokens) DeletePendingForUser(_ context.Context, userID string) error {
	f.deletedUsers = append(f.deletedUsers, userID)
	for h, v := range f.byHash {
		if v.UserID != nil && *v.UserID == userID && v.UsedAt == nil {
			delete(f.byHash, h)
		}
	}
	return nil
}

func (f *fakeVerifyTokens) DeletePendingForEmail(_ context.Context, email, purpose string) error {
	f.deletedEmails = append(f.deletedEmails, email+"|"+purpose)
	for h, v := range f.byHash {
		if v.Email == email && v.Purpose == purpose && v.UsedAt == nil {
			delete(f.byHash, h)
		}
	}
	return nil
}

type fakeIssuer struct {
	issued [][2]string
}

func (f *fakeIssuer) Issue(_ context.Context, userID, role string) (*Tokens, error) {
	f.issued = append(f.issued, [2]string{userID, role})
	return &Tokens{AccessToken: "access-" + userID, RefreshToken: "refresh-" + userID, ExpiresIn: 900}, nil
}

type fakeOAuthSignup struct {
	users map[string]*domain.User
	calls [][2]string
	err   error
}

func (f *fakeOAuthSignup) FinishVerifiedSignup(_ context.Context, emailToken, email string) (*domain.User, error) {
	f.calls = append(f.calls, [2]string{emailToken, email})
	if f.err != nil {
		return nil, f.err
	}
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	u := &domain.User{ID: "u-oauth", Email: email, Role: domain.RoleUser, IsActive: true}
	if f.users == nil {
		f.users = map[string]*domain.User{}
	}
	f.users[email] = u
	return u, nil
}

type verifyFixture struct {
	s      *EmailVerifyService
	users  *fakeVerifyUsers
	tokens *fakeVerifyTokens
	mailer *fakeMailer
	issuer *fakeIssuer
	oauth  *fakeOAuthSignup
	now    time.Time
}

func verifyTestService() *verifyFixture {
	fx := &verifyFixture{
		users:  &fakeVerifyUsers{byID: map[string]*domain.User{}},
		tokens: &fakeVerifyTokens{},
		mailer: &fakeMailer{},
		issuer: &fakeIssuer{},
		oauth:  &fakeOAuthSignup{},
		now:    time.Now(),
	}
	fx.s = &EmailVerifyService{
		users: fx.users, tokens: fx.tokens, mailer: fx.mailer,
		issuer: fx.issuer, oauth: fx.oauth,
		ttl: 24 * time.Hour, linkBase: "https://app.example.test",
		now: func() time.Time { return fx.now },
	}
	return fx
}

func TestHashVerifyTokenStable(t *testing.T) {
	a, b := HashVerifyToken("abc"), HashVerifyToken("abc")
	if a != b || len(a) != 64 {
		t.Fatalf("hash unstable: %q", a)
	}
	if HashVerifyToken("abd") == a {
		t.Fatalf("hash collision across inputs")
	}
}

func TestBuildVerifyLink(t *testing.T) {
	got := BuildVerifyLink("https://app.example.test/", "tok-1")
	if got != "https://app.example.test/verify-email?token=tok-1" {
		t.Fatalf("link = %q", got)
	}
}

func TestGenerateVerifyTokenRandom(t *testing.T) {
	a, err := GenerateVerifyToken()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	b, _ := GenerateVerifyToken()
	if len(a) != 64 || a == b {
		t.Fatalf("weak token: %q", a)
	}
}

func TestVerifyRequestMailsLocaleLink(t *testing.T) {
	for _, tc := range []struct {
		locale  string
		subject string
	}{
		{"ko", "이메일 인증"},
		{"en", "Verify your email"},
		{"ja", "メールアドレスの確認"},
		{"zh-cn", "验证您的电子邮箱"},
		{"th", "ยืนยันอีเมลของคุณ"},
	} {
		fx := verifyTestService()
		fx.users.byID["u-1"] = &domain.User{ID: "u-1", Email: "u@example.com", IsActive: true, Locale: tc.locale}
		if err := fx.s.Request(context.Background(), "u-1"); err != nil {
			t.Fatalf("locale %s request: %v", tc.locale, err)
		}
		if fx.tokens.created != 1 || len(fx.mailer.to) != 1 {
			t.Fatalf("locale %s side effects: created=%d mailed=%v", tc.locale, fx.tokens.created, fx.mailer.to)
		}
		body := strings.Join(fx.mailer.bodies, "\n")
		if !strings.Contains(body, "/verify-email?token=") || !strings.Contains(body, tc.subject) {
			t.Fatalf("locale %s mail missing link/subject: %q", tc.locale, body)
		}
	}
}

func TestVerifyRequestSkipsVerified(t *testing.T) {
	fx := verifyTestService()
	stamped := fx.now.Add(-time.Hour)
	fx.users.byID["u-1"] = &domain.User{ID: "u-1", Email: "u@example.com", IsActive: true, Locale: "ko", EmailVerifiedAt: &stamped}
	if err := fx.s.Request(context.Background(), "u-1"); err != nil {
		t.Fatalf("request: %v", err)
	}
	if fx.tokens.created != 0 || len(fx.mailer.to) != 0 {
		t.Fatalf("verified account mailed: created=%d mailed=%v", fx.tokens.created, fx.mailer.to)
	}
}

func TestVerifyRequestUnknownUser(t *testing.T) {
	fx := verifyTestService()
	if err := fx.s.Request(context.Background(), "ghost"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if err := fx.s.Request(context.Background(), ""); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("empty user err = %v", err)
	}
}

func seedVerifyToken(t *testing.T, fx *verifyFixture, raw, userID, email, purpose, ref string, exp time.Time) {
	t.Helper()
	if _, err := fx.tokens.Create(context.Background(), userID, email, purpose, HashVerifyToken(raw), ref, exp); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestVerifyConfirmAccountHappy(t *testing.T) {
	fx := verifyTestService()
	fx.users.byID["u-1"] = &domain.User{ID: "u-1", Email: "u@example.com", Role: "USER", IsActive: true, Locale: "ko"}
	seedVerifyToken(t, fx, "raw-1", "u-1", "u@example.com", domain.VerifyPurposeAccount, "", fx.now.Add(time.Hour))
	res, err := fx.s.Confirm(context.Background(), "raw-1")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if res.User.ID != "u-1" || res.Tokens.AccessToken == "" {
		t.Fatalf("result = %+v", res)
	}
	if len(fx.users.verified) != 1 || res.User.EmailVerifiedAt == nil {
		t.Fatalf("not stamped verified: %v", fx.users.verified)
	}
	if len(fx.issuer.issued) != 1 || fx.issuer.issued[0] != [2]string{"u-1", "USER"} {
		t.Fatalf("issued = %v", fx.issuer.issued)
	}
	// Replay of a consumed link must fail.
	if _, err := fx.s.Confirm(context.Background(), "raw-1"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("replay err = %v", err)
	}
}

func TestVerifyConfirmRejectsBadTokens(t *testing.T) {
	fx := verifyTestService()
	ctx := context.Background()
	if _, err := fx.s.Confirm(ctx, "unknown"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("unknown token err = %v", err)
	}
	if _, err := fx.s.Confirm(ctx, ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("empty token err = %v", err)
	}
	fx.users.byID["u-1"] = &domain.User{ID: "u-1", Email: "u@example.com", IsActive: true}
	seedVerifyToken(t, fx, "expired", "u-1", "u@example.com", domain.VerifyPurposeAccount, "", fx.now.Add(-time.Hour))
	if _, err := fx.s.Confirm(ctx, "expired"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expired token err = %v", err)
	}
}

func TestVerifyConfirmInactiveRefused(t *testing.T) {
	fx := verifyTestService()
	fx.users.byID["u-1"] = &domain.User{ID: "u-1", Email: "u@example.com", IsActive: false}
	seedVerifyToken(t, fx, "raw-1", "u-1", "u@example.com", domain.VerifyPurposeAccount, "", fx.now.Add(time.Hour))
	if _, err := fx.s.Confirm(context.Background(), "raw-1"); !errors.Is(err, domain.ErrUserInactive) {
		t.Fatalf("err = %v", err)
	}
	if len(fx.issuer.issued) != 0 {
		t.Fatalf("tokens issued for inactive account: %v", fx.issuer.issued)
	}
}

func TestVerifyOAuthSignupRequestAndConfirm(t *testing.T) {
	fx := verifyTestService()
	ctx := context.Background()
	if err := fx.s.RequestOAuthSignup(ctx, "New@Example.com", "em-1", "en"); err != nil {
		t.Fatalf("request: %v", err)
	}
	if fx.tokens.created != 1 || len(fx.mailer.to) != 1 || fx.mailer.to[0] != "new@example.com" {
		t.Fatalf("side effects: created=%d mailed=%v", fx.tokens.created, fx.mailer.to)
	}
	if !strings.Contains(strings.Join(fx.mailer.bodies, "\n"), "Verify your email") {
		t.Fatalf("mail not in en template: %q", fx.mailer.bodies)
	}
	// Find the issued raw token via the stored hash is impossible by design;
	// exercise confirm through a seeded row carrying the same ref.
	fx2 := verifyTestService()
	fx2.oauth = fx.oauth
	fx2.s.oauth = fx.oauth
	seedVerifyToken(t, fx2, "raw-oauth", "", "new@example.com", domain.VerifyPurposeOAuthSignup, "em-1", fx2.now.Add(time.Hour))
	res, err := fx2.s.Confirm(ctx, "raw-oauth")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if res.User.Email != "new@example.com" || res.Tokens.AccessToken == "" {
		t.Fatalf("result = %+v", res)
	}
	if len(fx.oauth.calls) != 1 || fx.oauth.calls[0] != [2]string{"em-1", "new@example.com"} {
		t.Fatalf("oauth calls = %v", fx.oauth.calls)
	}
	if len(fx2.users.verified) != 1 {
		t.Fatalf("verified = %v", fx2.users.verified)
	}
}

func TestVerifyOAuthSignupRejectsBadInput(t *testing.T) {
	fx := verifyTestService()
	ctx := context.Background()
	if err := fx.s.RequestOAuthSignup(ctx, "not-an-email", "em-1", "ko"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("bad email err = %v", err)
	}
	if err := fx.s.RequestOAuthSignup(ctx, "a@example.com", "", "ko"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("empty ref err = %v", err)
	}
}

func TestVerifyConfirmOAuthSignupFailure(t *testing.T) {
	fx := verifyTestService()
	fx.oauth.err = domain.ErrInvalidCredentials
	seedVerifyToken(t, fx, "raw-oauth", "", "new@example.com", domain.VerifyPurposeOAuthSignup, "em-stale", fx.now.Add(time.Hour))
	if _, err := fx.s.Confirm(context.Background(), "raw-oauth"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("err = %v", err)
	}
	if len(fx.issuer.issued) != 0 {
		t.Fatalf("tokens issued on failed signup: %v", fx.issuer.issued)
	}
}
