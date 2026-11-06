// Package config loads process configuration from environment variables.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is the backend process configuration.
type Config struct {
	Env     string
	AppAddr string
	DB      DBConfig
	Redis   RedisConfig
	Storage StorageConfig
	JWT     JWTConfig
	// AESKey is the decoded 32-byte key from AES256_KEY_BASE64 (nil when unset).
	AESKey []byte
	HLS    HLSConfig
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

// StorageConfig configures S3-compatible object storage (MinIO or R2).
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
		Name:     getenv("DB_NAME", "ai_music"),
		User:     getenv("DB_USER", "ai_music"),
		Password: os.Getenv("DB_PASSWORD"),
	}
	if cfg.DB.DSN == "" {
		cfg.DB.DSN = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)
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
	return cfg, nil
}

func loadStorage() StorageConfig {
	driver := getenv("STORAGE_DRIVER", "minio")
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
		Driver:          "minio",
		Endpoint:        getenv("S3_ENDPOINT", "http://localhost:9000"),
		Region:          getenv("S3_REGION", "us-east-1"),
		AccessKey:       os.Getenv("S3_ACCESS_KEY"),
		SecretKey:       os.Getenv("S3_SECRET_KEY"),
		UseSSL:          getenvBool("S3_USE_SSL", false),
		StagingBucket:   getenv("MINIO_STAGING_BUCKET", "minio-staging"),
		PublicHLSBucket: getenv("MINIO_PUBLIC_HLS_BUCKET", "minio-public-hls"),
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
