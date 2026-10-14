package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Kameleo KameleoConfig `yaml:"kameleo"`
	Network NetworkConfig `yaml:"network"`
	Browser BrowserConfig `yaml:"browser"`
	Pacing  PacingConfig  `yaml:"pacing"`
	Session SessionConfig `yaml:"session"`
	Retry   RetryConfig   `yaml:"retry"`
	Captcha CaptchaConfig `yaml:"captcha"`
	Storage StorageConfig `yaml:"storage"`
	Filter  FilterConfig  `yaml:"filter"`
	Logging LoggingConfig `yaml:"logging"`
}

type KameleoConfig struct {
	Endpoint          string `yaml:"endpoint"`
	ProfileNamePrefix string `yaml:"profile_name_prefix"`
	DeviceType        string `yaml:"device_type"`
	BrowserProduct    string `yaml:"browser_product"`
	ConnectTimeoutMS  int    `yaml:"connect_timeout_ms"`
	Language          string `yaml:"language"`
	Storage           string `yaml:"storage"`
}

type NetworkConfig struct {
	Mode                   string `yaml:"mode"`
	VerifyIPBeforeSearch   bool   `yaml:"verify_ip_before_search"`
	IPCheckURL             string `yaml:"ip_check_url"`
	CooldownOnBlockMinutes int    `yaml:"cooldown_on_block_minutes"`
}

type BrowserConfig struct {
	GoogleBaseURL       string `yaml:"google_base_url"`
	NavigationTimeoutMS int    `yaml:"navigation_timeout_ms"`
	PageSettleMS        int    `yaml:"page_settle_ms"`
	ResultsPerPage      int    `yaml:"results_per_page"`
	MaxPages            int    `yaml:"max_pages"`
	HumanSearch         bool   `yaml:"human_search"`
	WarmSession         bool   `yaml:"warm_session"`
	TypingDelayMS       int    `yaml:"typing_delay_ms"`
}

type PacingConfig struct {
	Enabled             bool `yaml:"enabled"`
	MinActionDelayMS    int  `yaml:"min_action_delay_ms"`
	MaxActionDelayMS    int  `yaml:"max_action_delay_ms"`
	MaxSearchesPerHour  int  `yaml:"max_searches_per_hour"`
}

type SessionConfig struct {
	RotateProfileAfterSearches int  `yaml:"rotate_profile_after_searches"`
	RotateProfileAfterCaptchas int  `yaml:"rotate_profile_after_captchas"`
	IdleProfileTTLMinutes      int  `yaml:"idle_profile_ttl_minutes"`
	ReuseProfiles              bool `yaml:"reuse_profiles"`
}

type RetryConfig struct {
	MaxAttempts      int     `yaml:"max_attempts"`
	InitialBackoffMS int     `yaml:"initial_backoff_ms"`
	MaxBackoffMS     int     `yaml:"max_backoff_ms"`
	Multiplier       float64 `yaml:"multiplier"`
}

type CaptchaTiersConfig struct {
	Script   bool `yaml:"script"`
	Token    bool `yaml:"token"`
	Provider bool `yaml:"provider"`
}

type CaptchaConfig struct {
	Enabled          bool               `yaml:"enabled"`
	MaxSolveAttempts int                `yaml:"max_solve_attempts"`
	Solver           string             `yaml:"solver"`
	SidecarEndpoint  string             `yaml:"sidecar_endpoint"`
	SidecarTimeoutMS int                `yaml:"sidecar_timeout_ms"`
	DetectTimeoutMS  int                `yaml:"detect_timeout_ms"`
	Tiers            CaptchaTiersConfig `yaml:"tiers"`
}

type StorageConfig struct {
	OutputDir   string `yaml:"output_dir"`
	ResultsFile string `yaml:"results_file"`
	JobStateDir string `yaml:"job_state_dir"`
}

