// Package config loads process configuration from environment variables.
package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config is the backend process configuration.
type Config struct {
	Env      string
	AppAddr  string
	DB       DBConfig
	Redis    RedisConfig
	Storage  StorageConfig
	JWT      JWTConfig
	Security SecurityConfig
	// AESKey is the decoded 32-byte key from AES256_KEY_BASE64 (nil when unset).
	AESKey []byte
	HLS    HLSConfig
	Reset  PasswordResetConfig
	Verify EmailVerifyConfig
	OAuth  OAuthConfig
	Toss   TossConfig
	Mail   MailConfig
	// Generation is the external music-generation backend (WI-03 setup-only).
	// Empty BackendURL means no GPU server is connected; generation jobs then
	// fail with a clear message instead of fake audio.
	Generation GenerationConfig
	// SEO covers sitemap/IndexNow/cache-purge/search-console knobs (WI-S*).
	SEO SEOConfig
}

// SEOConfig configures search exposure (WI-S1~S5). Keys stay empty until the
// PO provisions them; every consumer degrades gracefully (log + skip) while
// unset, and StartupWarnings names only the missing variable, never values.
type SEOConfig struct {
	// SiteBaseURL is the public web origin (canonical host, no trailing slash).
	SiteBaseURL string
	// IndexNowKey enables IndexNow submission; the same value is served at
	// /{key}.txt by the web tier for ownership proof.
	IndexNowKey string
	// IndexNowEndpoint defaults to https://api.indexnow.org/indexnow (tests
	// override it with a stub server).
	IndexNowEndpoint string
	// IndexNowDailyCap bounds submissions per rolling 24h (default 10000,
	// the IndexNow quota).
	IndexNowDailyCap int
	// CachePurgeToken + ZoneID enable Cloudflare cache purge on content
	// change; empty token means rely on SWR expiry.
	CachePurgeToken string
	CachePurgeZone  string
	// Console verification tokens (meta/file placeholders; PO-provided).
	GoogleVerify string
	BingVerify   string
	NaverVerify  string
	DaumVerify   string
}

// SecurityConfig configures HTTP hardening (CORS allowlist, trusted proxies).
type SecurityConfig struct {
	// CORSAllowedOrigins is the exact-match allowlist for the Origin header.
	// Empty means same-origin only (no CORS headers emitted).
	CORSAllowedOrigins []string
	// TrustedProxies is passed to gin.Engine.SetTrustedProxies so ClientIP
	// only trusts X-Forwarded-For from these CIDRs/IPs. Empty trusts none.
	TrustedProxies []string
	// RegistrationInviteCode gates sign-up while set (C8): register calls
	// must present this code. Empty means open registration.
	RegistrationInviteCode string
	// AdminTOTPRequired blocks /admin/* for admins without an enrolled
	// second factor (S-03). Default: on outside local/dev/test.
	AdminTOTPRequired bool
}

