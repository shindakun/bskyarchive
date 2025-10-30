package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Archive   ArchiveConfig   `yaml:"archive"`
	OAuth     OAuthConfig     `yaml:"oauth"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Port            int           `yaml:"port"`
	Host            string        `yaml:"host"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// ArchiveConfig contains archive-specific settings
type ArchiveConfig struct {
	DBPath           string `yaml:"db_path"`
	MediaPath        string `yaml:"media_path"`
	MaxArchiveSizeGB int    `yaml:"max_archive_size_gb"`
	WorkerCount      int    `yaml:"worker_count"`
	BatchSize        int    `yaml:"batch_size"`
}

// OAuthConfig contains Bluesky OAuth settings
type OAuthConfig struct {
	ClientID      string   `yaml:"client_id"`
	ClientSecret  string   `yaml:"client_secret"`
	RedirectURL   string   `yaml:"redirect_url"`
	Scopes        []string `yaml:"scopes"`
	SessionSecret string   `yaml:"session_secret"`
	SessionMaxAge int      `yaml:"session_max_age"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	RequestsPerWindow int           `yaml:"requests_per_window"`
	WindowDuration    time.Duration `yaml:"window_duration"`
	Burst             int           `yaml:"burst"`
}

// Load reads configuration from the specified file path
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the config
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate checks that all required configuration fields are set
func (c *Config) Validate() error {
	// OAuth validation
	if c.OAuth.ClientID == "" || strings.Contains(c.OAuth.ClientID, "${") {
		return fmt.Errorf("oauth.client_id is required (set OAUTH_CLIENT_ID environment variable)")
	}
	if c.OAuth.ClientSecret == "" || strings.Contains(c.OAuth.ClientSecret, "${") {
		return fmt.Errorf("oauth.client_secret is required (set OAUTH_CLIENT_SECRET environment variable)")
	}
	if c.OAuth.RedirectURL == "" || strings.Contains(c.OAuth.RedirectURL, "${") {
		return fmt.Errorf("oauth.redirect_url is required (set OAUTH_REDIRECT_URL environment variable)")
	}
	if c.OAuth.SessionSecret == "" || strings.Contains(c.OAuth.SessionSecret, "${") {
		return fmt.Errorf("oauth.session_secret is required (set SESSION_SECRET environment variable)")
	}
	if len(c.OAuth.SessionSecret) < 32 {
		return fmt.Errorf("oauth.session_secret must be at least 32 characters")
	}

	// Server validation
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	// Archive validation
	if c.Archive.DBPath == "" {
		return fmt.Errorf("archive.db_path is required")
	}
	if c.Archive.MediaPath == "" {
		return fmt.Errorf("archive.media_path is required")
	}
	if c.Archive.WorkerCount < 1 {
		return fmt.Errorf("archive.worker_count must be at least 1")
	}
	if c.Archive.BatchSize < 1 {
		return fmt.Errorf("archive.batch_size must be at least 1")
	}

	// Rate limit validation
	if c.RateLimit.RequestsPerWindow < 1 {
		return fmt.Errorf("rate_limit.requests_per_window must be at least 1")
	}

	return nil
}

// GetAddr returns the full server address (host:port)
func (c *Config) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
