package service

import (
	"context"
	"strings"
	"testing"

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