// DBConfig configures PostgreSQL access.
type DBConfig struct {
	DSN      string
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

// RedisConfig configures Redis access.
type RedisConfig struct {
	Addr string
	Host string
	Port int
}

// StorageConfig configures S3-compatible object storage (LocalStack local, R2 in staging/prod).
type StorageConfig struct {
	Driver          string
	Endpoint        string
	Region          string
	AccessKey       string
	SecretKey       string
	UseSSL          bool
	StagingBucket   string
	PublicHLSBucket string
	CDNBaseURL      string
}

// JWTConfig configures token issuance.
type JWTConfig struct {
	AccessSecret     string
	RefreshSecret    string
	AccessTTLMinutes int
	RefreshTTLDays   int
}

// HLSConfig configures signed playback URLs.
type HLSConfig struct {
	SigningKey string
	URLTTLSecs int
}

// PasswordResetConfig configures the reset-link flow.
type PasswordResetConfig struct {
	// TokenTTLMinutes bounds single-use reset links (default 30).
	TokenTTLMinutes int
	// FrontendBaseURL is the web origin links point at (no trailing slash).
	FrontendBaseURL string
}

// EmailVerifyConfig configures the email-verification flow (WI-A2).
type EmailVerifyConfig struct {
	// TokenTTLHours bounds single-use verification links (default 24).
	TokenTTLHours int
}

// OAuthConfig configures Google/Kakao/Apple/Naver/LINE social login (WI-A1).
// A provider enables when every required credential is set (Kakao secret is
// optional); endpoint URLs default to the real provider hosts and exist as
// env knobs for hermetic tests.
type OAuthConfig struct {
	FrontendBaseURL string
	RedirectBaseURL string
	// SecureCookies marks OAuth state cookies Secure (off on local/dev/test
	// so plain-http flows keep working).
	SecureCookies      bool
	GoogleClientID     string
	GoogleClientSecret string
	GoogleTokenURL     string
	GoogleUserinfoURL  string
	KakaoClientID      string
	KakaoClientSecret  string
	KakaoTokenURL      string
	KakaoUserinfoURL   string
	AppleServicesID    string
	AppleTeamID        string
	AppleKeyID         string
	ApplePrivateKey    string
	AppleTokenURL      string
	AppleJWKSURL       string
	NaverClientID      string
	NaverClientSecret  string
	NaverTokenURL      string
	NaverUserinfoURL   string
	LineChannelID      string
	LineChannelSecret  string
	LineTokenURL       string
	LineJWKSURL        string
	StateSecret        string
}

// TossConfig configures Toss Payments billing. Billing endpoints fail closed
// with 503 while SecretKey is empty.
type TossConfig struct {
	SecretKey  string
	APIBaseURL string
}

// MailConfig configures outbound mail. When SMTPHost is empty the API logs
// mail instead of sending (local/dev behavior).
type MailConfig struct {
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	From     string
	FromName string
	// NotifyEmail receives ops alerts (business inquiries). Empty disables.
	NotifyEmail string
}

// GenerationConfig points the worker at an external GPU generation server
// (reference: generator/server.py). Timeouts in seconds.
type GenerationConfig struct {
	BackendURL string
	// BackendToken is sent as `Authorization: Bearer` to the generation
	// server (WI-N12). Empty = no header (local servers without auth).
	BackendToken   string
	TimeoutSeconds int
	// DailyCapPerPartner is the server-enforced submit quota (default 5).
	DailyCapPerPartner int
	// BannedKeywordsExtra appends operator block phrases (comma-separated).
	BannedKeywordsExtra []string
}

// Load reads configuration from the environment.
func Load() (*Config, error) {
	cfg := &Config{
		Env:     getenv("APP_ENV", "local"),
		AppAddr: getenv("APP_ADDR", ":8080"),
	}
	cfg.DB = DBConfig{
		DSN:      os.Getenv("DB_DSN"),
		Host:     getenv("DB_HOST", "localhost"),
		Port:     getenvInt("DB_PORT", 5432),
		Name:     getenv("DB_NAME", "avibe"),
		User:     getenv("DB_USER", "avibe"),
		Password: os.Getenv("DB_PASSWORD"),
	}
	if cfg.DB.DSN == "" {
		sslMode := "disable"
		if !nonProdEnv(cfg.Env) {
			// Never assemble a plaintext default for prod: without a
			// TLS-capable server this fails closed at connect time.
			sslMode = "require"
		}
		cfg.DB.DSN = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, sslMode)
	}
	redisHost := getenv("REDIS_HOST", "localhost")
	redisPort := getenvInt("REDIS_PORT", 6379)
	cfg.Redis = RedisConfig{
		Addr: getenv("REDIS_ADDR", fmt.Sprintf("%s:%d", redisHost, redisPort)),
		Host: redisHost,
		Port: redisPort,
	}
	cfg.Storage = loadStorage()
	cfg.JWT = JWTConfig{
		AccessSecret:     os.Getenv("JWT_ACCESS_SECRET"),
		RefreshSecret:    os.Getenv("JWT_REFRESH_SECRET"),
		AccessTTLMinutes: getenvInt("JWT_ACCESS_TTL_MINUTES", 15),
		RefreshTTLDays:   getenvInt("JWT_REFRESH_TTL_DAYS", 14),
	}
	if raw := strings.TrimSpace(os.Getenv("AES256_KEY_BASE64")); raw != "" {
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("decode AES256_KEY_BASE64: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("AES256_KEY_BASE64 must decode to 32 bytes, got %d", len(key))
		}
		cfg.AESKey = key
	}
	cfg.HLS = HLSConfig{
		SigningKey: os.Getenv("HLS_SIGNING_KEY"),
		URLTTLSecs: getenvInt("HLS_URL_TTL_SECONDS", 300),
	}
	cfg.Security = SecurityConfig{
		CORSAllowedOrigins:     getenvList("CORS_ALLOWED_ORIGINS"),
		TrustedProxies:         getenvList("TRUSTED_PROXIES"),
		RegistrationInviteCode: strings.TrimSpace(os.Getenv("REGISTRATION_INVITE_CODE")),
		AdminTOTPRequired:      getenvBool("ADMIN_TOTP_REQUIRED", !nonProdEnv(cfg.Env)),
	}
	frontendBase := strings.TrimRight(getenv("FRONTEND_BASE_URL", "http://localhost:5173"), "/")
	cfg.Reset = PasswordResetConfig{
		TokenTTLMinutes: getenvInt("PASSWORD_RESET_TTL_MINUTES", 30),
		FrontendBaseURL: frontendBase,
	}
	cfg.Verify = EmailVerifyConfig{
		TokenTTLHours: getenvInt("EMAIL_VERIFY_TTL_HOURS", 24),
	}
	cfg.OAuth = OAuthConfig{
		FrontendBaseURL:    frontendBase,
		RedirectBaseURL:    strings.TrimRight(getenv("OAUTH_REDIRECT_BASE_URL", "http://localhost:8080/api/v1/auth/oauth"), "/"),
		SecureCookies:      !nonProdEnv(cfg.Env),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleTokenURL:     getenv("GOOGLE_TOKEN_URL", "https://oauth2.googleapis.com/token"),
		GoogleUserinfoURL:  getenv("GOOGLE_USERINFO_URL", "https://openidconnect.googleapis.com/v1/userinfo"),
		KakaoClientID:      os.Getenv("KAKAO_CLIENT_ID"),
		KakaoClientSecret:  os.Getenv("KAKAO_CLIENT_SECRET"),
		KakaoTokenURL:      getenv("KAKAO_TOKEN_URL", "https://kauth.kakao.com/oauth/token"),
		KakaoUserinfoURL:   getenv("KAKAO_USERINFO_URL", "https://kapi.kakao.com/v2/user/me"),
		AppleServicesID:    os.Getenv("APPLE_SERVICES_ID"),
		AppleTeamID:        os.Getenv("APPLE_TEAM_ID"),
		AppleKeyID:         os.Getenv("APPLE_KEY_ID"),
		ApplePrivateKey:    os.Getenv("APPLE_PRIVATE_KEY"),
		AppleTokenURL:      getenv("APPLE_TOKEN_URL", "https://appleid.apple.com/auth/token"),
		AppleJWKSURL:       getenv("APPLE_JWKS_URL", "https://appleid.apple.com/auth/keys"),
		NaverClientID:      os.Getenv("NAVER_CLIENT_ID"),
		NaverClientSecret:  os.Getenv("NAVER_CLIENT_SECRET"),
		NaverTokenURL:      getenv("NAVER_TOKEN_URL", "https://nid.naver.com/oauth2.0/token"),
		NaverUserinfoURL:   getenv("NAVER_USERINFO_URL", "https://openapi.naver.com/v1/nid/me"),
		LineChannelID:      os.Getenv("LINE_CHANNEL_ID"),
		LineChannelSecret:  os.Getenv("LINE_CHANNEL_SECRET"),
		LineTokenURL:       getenv("LINE_TOKEN_URL", "https://api.line.me/oauth2/v2.1/token"),
		LineJWKSURL:        getenv("LINE_JWKS_URL", "https://api.line.me/oauth2/v2.1/certs"),
		StateSecret:        os.Getenv("OAUTH_STATE_SECRET"),
	}
	cfg.Toss = TossConfig{
		SecretKey:  os.Getenv("TOSS_SECRET_KEY"),
		APIBaseURL: strings.TrimRight(getenv("TOSS_API_BASE_URL", "https://api.tosspayments.com"), "/"),
	}
	cfg.Mail = MailConfig{
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPPort:    getenvInt("SMTP_PORT", 587),
		SMTPUser:    os.Getenv("SMTP_USER"),
		SMTPPass:    os.Getenv("SMTP_PASS"),
		NotifyEmail: os.Getenv("OPS_NOTIFY_EMAIL"),
		From:        getenv("MAIL_FROM", "no-reply@avibemusic.com"),
		FromName:    getenv("MAIL_FROM_NAME", "AVIBE"),
	}
	cfg.Generation = GenerationConfig{
		BackendURL:          strings.TrimRight(strings.TrimSpace(os.Getenv("GENERATION_BACKEND_URL")), "/"),
		BackendToken:        strings.TrimSpace(os.Getenv("GENERATION_BACKEND_TOKEN")),
		TimeoutSeconds:      getenvInt("GENERATION_TIMEOUT_SECONDS", 600),
		DailyCapPerPartner:  getenvInt("GENERATION_DAILY_CAP", 5),
		BannedKeywordsExtra: getenvList("GENERATION_BANNED_KEYWORDS"),
	}
	cfg.SEO = SEOConfig{
		SiteBaseURL:      strings.TrimRight(getenv("SITE_BASE_URL", "https://avibemusic.com"), "/"),
		IndexNowKey:      strings.TrimSpace(os.Getenv("INDEXNOW_KEY")),
		IndexNowEndpoint: strings.TrimRight(getenv("INDEXNOW_ENDPOINT", "https://api.indexnow.org/indexnow"), "/"),
		IndexNowDailyCap: getenvInt("INDEXNOW_DAILY_CAP", 10000),
		CachePurgeToken:  strings.TrimSpace(os.Getenv("CF_CACHE_PURGE_TOKEN")),
		CachePurgeZone:   strings.TrimSpace(os.Getenv("CF_ZONE_ID")),
		GoogleVerify:     strings.TrimSpace(os.Getenv("GOOGLE_SITE_VERIFICATION")),
		BingVerify:       strings.TrimSpace(os.Getenv("BING_SITE_VERIFICATION")),
		NaverVerify:      strings.TrimSpace(os.Getenv("NAVER_SITE_VERIFICATION")),
		DaumVerify:       strings.TrimSpace(os.Getenv("DAUM_SITE_VERIFICATION")),
	}
	return cfg, nil
}

