package config

import (
	"strings"
	"testing"
)

func prodConfig() *Config {
	return &Config{
		Env:  "prod",
		DB:   DBConfig{DSN: "postgres://avibe:secret@postgres:5432/avibe?sslmode=require"},
		JWT:  JWTConfig{AccessSecret: strings.Repeat("a", 32), RefreshSecret: strings.Repeat("r", 32)},
		HLS:  HLSConfig{SigningKey: "hls-key", URLTTLSecs: 300},
		Mail: MailConfig{SMTPHost: "smtp.example.com"},
	}
}

func TestValidateProdHappy(t *testing.T) {
	if err := prodConfig().Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateProdFailsClosed(t *testing.T) {
	base := prodConfig()
	base.Mail.SMTPHost = ""
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "SMTP_HOST") {
		t.Fatalf("missing SMTP accepted: %v", err)
	}
	short := prodConfig()
	short.JWT.AccessSecret = "short"
	if err := short.Validate(); err == nil || !strings.Contains(err.Error(), "JWT") {
		t.Fatalf("short JWT accepted: %v", err)
	}
	noHLS := prodConfig()
	noHLS.HLS.SigningKey = ""
	if err := noHLS.Validate(); err == nil || !strings.Contains(err.Error(), "HLS_SIGNING_KEY") {
		t.Fatalf("missing HLS key accepted: %v", err)
	}
	noTTL := prodConfig()
	noTTL.HLS.URLTTLSecs = 0
	if err := noTTL.Validate(); err == nil || !strings.Contains(err.Error(), "HLS_URL_TTL_SECONDS") {
		t.Fatalf("non-positive HLS TTL accepted: %v", err)
	}
}

func TestValidateProdRequiresDBTLS(t *testing.T) {
	disable := prodConfig()
	disable.DB.DSN = "postgres://avibe:pw@postgres:5432/avibe?sslmode=disable"
	if err := disable.Validate(); err == nil || !strings.Contains(err.Error(), "sslmode") {
		t.Fatalf("disable DSN accepted: %v", err)
	}
	empty := prodConfig()
	empty.DB.DSN = ""
	if err := empty.Validate(); err == nil || !strings.Contains(err.Error(), "DB_DSN") {
		t.Fatalf("empty DSN accepted: %v", err)
	}
	implied := prodConfig()
	implied.DB.DSN = "postgres://avibe:pw@postgres:5432/avibe"
	if err := implied.Validate(); err == nil || !strings.Contains(err.Error(), "sslmode") {
		t.Fatalf("default (prefer) DSN accepted: %v", err)
	}
}

func TestLoadBuildsSecureProdDSN(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("DB_DSN", "")
	t.Setenv("DB_HOST", "postgres")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cfg.DB.DSN, "sslmode=require") {
		t.Fatalf("prod built DSN = %q, want sslmode=require", cfg.DB.DSN)
	}
	t.Setenv("APP_ENV", "local")
	if cfg, _ = Load(); !strings.Contains(cfg.DB.DSN, "sslmode=disable") {
		t.Fatalf("local built DSN = %q, want sslmode=disable default", cfg.DB.DSN)
	}
}

func TestValidateNonProdSkips(t *testing.T) {
	for _, env := range []string{"", "local", "dev", "test", "LOCAL"} {
		c := &Config{Env: env}
		if err := c.Validate(); err != nil {
			t.Fatalf("env %q: %v", env, err)
		}
	}
}

func TestOAuthCredentialsFromEnv(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	t.Setenv("KAKAO_CLIENT_ID", "")
	t.Setenv("KAKAO_CLIENT_SECRET", "")
	t.Setenv("APPLE_SERVICES_ID", "")
	t.Setenv("APPLE_TEAM_ID", "")
	t.Setenv("APPLE_KEY_ID", "")
	t.Setenv("APPLE_PRIVATE_KEY", "")
	t.Setenv("NAVER_CLIENT_ID", "")
	t.Setenv("NAVER_CLIENT_SECRET", "")
	t.Setenv("LINE_CHANNEL_ID", "")
	t.Setenv("LINE_CHANNEL_SECRET", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OAuth.GoogleClientID != "" || cfg.OAuth.AppleTeamID != "" || cfg.OAuth.LineChannelSecret != "" {
		t.Fatalf("oauth creds leak across env: %+v", cfg.OAuth)
	}
	if cfg.OAuth.SecureCookies {
		t.Fatalf("test env must not set Secure cookies")
	}
	t.Setenv("APPLE_SERVICES_ID", "com.example.signin")
	t.Setenv("APPLE_TEAM_ID", "TEAM1")
	t.Setenv("APPLE_KEY_ID", "KEY1")
	t.Setenv("APPLE_PRIVATE_KEY", "p8-pem")
	t.Setenv("LINE_CHANNEL_ID", "lc")
	t.Setenv("LINE_CHANNEL_SECRET", "ls")
	if cfg, _ = Load(); cfg.OAuth.AppleServicesID != "com.example.signin" || cfg.OAuth.LineChannelSecret != "ls" {
		t.Fatalf("oauth creds not loaded: %+v", cfg.OAuth)
	}
	if cfg.OAuth.GoogleTokenURL != "https://oauth2.googleapis.com/token" || cfg.OAuth.AppleJWKSURL != "https://appleid.apple.com/auth/keys" || cfg.OAuth.LineJWKSURL != "https://api.line.me/oauth2/v2.1/certs" {
		t.Fatalf("oauth endpoint defaults: %+v", cfg.OAuth)
	}
	t.Setenv("APP_ENV", "prod")
	if cfg, _ = Load(); !cfg.OAuth.SecureCookies {
		t.Fatalf("prod must set Secure cookies")
	}
}

func TestEmailVerifyTTLFromEnv(t *testing.T) {
	t.Setenv("EMAIL_VERIFY_TTL_HOURS", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Verify.TokenTTLHours != 24 {
		t.Fatalf("default ttl = %d, want 24", cfg.Verify.TokenTTLHours)
	}
	t.Setenv("EMAIL_VERIFY_TTL_HOURS", "48")
	if cfg, _ = Load(); cfg.Verify.TokenTTLHours != 48 {
		t.Fatalf("ttl = %d, want 48", cfg.Verify.TokenTTLHours)
	}
}

func TestAdminTOTPRequiredDefaults(t *testing.T) {
	t.Setenv("ADMIN_TOTP_REQUIRED", "")
	t.Setenv("APP_ENV", "local")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.AdminTOTPRequired {
		t.Fatal("local must not require admin TOTP by default")
	}
	t.Setenv("APP_ENV", "prod")
	if cfg, _ = Load(); !cfg.Security.AdminTOTPRequired {
		t.Fatal("prod must require admin TOTP by default")
	}
	t.Setenv("ADMIN_TOTP_REQUIRED", "false")
	if cfg, _ = Load(); cfg.Security.AdminTOTPRequired {
		t.Fatal("explicit false must win")
	}
}
