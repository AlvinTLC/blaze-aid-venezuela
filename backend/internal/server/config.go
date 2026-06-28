package server

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const defaultJWTSecret = "dev-insecure-secret-change-me"

// Config holds runtime configuration sourced from the environment.
type Config struct {
	Env         string
	Addr        string
	DatabaseURL string
	RedisURL    string
	JWTSecret    string
	CORSOrigins  []string
	RateLimitRPM int
	MaxBodyBytes int64
	AppBaseURL   string
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPass     string
	SMTPFrom     string
	SMTPTLS      bool
}

// LoadConfig reads configuration from the environment with dev-friendly defaults.
func LoadConfig() Config {
	return Config{
		Env:         env("ENV", "development"),
		Addr:        env("ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://blazeaid:blazeaid@localhost:5432/blazeaid?sslmode=disable"),
		RedisURL:    env("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:    env("JWT_SECRET", defaultJWTSecret),
		CORSOrigins:  splitCSV(env("CORS_ORIGINS", "*")),
		RateLimitRPM: envInt("RATE_LIMIT_RPM", 100),
		MaxBodyBytes: int64(envInt("MAX_BODY_BYTES", 1<<20)), // 1 MiB
		AppBaseURL:   env("APP_BASE_URL", ""),
		SMTPHost:     env("SMTP_HOST", ""),
		SMTPPort:     env("SMTP_PORT", ""),
		SMTPUser:     env("SMTP_USER", ""),
		SMTPPass:     env("SMTP_PASS", ""),
		SMTPFrom:     env("SMTP_FROM", ""),
		SMTPTLS:      envBool("SMTP_TLS", false),
	}
}

// envBool reads a boolean env var ("true"/"1"); falls back to def.
func envBool(key string, def bool) bool {
	switch strings.ToLower(os.Getenv(key)) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return def
	}
}

// envInt reads an integer env var, falling back to def on missing/invalid.
func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// splitCSV parses a comma-separated env value into a trimmed, non-empty slice.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

// IsProduction reports whether the service runs in a production-like environment.
func (c Config) IsProduction() bool {
	switch strings.ToLower(c.Env) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

// Validate fails fast on insecure configuration in production, so we never ship
// the development defaults (e.g. the well-known JWT secret) to a live deploy.
func (c Config) Validate() error {
	if c.IsProduction() && c.JWTSecret == defaultJWTSecret {
		return errors.New("JWT_SECRET is the insecure default; set a strong secret in production")
	}
	return nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