type FilterConfig struct {
	IncludeKeywords     []string `yaml:"include_keywords"`
	ExcludeKeywords     []string `yaml:"exclude_keywords"`
	IncludeDomains      []string `yaml:"include_domains"`
	ExcludeDomains      []string `yaml:"exclude_domains"`
	IncludeURLPatterns  []string `yaml:"include_url_patterns"`
	ExcludeURLPatterns  []string `yaml:"exclude_url_patterns"`
	IncludePathPatterns []string `yaml:"include_path_patterns"`
	ExcludePathPatterns []string `yaml:"exclude_path_patterns"`
	IncludeQueryParams  []string `yaml:"include_query_params"`
	ExcludeQueryParams  []string `yaml:"exclude_query_params"`
	CustomRules         []string `yaml:"custom_rules"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Default() *Config {
	return &Config{
		Kameleo: KameleoConfig{
			Endpoint:          "http://localhost:5050",
			ProfileNamePrefix: "parser",
			DeviceType:        "desktop",
			BrowserProduct:    "chrome",
			ConnectTimeoutMS:  90000,
			Language:          "en-US,en",
			Storage:           "local",
		},
		Network: NetworkConfig{
			Mode:                   "hotspot",
			VerifyIPBeforeSearch:   true,
			IPCheckURL:             "https://api.ipify.org?format=text",
			CooldownOnBlockMinutes: 45,
		},
		Browser: BrowserConfig{
			GoogleBaseURL:       "https://www.google.com",
			NavigationTimeoutMS: 60000,
			PageSettleMS:        1500,
			ResultsPerPage:      10,
			MaxPages:            100,
			HumanSearch:         true,
			WarmSession:         true,
			TypingDelayMS:       45,
		},
		Pacing: PacingConfig{
			Enabled:            true,
			MinActionDelayMS:   2000,
			MaxActionDelayMS:   5000,
			MaxSearchesPerHour: 25,
		},
		Session: SessionConfig{
			RotateProfileAfterSearches: 50,
			RotateProfileAfterCaptchas: 0,
			IdleProfileTTLMinutes:      120,
			ReuseProfiles:              true,
		},
		Retry: RetryConfig{
			MaxAttempts:      3,
			InitialBackoffMS: 3000,
			MaxBackoffMS:     60000,
			Multiplier:       2.0,
		},
		Captcha: CaptchaConfig{
			Enabled:          true,
			MaxSolveAttempts: 1,
			Solver:           "sidecar",
			SidecarEndpoint:  "http://127.0.0.1:8787/solve",
			SidecarTimeoutMS: 180000,
			DetectTimeoutMS:  5000,
			Tiers: CaptchaTiersConfig{
				Script:   true,
				Token:    true,
				Provider: true,
			},
		},
		Storage: StorageConfig{
			OutputDir:   "./output",
			ResultsFile: "results.txt",
			JobStateDir: "./state",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func (c *Config) Validate() error {
	if c.Kameleo.Endpoint == "" {
		return fmt.Errorf("kameleo.endpoint is required")
	}
	if c.Browser.ResultsPerPage <= 0 {
		return fmt.Errorf("browser.results_per_page must be > 0")
	}
	if c.Browser.MaxPages <= 0 {
		return fmt.Errorf("browser.max_pages must be > 0")
	}
	if c.Retry.MaxAttempts <= 0 {
		return fmt.Errorf("retry.max_attempts must be > 0")
	}
	return nil
}

func (c *Config) ConnectTimeout() time.Duration {
	return time.Duration(c.Kameleo.ConnectTimeoutMS) * time.Millisecond
}

func (c *Config) NavigationTimeout() time.Duration {
	return time.Duration(c.Browser.NavigationTimeoutMS) * time.Millisecond
}

func (c *Config) PageSettle() time.Duration {
	return time.Duration(c.Browser.PageSettleMS) * time.Millisecond
}

func (c *Config) InitialBackoff() time.Duration {
	return time.Duration(c.Retry.InitialBackoffMS) * time.Millisecond
}

func (c *Config) MaxBackoff() time.Duration {
	return time.Duration(c.Retry.MaxBackoffMS) * time.Millisecond
}

func (c *Config) SidecarTimeout() time.Duration {
	if c.Captcha.SidecarTimeoutMS <= 0 {
		return 180 * time.Second
	}
	return time.Duration(c.Captcha.SidecarTimeoutMS) * time.Millisecond
}

func (c *Config) CooldownOnBlock() time.Duration {
	if c.Network.CooldownOnBlockMinutes <= 0 {
		return 45 * time.Minute
	}
	return time.Duration(c.Network.CooldownOnBlockMinutes) * time.Minute
}

func (c *Config) TypingDelay() time.Duration {
	if c.Browser.TypingDelayMS <= 0 {
		return 45 * time.Millisecond
	}
	return time.Duration(c.Browser.TypingDelayMS) * time.Millisecond
}
