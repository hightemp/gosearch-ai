package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Env         string
	HTTPAddr    string
	BaseURL     string
	DatabaseURL string

	JWTIssuer     string
	JWTSigningKey string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration

	OpenRouterAPIKey  string
	OpenRouterBaseURL string
	OpenRouterModels  []string

	SearxNGBaseURL string
}

func LoadFromEnv() (Config, error) {
	c := Config{}

	c.Env = getenv("APP_ENV", "dev")
	c.HTTPAddr = getenv("APP_HTTP_ADDR", ":8081")
	c.BaseURL = getenv("APP_BASE_URL", "http://localhost:8081")
	c.DatabaseURL = getenv("DATABASE_URL", "")
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	c.JWTIssuer = getenv("JWT_ISSUER", "gosearch-ai")
	c.JWTSigningKey = getenv("JWT_SIGNING_KEY", "")
	if c.JWTSigningKey == "" {
		return Config{}, fmt.Errorf("JWT_SIGNING_KEY is required")
	}

	accessTTL, err := time.ParseDuration(getenv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("JWT_ACCESS_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(getenv("JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return Config{}, fmt.Errorf("JWT_REFRESH_TTL: %w", err)
	}
	c.AccessTTL = accessTTL
	c.RefreshTTL = refreshTTL

	c.OpenRouterAPIKey = getenv("OPENROUTER_API_KEY", "")
	c.OpenRouterBaseURL = getenv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1")
	c.OpenRouterModels = splitComma(getenv("OPENROUTER_MODELS", "openai/gpt-4.1-mini"))

	c.SearxNGBaseURL = getenv("SEARXNG_BASE_URL", "http://searxng:8080")

	return c, nil
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
