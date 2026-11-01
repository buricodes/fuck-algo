package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const EnvDevelopment = "development"

type Config struct {
	Server      Server      `yaml:"server"`
	WebsiteURL  string      `yaml:"website_url"`
	Trial       Trial       `yaml:"trial"`
	Limits      Limits      `yaml:"limits"`
	DB          DB          `yaml:"db"`
	AI          AI          `yaml:"ai"`
	Jev         Jev         `yaml:"jev"`
	ClaudeCLI   ClaudeCLI   `yaml:"claude_cli"`
	Anthropic   Anthropic   `yaml:"anthropic"`
	ElevenLabs  ElevenLabs  `yaml:"elevenlabs"`
	ChecksCache ChecksCache `yaml:"checks_cache"`
	RateLimits  RateLimits  `yaml:"rate_limits"`

	Env     string  `yaml:"-"`
	Secrets Secrets `yaml:"-"`

	ExtensionOrigins       []string `yaml:"-"`
	ClerkAuthorizedParties []string `yaml:"-"`
}

type Server struct {
	Port           int      `yaml:"port"`
	TrustedProxies []string `yaml:"trusted_proxies"`
}

type Trial struct {
	Voice  int `yaml:"voice"`
	Checks int `yaml:"checks"`
}

type Limits struct {
	BodyBytes           int64 `yaml:"body_bytes"`
	TranscribeBodyBytes int64 `yaml:"transcribe_body_bytes"`
	AudioBytes          int64 `yaml:"audio_bytes"`
}

type DB struct {
	MaxConns int32 `yaml:"max_conns"`
}

const (
	ProviderJev       = "jev"
	ProviderCLI       = "cli"
	ProviderAnthropic = "anthropic"
)

type AI struct {
	Checks string `yaml:"checks"`
	Rules  string `yaml:"rules"`
}

type Jev struct {
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MaxConcurrent  int    `yaml:"max_concurrent"`
}

type ClaudeCLI struct {
	Binary         string `yaml:"binary"`
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MaxConcurrent  int    `yaml:"max_concurrent"`
}

type Anthropic struct {
	Checks Model `yaml:"checks"`
	Rules  Model `yaml:"rules"`
}

type Model struct {
	Model          string `yaml:"model"`
	Thinking       string `yaml:"thinking"`
	MaxTokens      int    `yaml:"max_tokens"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MaxConcurrent  int    `yaml:"max_concurrent"`
}

type ElevenLabs struct {
	STTModel       string `yaml:"stt_model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MaxConcurrent  int    `yaml:"max_concurrent"`
}

type ChecksCache struct {
	TTLHours   int `yaml:"ttl_hours"`
	MaxEntries int `yaml:"max_entries"`
}

type RateLimits struct {
	Checkout        Rate `yaml:"checkout"`
	VoiceToken      Rate `yaml:"voice_token"`
	VoiceTranscribe Rate `yaml:"voice_transcribe"`
	VoiceRules      Rate `yaml:"voice_rules"`
	Checks          Rate `yaml:"checks"`
}

type Rate struct {
	PerMinute float64 `yaml:"per_minute"`
	Burst     float64 `yaml:"burst"`
}

type Secrets struct {
	DatabaseURL       string
	AnthropicKey      string
	OpenRouterKey     string
	ElevenLabsKey     string
	DodoAPIKey        string
	DodoWebhookSecret string
	DodoProductID     string
	ClerkSecretKey    string
}

func (c *Config) IsDevelopment() bool { return c.Env == EnvDevelopment }

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.applyEnv(); err != nil {
		return nil, err
	}
	return &cfg, cfg.validate()
}

func (c *Config) applyEnv() error {
	c.Env = envOr("APP_ENV", EnvDevelopment)
	if port := os.Getenv("PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("PORT: %w", err)
		}
		c.Server.Port = p
	}
	c.WebsiteURL = strings.TrimRight(envOr("WEBSITE_URL", c.WebsiteURL), "/")
	c.ExtensionOrigins = splitList(os.Getenv("EXTENSION_ORIGINS"))
	c.ClerkAuthorizedParties = splitList(os.Getenv("CLERK_AUTHORIZED_PARTIES"))
	c.AI.Checks = envOr("AI_CHECKS", c.AI.Checks)
	c.AI.Rules = envOr("AI_RULES", c.AI.Rules)
	c.Secrets = Secrets{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		AnthropicKey:      os.Getenv("ANTHROPIC_API_KEY"),
		OpenRouterKey:     os.Getenv("OPEN_ROUTER_KEY"),
		ElevenLabsKey:     os.Getenv("ELEVENLABS_API_KEY"),
		DodoAPIKey:        os.Getenv("DODO_PAYMENTS_API_KEY"),
		DodoWebhookSecret: os.Getenv("DODO_PAYMENTS_WEBHOOK_SECRET"),
		DodoProductID:     os.Getenv("DODO_PRODUCT_ID"),
		ClerkSecretKey:    os.Getenv("CLERK_SECRET_KEY"),
	}
	return nil
}

func (c *Config) validate() error {
	var errs []error
	if c.Trial.Voice < 0 || c.Trial.Checks < 0 {
		errs = append(errs, errors.New("trial limits must not be negative"))
	}
	if c.Limits.BodyBytes <= 0 || c.Limits.TranscribeBodyBytes < c.Limits.AudioBytes {
		errs = append(errs, errors.New("limits: body sizes must be positive and the transcribe body must fit the audio"))
	}
	if c.ChecksCache.MaxEntries < 1 || c.ChecksCache.TTLHours < 1 {
		errs = append(errs, errors.New("checks_cache needs a positive ttl and size"))
	}
	if !c.IsDevelopment() && c.Secrets.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required outside development"))
	}
	if !c.IsDevelopment() && c.Secrets.ClerkSecretKey == "" {
		errs = append(errs, errors.New("CLERK_SECRET_KEY is required outside development"))
	}
	if !c.IsDevelopment() && len(c.ClerkAuthorizedParties) == 0 {
		errs = append(errs, errors.New("CLERK_AUTHORIZED_PARTIES must list the allowed origins outside development"))
	}
	if !oneOf(c.AI.Checks, ProviderJev, ProviderAnthropic, ProviderCLI) {
		errs = append(errs, errors.New("ai.checks must be jev, anthropic or cli"))
	}
	if !oneOf(c.AI.Rules, ProviderAnthropic, ProviderCLI) {
		errs = append(errs, errors.New("ai.rules must be anthropic or cli"))
	}
	if c.WebsiteURL == "" {
		errs = append(errs, errors.New("website_url (or WEBSITE_URL) is required"))
	}
	return errors.Join(errs...)
}

func oneOf(value string, options ...string) bool {
	for _, o := range options {
		if value == o {
			return true
		}
	}
	return false
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, strings.TrimRight(part, "/"))
		}
	}
	return out
}
