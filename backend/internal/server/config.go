package server

import (
	"errors"
	"os"
	"strings"
)

const defaultJWTSecret = "dev-insecure-secret-change-me"

// Config holds runtime configuration sourced from the environment.
type Config struct {
	Env         string
	Addr        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

// LoadConfig reads configuration from the environment with dev-friendly defaults.
func LoadConfig() Config {
	return Config{
		Env:         env("ENV", "development"),
		Addr:        env("ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://blazeaid:blazeaid@localhost:5432/blazeaid?sslmode=disable"),
		RedisURL:    env("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:   env("JWT_SECRET", defaultJWTSecret),
	}
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
