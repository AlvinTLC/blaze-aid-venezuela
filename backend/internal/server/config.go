package server

import "os"

// Config holds runtime configuration sourced from the environment.
type Config struct {
	Addr        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

// LoadConfig reads configuration from the environment with dev-friendly defaults.
func LoadConfig() Config {
	return Config{
		Addr:        env("ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://blazeaid:blazeaid@localhost:5432/blazeaid?sslmode=disable"),
		RedisURL:    env("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:   env("JWT_SECRET", "dev-insecure-secret-change-me"),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