// nonProdEnvs skips fail-closed guards (log-only mail, test keys allowed).
func nonProdEnv(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "", "local", "dev", "test":
		return true
	default:
		return false
	}
}

// Validate fails closed on missing production secrets. Without this the API
// would boot in prod with log-only mail (password resets silently never sent),
// unsigned-strength JWT keys, or unsigned HLS grants.
func (c *Config) Validate() error {
	if nonProdEnv(c.Env) {
		return nil
	}
	if strings.TrimSpace(c.Mail.SMTPHost) == "" {
		return fmt.Errorf("SMTP_HOST is required in env %q (refusing log-only mail)", c.Env)
	}
	if len(c.JWT.AccessSecret) < 32 || len(c.JWT.RefreshSecret) < 32 {
		return fmt.Errorf("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must be >= 32 chars in env %q", c.Env)
	}
	if strings.TrimSpace(c.HLS.SigningKey) == "" {
		return fmt.Errorf("HLS_SIGNING_KEY is required in env %q", c.Env)
	}
	if c.HLS.URLTTLSecs <= 0 {
		return fmt.Errorf("HLS_URL_TTL_SECONDS must be positive in env %q", c.Env)
	}
	if !prodDBTLSOK(c.DB.DSN) {
		return fmt.Errorf("DB_DSN must use TLS in env %q (sslmode=require, verify-ca, or verify-full)", c.Env)
	}
	return nil
}

