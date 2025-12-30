package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
	OpenRouterReasoning       bool
	OpenRouterReasoningEffort string

	SearxNGBaseURL string
	SearchProvider string

	PipelineTimeout     time.Duration
	SearchTimeout       time.Duration
	FetchTimeout        time.Duration
	OpenRouterTimeout   time.Duration
	SearchMaxQueries    int
	SearchMaxSources    int
	SnippetMaxPerSource int
	PageCacheTTL        time.Duration
	ChatHistoryLimit    int

	SerperAPIKey  string
	SerperBaseURL string
	SerperNum     int
	SerperHL      string
	SerperGL      string
}

type fileConfig struct {
	OpenRouter struct {
		Models []string `yaml:"models"`
	} `yaml:"openrouter"`
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
	models, err := loadModelsFromFile()
	if err != nil {
		return Config{}, err
	}
	c.OpenRouterModels = models
	c.OpenRouterReasoning = strings.EqualFold(getenv("OPENROUTER_REASONING", "false"), "true")
	c.OpenRouterReasoningEffort = strings.TrimSpace(getenv("OPENROUTER_REASONING_EFFORT", "medium"))

	c.SearxNGBaseURL = getenv("SEARXNG_BASE_URL", "http://searxng:8080")
	c.SearchProvider = strings.ToLower(getenv("SEARCH_PROVIDER", "searxng"))

	if c.PipelineTimeout, err = parseDurationEnv("PIPELINE_TIMEOUT", "120s"); err != nil {
		return Config{}, err
	}
	if c.SearchTimeout, err = parseDurationEnv("SEARCH_TIMEOUT", "20s"); err != nil {
		return Config{}, err
	}
	if c.FetchTimeout, err = parseDurationEnv("FETCH_TIMEOUT", "20s"); err != nil {
		return Config{}, err
	}
	if c.OpenRouterTimeout, err = parseDurationEnv("OPENROUTER_TIMEOUT", "60s"); err != nil {
		return Config{}, err
	}
	if c.PageCacheTTL, err = parseDurationEnv("PAGE_CACHE_TTL", "24h"); err != nil {
		return Config{}, err
	}

	if c.SearchMaxQueries, err = parseIntEnv("SEARCH_MAX_QUERIES", 3); err != nil {
		return Config{}, err
	}
	if c.SearchMaxSources, err = parseIntEnv("SEARCH_MAX_SOURCES", 5); err != nil {
		return Config{}, err
	}
	if c.SnippetMaxPerSource, err = parseIntEnv("SNIPPET_MAX_PER_SOURCE", 3); err != nil {
		return Config{}, err
	}
	if c.ChatHistoryLimit, err = parseIntEnv("CHAT_HISTORY_LIMIT", 12); err != nil {
		return Config{}, err
	}

	c.SerperAPIKey = strings.TrimSpace(os.Getenv("SERPER_API_KEY"))
	c.SerperBaseURL = getenv("SERPER_BASE_URL", "https://google.serper.dev")
	if c.SerperNum, err = parseIntEnv("SERPER_NUM", 10); err != nil {
		return Config{}, err
	}
	c.SerperHL = getenv("SERPER_HL", "en")
	c.SerperGL = getenv("SERPER_GL", "us")

	return c, nil
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func loadModelsFromFile() ([]string, error) {
	explicit := strings.TrimSpace(os.Getenv("APP_CONFIG_PATH"))
	if explicit != "" {
		return readModels(explicit)
	}

	candidates := []string{"config.yaml", "../config.yaml"}
	for _, path := range candidates {
		models, err := readModels(path)
		if err == nil {
			return models, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("config.yaml not found (set APP_CONFIG_PATH)")
}

func readModels(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fc fileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	models := make([]string, 0, len(fc.OpenRouter.Models))
	for _, model := range fc.OpenRouter.Models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		models = append(models, model)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("no openrouter.models in %s", path)
	}
	return models, nil
}

func parseDurationEnv(key, def string) (time.Duration, error) {
	raw := getenv(key, def)
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}

func parseIntEnv(key string, def int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def, nil
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return val, nil
}
