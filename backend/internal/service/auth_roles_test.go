package service

import (
	"context"
	"errors"
	"testing"

	"avibe-platform/backend/internal/domain"
)

func TestClaimsCarryRoles(t *testing.T) {
	s := testAuthService()
	s.listRoles = func(_ context.Context, _ string) ([]string, error) {
		return []string{domain.RolePartner, domain.RoleBusiness}, nil
	}
	tok, err := s.Issue(context.Background(), "user-1", domain.RolePartner)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	claims, err := s.ParseAccess(tok.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != domain.RolePartner || claims.Roles[1] != domain.RoleBusiness {
		t.Fatalf("roles = %v, want [PARTNER BUSINESS]", claims.Roles)
	}
}

func TestEffectiveRolesFallback(t *testing.T) {
	s := testAuthService() // nil repos: listRoles unset
	if got := s.EffectiveRoles(context.Background(), "u", domain.RoleUser); len(got) != 1 || got[0] != domain.RoleUser {
		t.Fatalf("fallback = %v, want [USER]", got)
	}
	s.listRoles = func(_ context.Context, _ string) ([]string, error) { return nil, errors.New("db down") }
	if got := s.EffectiveRoles(context.Background(), "u", domain.RoleUser); len(got) != 1 || got[0] != domain.RoleUser {
		t.Fatalf("error fallback = %v, want [USER]", got)
	}
}

func TestValidateAccessRefreshesRoles(t *testing.T) {
	s := testAuthService()
	user := &domain.User{ID: "user-1", Role: domain.RolePartner, IsActive: true}
	s.getUser = func(_ context.Context, id string) (*domain.User, error) { return user, nil }
	s.listRoles = func(_ context.Context, _ string) ([]string, error) {
		return []string{domain.RolePartner, domain.RoleBusiness}, nil
	}
	tok, err := s.Issue(context.Background(), "user-1", domain.RolePartner)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	claims, err := s.ValidateAccess(context.Background(), tok.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccess: %v", err)
	}
	if len(claims.Roles) != 2 || claims.Roles[1] != domain.RoleBusiness {
		t.Fatalf("roles = %v, want [PARTNER BUSINESS]", claims.Roles)
	}
}

func TestRegisterBusinessValidation(t *testing.T) {
	s := testAuthService() // nil repos: only pre-DB validation paths run
	ctx := context.Background()
	cases := []struct {
		name    string
		email   string
		pw      string
		company string
		bizNo   string
		country string
	}{
		{"bad email", "not-an-email", "Password1!", "Co", "1234567891", "KR"},
		{"short password", "b@example.com", "short", "Co", "1234567891", "KR"},
		{"empty company", "b@example.com", "Password1!", "", "1234567891", "KR"},
		{"KR bad checksum", "b@example.com", "Password1!", "Co", "1234567890", "KR"},
		{"KR missing bizno", "b@example.com", "Password1!", "Co", "", "KR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := s.RegisterBusiness(ctx, tc.email, tc.pw, "", tc.company, tc.bizNo, tc.country, "", nil, true, "")
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestValidRole(t *testing.T) {
	for _, r := range []string{domain.RoleAdmin, domain.RolePartner, domain.RoleUser, domain.RoleBusiness} {
		if !domain.ValidRole(r) {
			t.Fatalf("ValidRole(%s) = false", r)
		}
	}
	if domain.ValidRole("SUPERUSER") {
		t.Fatal("ValidRole(SUPERUSER) = true")
	}
}
