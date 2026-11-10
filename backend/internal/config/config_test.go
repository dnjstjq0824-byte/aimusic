package config

import (
	"strings"
	"testing"
)

func prodConfig() *Config {
	return &Config{
		Env:  "prod",
		JWT:  JWTConfig{AccessSecret: strings.Repeat("a", 32), RefreshSecret: strings.Repeat("r", 32)},
		HLS:  HLSConfig{SigningKey: "hls-key"},
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
}

func TestValidateNonProdSkips(t *testing.T) {
	for _, env := range []string{"", "local", "dev", "test", "LOCAL"} {
		c := &Config{Env: env}
		if err := c.Validate(); err != nil {
			t.Fatalf("env %q: %v", env, err)
		}
	}
}