// prodDBTLSOK reports whether a database URL carries explicit TLS. The
// implied default (prefer) silently falls back to plaintext against a
// non-TLS server, so prod requires an explicit encrypted mode.
func prodDBTLSOK(dsn string) bool {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return false
	}
	switch u.Query().Get("sslmode") {
	case "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}

func getenvList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadStorage() StorageConfig {
	driver := getenv("STORAGE_DRIVER", "s3local")
	if driver == "r2" {
		return StorageConfig{
			Driver:          "r2",
			Endpoint:        os.Getenv("R2_ENDPOINT"),
			Region:          getenv("S3_REGION", "auto"),
			AccessKey:       os.Getenv("R2_ACCESS_KEY_ID"),
			SecretKey:       os.Getenv("R2_SECRET_ACCESS_KEY"),
			UseSSL:          true,
			StagingBucket:   getenv("R2_STAGING_BUCKET", "staging-private"),
			PublicHLSBucket: getenv("R2_PUBLIC_HLS_BUCKET", "public-hls"),
			CDNBaseURL:      os.Getenv("CDN_BASE_URL"),
		}
	}
	return StorageConfig{
		Driver:          "s3local",
		Endpoint:        getenv("S3_ENDPOINT", "http://localhost:4566"),
		Region:          getenv("S3_REGION", "us-east-1"),
		AccessKey:       os.Getenv("S3_ACCESS_KEY"),
		SecretKey:       os.Getenv("S3_SECRET_KEY"),
		UseSSL:          getenvBool("S3_USE_SSL", false),
		StagingBucket:   getenv("S3_STAGING_BUCKET", "local-staging"),
		PublicHLSBucket: getenv("S3_PUBLIC_HLS_BUCKET", "local-hls"),
		CDNBaseURL:      os.Getenv("CDN_BASE_URL"),
	}
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(key string, def bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
