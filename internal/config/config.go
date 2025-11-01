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
	BaseURL         string        `yaml:"base_url"` // Optional: Override for OAuth (e.g., https://your-domain.com)
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	Security        SecurityConfig `yaml:"security"`
}

// SecurityConfig contains security-related settings
type SecurityConfig struct {
	CSRFEnabled     bool                  `yaml:"csrf_enabled"`
	CSRFFieldName   string                `yaml:"csrf_field_name"`
	MaxRequestBytes int64                 `yaml:"max_request_bytes"`
	Headers         SecurityHeadersConfig `yaml:"headers"`
}

// SecurityHeadersConfig contains HTTP security header settings
type SecurityHeadersConfig struct {
	XFrameOptions           string `yaml:"x_frame_options"`
	XContentTypeOptions     string `yaml:"x_content_type_options"`
	XXSSProtection          string `yaml:"x_xss_protection"`
	ReferrerPolicy          string `yaml:"referrer_policy"`
	ContentSecurityPolicy   string `yaml:"content_security_policy"`
	StrictTransportSecurity string `yaml:"strict_transport_security"`
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
	Scopes         []string `yaml:"scopes"`
	SessionSecret  string   `yaml:"session_secret"`
	SessionMaxAge  int      `yaml:"session_max_age"`
	CookieSecure   string   `yaml:"cookie_secure"`   // "auto", "true", "false"
	CookieSameSite string   `yaml:"cookie_samesite"` // "strict", "lax", "none"
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

	// Override with environment variables if set
	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		cfg.Server.BaseURL = baseURL
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
	if c.OAuth.SessionSecret == "" || strings.Contains(c.OAuth.SessionSecret, "${") {
		return fmt.Errorf("oauth.session_secret is required (set SESSION_SECRET environment variable)")
	}
	if len(c.OAuth.SessionSecret) < 32 {
		return fmt.Errorf("oauth.session_secret must be at least 32 characters")
	}
	if len(c.OAuth.Scopes) == 0 {
		return fmt.Errorf("oauth.scopes must contain at least one scope")
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

// GetBaseURL returns the base URL for OAuth
// Uses base_url if set, otherwise constructs from host:port
func (c *Config) GetBaseURL() string {
	if c.Server.BaseURL != "" {
		return c.Server.BaseURL
	}
	return fmt.Sprintf("http://%s", c.GetAddr())
}

// IsHTTPS returns true if the base URL uses HTTPS
func (c *Config) IsHTTPS() bool {
	return strings.HasPrefix(strings.ToLower(c.GetBaseURL()), "https://")
}
