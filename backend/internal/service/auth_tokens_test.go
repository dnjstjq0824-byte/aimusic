package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
)

func testAuthService() *AuthService {
	return NewAuthService(nil, &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:     "test-access-secret-0123456789",
			RefreshSecret:    "test-refresh-secret-0123456789",
			AccessTTLMinutes: 15,
			RefreshTTLDays:   14,
		},
	})
}

func TestIssueParseAccessRoundTrip(t *testing.T) {
	s := testAuthService()
	tok, err := s.Issue(context.Background(), "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if tok.AccessToken == "" || tok.RefreshToken == "" {
		t.Fatalf("empty tokens: %+v", tok)
	}
	if tok.ExpiresIn != 15*60 {
		t.Fatalf("ExpiresIn = %d, want %d", tok.ExpiresIn, 15*60)
	}
	claims, err := s.ParseAccess(tok.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if claims.UserID != "user-1" || claims.Role != domain.RoleUser || claims.TokenType != TokenAccess {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestParseAccessRejectsRefreshToken(t *testing.T) {
	s := testAuthService()
	tok, err := s.Issue(context.Background(), "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if _, err := s.ParseAccess(tok.RefreshToken); err == nil {
		t.Fatalf("refresh token accepted as access token")
	}
}

func TestValidateAccessEnforcesVersionAndActive(t *testing.T) {
	s := testAuthService()
	user := &domain.User{ID: "user-1", Role: domain.RoleUser, IsActive: true, TokenVersion: 0}
	s.getUser = func(_ context.Context, id string) (*domain.User, error) {
		if id != user.ID {
			return nil, domain.ErrNotFound
		}
		return user, nil
	}
	tok, err := s.Issue(context.Background(), "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if _, err := s.ValidateAccess(context.Background(), tok.AccessToken); err != nil {
		t.Fatalf("fresh token rejected: %v", err)
	}
	user.TokenVersion = 1 // logout-all bump
	if _, err := s.ValidateAccess(context.Background(), tok.AccessToken); err == nil {
		t.Fatal("stale-version token accepted after logout-all")
	}
	user.TokenVersion = 0
	user.IsActive = false
	if _, err := s.ValidateAccess(context.Background(), tok.AccessToken); err == nil {
		t.Fatal("deactivated account token accepted")
	}
}

func TestParseAccessRejectsTamperedToken(t *testing.T) {
	s := testAuthService()
	tok, err := s.Issue(context.Background(), "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	tampered := tok.AccessToken[:len(tok.AccessToken)-2] + "xx"
	if _, err := s.ParseAccess(tampered); err == nil {
		t.Fatalf("tampered token accepted")
	}
	if _, err := s.ParseAccess("not-a-token"); err == nil {
		t.Fatalf("garbage token accepted")
	}
}

func TestParseAccessRejectsWrongKey(t *testing.T) {
	s := testAuthService()
	other := NewAuthService(nil, &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:     "different-access-secret-xyz",
			RefreshSecret:    "different-refresh-secret-xyz",
			AccessTTLMinutes: 15,
			RefreshTTLDays:   14,
		},
	})
	tok, err := other.Issue(context.Background(), "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if _, err := s.ParseAccess(tok.AccessToken); err == nil {
		t.Fatalf("token signed with another key accepted")
	}
}

func TestRefreshRejectsAccessToken(t *testing.T) {
	s := testAuthService()
	tok, err := s.Issue(context.Background(), "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	// Access token must fail type check before any DB lookup (repos is nil:
	// a pass here would panic, proving the gate ran first).
	if _, err := s.Refresh(context.Background(), tok.AccessToken); err == nil {
		t.Fatalf("access token accepted as refresh token")
	}
	if _, err := s.Refresh(context.Background(), "garbage"); err == nil {
		t.Fatalf("garbage refresh token accepted")
	}
}

func TestTokenTypeConstantsDistinct(t *testing.T) {
	if TokenAccess == TokenRefresh || TokenAccess == "" {
		t.Fatalf("token type constants invalid: %q/%q", TokenAccess, TokenRefresh)
	}
	if !strings.HasPrefix(TokenAccess, "access") || !strings.HasPrefix(TokenRefresh, "refresh") {
		t.Fatalf("unexpected token type values: %q/%q", TokenAccess, TokenRefresh)
	}
}

// memRefreshTracker is an in-memory refreshTracker mirroring the Redis Lua
// semantics (live -> consume once, used -> reuse, unknown -> grace).
type memRefreshTracker struct {
	mu   sync.Mutex
	live map[string]string
	used map[string]string
	fail error
}

func newMemRefreshTracker() *memRefreshTracker {
	return &memRefreshTracker{live: map[string]string{}, used: map[string]string{}}
}

func (m *memRefreshTracker) record(_ context.Context, jti, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return m.fail
	}
	m.live[jti] = userID
	return nil
}

func (m *memRefreshTracker) consume(_ context.Context, jti, userID string) (refreshConsume, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return refreshUnknown, m.fail
	}
	if owner, ok := m.live[jti]; ok {
		if owner != userID {
			return refreshMismatch, nil
		}
		delete(m.live, jti)
		m.used[jti] = userID
		return refreshLive, nil
	}
	if owner, ok := m.used[jti]; ok {
		if owner != userID {
			return refreshMismatch, nil
		}
		return refreshReused, nil
	}
	return refreshUnknown, nil
}

func (m *memRefreshTracker) markUsed(_ context.Context, jti, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return m.fail
	}
	m.used[jti] = userID
	return nil
}

// trackedAuthService wires an AuthService with an in-memory user row, tracker
// and version-bump seam (no DB/Redis).
func trackedAuthService(user *domain.User) (*AuthService, *memRefreshTracker) {
	s := testAuthService()
	tr := newMemRefreshTracker()
	s.refreshTracker = tr
	s.getUser = func(_ context.Context, id string) (*domain.User, error) {
		if id != user.ID {
			return nil, domain.ErrNotFound
		}
		return user, nil
	}
	s.bumpVersion = func(_ context.Context, id string) error {
		if id != user.ID {
			return domain.ErrNotFound
		}
		user.TokenVersion++
		return nil
	}
	return s, tr
}

func TestRefreshRotatesSingleUse(t *testing.T) {
	ctx := context.Background()
	user := &domain.User{ID: "user-1", Role: domain.RoleUser, IsActive: true}
	s, _ := trackedAuthService(user)
	first, err := s.Issue(ctx, "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	second, err := s.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("first rotation: %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Fatalf("rotation returned the same refresh token")
	}
	// The rotated token chains forward while no reuse is observed.
	third, err := s.Refresh(ctx, second.RefreshToken)
	if err != nil {
		t.Fatalf("rotated refresh token rejected: %v", err)
	}
	if _, err := s.Refresh(ctx, first.RefreshToken); err == nil {
		t.Fatalf("consumed refresh token accepted a second time")
	}
	// The reuse probe above revoked the family, so the chain is dead too.
	if _, err := s.Refresh(ctx, third.RefreshToken); err == nil {
		t.Fatalf("chain survived reuse revocation")
	}
}

func TestRefreshReuseRevokesWholeFamily(t *testing.T) {
	ctx := context.Background()
	user := &domain.User{ID: "user-1", Role: domain.RoleUser, IsActive: true}
	s, _ := trackedAuthService(user)
	first, err := s.Issue(ctx, "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	second, err := s.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("first rotation: %v", err)
	}
	// Attacker replays the consumed token: rejected AND the whole family dies.
	if _, err := s.Refresh(ctx, first.RefreshToken); err == nil {
		t.Fatalf("reused refresh token accepted")
	}
	if user.TokenVersion != 1 {
		t.Fatalf("reuse did not bump token version (got %d)", user.TokenVersion)
	}
	if _, err := s.ValidateAccess(ctx, second.AccessToken); err == nil {
		t.Fatalf("rotated access token still valid after reuse revocation")
	}
	if _, err := s.Refresh(ctx, second.RefreshToken); err == nil {
		t.Fatalf("rotated refresh token still valid after reuse revocation")
	}
	// A fresh login (current version) recovers the account.
	fresh, err := s.issue(ctx, "user-1", domain.RoleUser, user.TokenVersion)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := s.ValidateAccess(ctx, fresh.AccessToken); err != nil {
		t.Fatalf("fresh version token rejected: %v", err)
	}
}

func TestRefreshGraceAllowsUntrackedTokenOnce(t *testing.T) {
	ctx := context.Background()
	// Issued while tracking is disabled: simulates a pre-deploy token that was
	// never recorded.
	untracked := testAuthService()
	legacy, err := untracked.Issue(ctx, "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	user := &domain.User{ID: "user-1", Role: domain.RoleUser, IsActive: true}
	s, _ := trackedAuthService(user)
	if _, err := s.Refresh(ctx, legacy.RefreshToken); err != nil {
		t.Fatalf("untracked token not graced: %v", err)
	}
	if user.TokenVersion != 0 {
		t.Fatalf("grace rotation bumped version (got %d)", user.TokenVersion)
	}
	// Second presentation is now a tracked reuse: rejected, family revoked.
	if _, err := s.Refresh(ctx, legacy.RefreshToken); err == nil {
		t.Fatalf("graced token accepted a second time")
	}
	if user.TokenVersion != 1 {
		t.Fatalf("second use did not revoke (version %d)", user.TokenVersion)
	}
}

func TestRefreshMismatchRejectsWithoutRevoke(t *testing.T) {
	ctx := context.Background()
	user := &domain.User{ID: "user-1", Role: domain.RoleUser, IsActive: true}
	s, tr := trackedAuthService(user)
	// A live jti recorded for another user (store tampering): the presented
	// token carries a valid signature for user-1 but the store disagrees.
	jti := uuid.NewString()
	if err := tr.record(ctx, jti, "user-9"); err != nil {
		t.Fatalf("record: %v", err)
	}
	now := time.Now()
	claims := &Claims{
		UserID: "user-1", Role: domain.RoleUser, TokenType: TokenRefresh, TokenVersion: 0,
		RegisteredClaims: jwt.RegisteredClaims{
			ID: jti, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-refresh-secret-0123456789"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := s.Refresh(ctx, forged); err == nil {
		t.Fatalf("mismatched refresh token accepted")
	}
	if user.TokenVersion != 0 {
		t.Fatalf("mismatch revoked the family (version %d)", user.TokenVersion)
	}
}

func TestRefreshTrackerErrorFailsClosed(t *testing.T) {
	ctx := context.Background()
	user := &domain.User{ID: "user-1", Role: domain.RoleUser, IsActive: true}
	s, tr := trackedAuthService(user)
	first, err := s.Issue(ctx, "user-1", domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	tr.fail = errors.New("redis down")
	if _, err := s.Refresh(ctx, first.RefreshToken); err == nil {
		t.Fatalf("refresh succeeded with tracker error")
	}
	if _, err := s.Issue(ctx, "user-1", domain.RoleUser); err == nil {
		t.Fatalf("issue succeeded with tracker error")
	}
}
