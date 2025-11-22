// Package config provides configuration management for the CLI Proxy API server.
// It handles loading and parsing YAML configuration files, and provides structured
// access to application settings including server port, authentication directory,
// debug settings, proxy configuration, and API keys.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// Config represents the application's configuration, loaded from a YAML file.
type Config struct {
	config.SDKConfig `yaml:",inline"`
	// Port is the network port on which the API server will listen.
	Port int `yaml:"port" json:"-"`

	// AmpUpstreamURL defines the upstream Amp control plane used for non-provider calls.
	AmpUpstreamURL string `yaml:"amp-upstream-url" json:"amp-upstream-url"`

	// AmpUpstreamAPIKey optionally overrides the Authorization header when proxying Amp upstream calls.
	AmpUpstreamAPIKey string `yaml:"amp-upstream-api-key" json:"amp-upstream-api-key"`

	// AmpRestrictManagementToLocalhost restricts Amp management routes (/api/user, /api/threads, etc.)
	// to only accept connections from localhost (127.0.0.1, ::1). When true, prevents drive-by
	// browser attacks and remote access to management endpoints. Default: true (recommended).
	AmpRestrictManagementToLocalhost bool `yaml:"amp-restrict-management-to-localhost" json:"amp-restrict-management-to-localhost"`

	// AuthDir is the directory where authentication token files are stored.
	AuthDir string `yaml:"auth-dir" json:"-"`

	// Debug enables or disables debug-level logging and other debug features.
	Debug bool `yaml:"debug" json:"debug"`

	// LoggingToFile controls whether application logs are written to rotating files or stdout.
	LoggingToFile bool `yaml:"logging-to-file" json:"logging-to-file"`

	// UsageStatisticsEnabled toggles in-memory usage aggregation; when false, usage data is discarded.
	UsageStatisticsEnabled bool `yaml:"usage-statistics-enabled" json:"usage-statistics-enabled"`

	// DisableCooling disables quota cooldown scheduling when true.
	DisableCooling bool `yaml:"disable-cooling" json:"disable-cooling"`

	// QuotaExceeded defines the behavior when a quota is exceeded.
	QuotaExceeded QuotaExceeded `yaml:"quota-exceeded" json:"quota-exceeded"`

	// WebsocketAuth enables or disables authentication for the WebSocket API.
	WebsocketAuth bool `yaml:"ws-auth" json:"ws-auth"`

	// GlAPIKey exposes the legacy generative language API key list for backward compatibility.
	GlAPIKey []string `yaml:"generative-language-api-key" json:"generative-language-api-key"`

	// GeminiKey defines Gemini API key configurations with optional routing overrides.
	GeminiKey []GeminiKey `yaml:"gemini-api-key" json:"gemini-api-key"`

	// RequestRetry defines the retry times when the request failed.
	RequestRetry int `yaml:"request-retry" json:"request-retry"`

	// ClaudeKey defines a list of Claude API key configurations as specified in the YAML configuration file.
	ClaudeKey []ClaudeKey `yaml:"claude-api-key" json:"claude-api-key"`

	// Codex defines a list of Codex API key configurations as specified in the YAML configuration file.
	CodexKey []CodexKey `yaml:"codex-api-key" json:"codex-api-key"`

	// OpenAICompatibility defines OpenAI API compatibility configurations for external providers.
	OpenAICompatibility []OpenAICompatibility `yaml:"openai-compatibility" json:"openai-compatibility"`

	// RemoteManagement nests management-related options under 'remote-management'.
	RemoteManagement RemoteManagement `yaml:"remote-management" json:"-"`

	// Payload defines default and override rules for provider payload parameters.
	Payload PayloadConfig `yaml:"payload" json:"payload"`

	// Providers defines the unified provider configuration for the new architecture.
	Providers ProvidersConfig `yaml:"providers" json:"providers"`

	// Models defines model routing and capabilities configuration.
	Models ModelsConfig `yaml:"models" json:"models"`

	// Server defines additional server configuration.
	Server ServerConfig `yaml:"server" json:"server"`

	// Websocket defines WebSocket relay configuration.
	Websocket WebsocketConfig `yaml:"websocket" json:"websocket"`

	// Logging defines structured logging configuration.
	Logging LoggingConfig `yaml:"logging" json:"logging"`

	// Advanced defines advanced configuration options.
	Advanced AdvancedConfig `yaml:"advanced" json:"advanced"`
}

// RemoteManagement holds management API configuration under 'remote-management'.
type RemoteManagement struct {
	// AllowRemote toggles remote (non-localhost) access to management API.
	AllowRemote bool `yaml:"allow-remote"`
	// SecretKey is the management key (plaintext or bcrypt hashed). YAML key intentionally 'secret-key'.
	SecretKey string `yaml:"secret-key"`
	// DisableControlPanel skips serving and syncing the bundled management UI when true.
	DisableControlPanel bool `yaml:"disable-control-panel"`
}

// QuotaExceeded defines the behavior when API quota limits are exceeded.
// It provides configuration options for automatic failover mechanisms.
type QuotaExceeded struct {
	// SwitchProject indicates whether to automatically switch to another project when a quota is exceeded.
	SwitchProject bool `yaml:"switch-project" json:"switch-project"`

	// SwitchPreviewModel indicates whether to automatically switch to a preview model when a quota is exceeded.
	SwitchPreviewModel bool `yaml:"switch-preview-model" json:"switch-preview-model"`
}

// PayloadConfig defines default and override parameter rules applied to provider payloads.
type PayloadConfig struct {
	// Default defines rules that only set parameters when they are missing in the payload.
	Default []PayloadRule `yaml:"default" json:"default"`
	// Override defines rules that always set parameters, overwriting any existing values.
	Override []PayloadRule `yaml:"override" json:"override"`
}

// PayloadRule describes a single rule targeting a list of models with parameter updates.
type PayloadRule struct {
	// Models lists model entries with name pattern and protocol constraint.
	Models []PayloadModelRule `yaml:"models" json:"models"`
	// Params maps JSON paths (gjson/sjson syntax) to values written into the payload.
	Params map[string]any `yaml:"params" json:"params"`
}

// PayloadModelRule ties a model name pattern to a specific translator protocol.
type PayloadModelRule struct {
	// Name is the model name or wildcard pattern (e.g., "gpt-*", "*-5", "gemini-*-pro").
	Name string `yaml:"name" json:"name"`
	// Protocol restricts the rule to a specific translator format (e.g., "gemini", "responses").
	Protocol string `yaml:"protocol" json:"protocol"`
}

// ClaudeKey represents the configuration for a Claude API key,
// including the API key itself and an optional base URL for the API endpoint.
type ClaudeKey struct {
	// APIKey is the authentication key for accessing Claude API services.
	APIKey string `yaml:"api-key" json:"api-key"`

	// BaseURL is the base URL for the Claude API endpoint.
	// If empty, the default Claude API URL will be used.
	BaseURL string `yaml:"base-url" json:"base-url"`

	// ProxyURL overrides the global proxy setting for this API key if provided.
	ProxyURL string `yaml:"proxy-url" json:"proxy-url"`

	// Models defines upstream model names and aliases for request routing.
	Models []ClaudeModel `yaml:"models" json:"models"`

	// Headers optionally adds extra HTTP headers for requests sent with this key.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// ClaudeModel describes a mapping between an alias and the actual upstream model name.
type ClaudeModel struct {
	// Name is the upstream model identifier used when issuing requests.
	Name string `yaml:"name" json:"name"`

	// Alias is the client-facing model name that maps to Name.
	Alias string `yaml:"alias" json:"alias"`
}

// CodexKey represents the configuration for a Codex API key,
// including the API key itself and an optional base URL for the API endpoint.
type CodexKey struct {
	// APIKey is the authentication key for accessing Codex API services.
	APIKey string `yaml:"api-key" json:"api-key"`

	// BaseURL is the base URL for the Codex API endpoint.
	// If empty, the default Codex API URL will be used.
	BaseURL string `yaml:"base-url" json:"base-url"`

	// ProxyURL overrides the global proxy setting for this API key if provided.
	ProxyURL string `yaml:"proxy-url" json:"proxy-url"`

	// Headers optionally adds extra HTTP headers for requests sent with this key.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// GeminiKey represents the configuration for a Gemini API key,
// including optional overrides for upstream base URL, proxy routing, and headers.
type GeminiKey struct {
	// APIKey is the authentication key for accessing Gemini API services.
	APIKey string `yaml:"api-key" json:"api-key"`

	// BaseURL optionally overrides the Gemini API endpoint.
	BaseURL string `yaml:"base-url,omitempty" json:"base-url,omitempty"`

	// ProxyURL optionally overrides the global proxy for this API key.
	ProxyURL string `yaml:"proxy-url,omitempty" json:"proxy-url,omitempty"`

	// Headers optionally adds extra HTTP headers for requests sent with this key.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// OpenAICompatibility represents the configuration for OpenAI API compatibility
// with external providers, allowing model aliases to be routed through OpenAI API format.
type OpenAICompatibility struct {
	// Name is the identifier for this OpenAI compatibility configuration.
	Name string `yaml:"name" json:"name"`

	// BaseURL is the base URL for the external OpenAI-compatible API endpoint.
	BaseURL string `yaml:"base-url" json:"base-url"`

	// APIKeys are the authentication keys for accessing the external API services.
	// Deprecated: Use APIKeyEntries instead to support per-key proxy configuration.
	APIKeys []string `yaml:"api-keys,omitempty" json:"api-keys,omitempty"`

	// APIKeyEntries defines API keys with optional per-key proxy configuration.
	APIKeyEntries []OpenAICompatibilityAPIKey `yaml:"api-key-entries,omitempty" json:"api-key-entries,omitempty"`

	// Models defines the model configurations including aliases for routing.
	Models []OpenAICompatibilityModel `yaml:"models" json:"models"`

	// Headers optionally adds extra HTTP headers for requests sent to this provider.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// OpenAICompatibilityAPIKey represents an API key configuration with optional proxy setting.
type OpenAICompatibilityAPIKey struct {
	// APIKey is the authentication key for accessing the external API services.
	APIKey string `yaml:"api-key" json:"api-key"`

	// ProxyURL overrides the global proxy setting for this API key if provided.
	ProxyURL string `yaml:"proxy-url,omitempty" json:"proxy-url,omitempty"`
}

// OpenAICompatibilityModel represents a model configuration for OpenAI compatibility,
// including the actual model name and its alias for API routing.
type OpenAICompatibilityModel struct {
	// Name is the actual model name used by the external provider.
	Name string `yaml:"name" json:"name"`

	// Alias is the model name alias that clients will use to reference this model.
	Alias string `yaml:"alias" json:"alias"`
}

// ============================================
// NEW UNIFIED ARCHITECTURE CONFIGURATION
// ============================================

// ServerConfig defines additional server configuration options.
type ServerConfig struct {
	// ReadTimeout is the maximum duration for reading the entire request.
	ReadTimeout int `yaml:"read_timeout" json:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes of the response.
	WriteTimeout int `yaml:"write_timeout" json:"write_timeout"`

	// Host is the host interface to bind to (default: "0.0.0.0").
	Host string `yaml:"host" json:"host"`
}

// WebsocketConfig defines WebSocket relay configuration.
type WebsocketConfig struct {
	// Enabled toggles WebSocket relay functionality.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Path is the WebSocket endpoint path (default: "/v1/ws").
	Path string `yaml:"path" json:"path"`

	// Timeout is the maximum duration for WebSocket connections in seconds.
	Timeout int `yaml:"timeout" json:"timeout"`

	// PingInterval is the interval for sending ping messages in seconds.
	PingInterval int `yaml:"ping_interval" json:"ping_interval"`

	// MaxMessageSize is the maximum message size in bytes.
	MaxMessageSize int `yaml:"max_message_size" json:"max_message_size"`

	// Auth toggles authentication for WebSocket connections.
	Auth bool `yaml:"auth" json:"auth"`
}

// LoggingConfig defines structured logging configuration.
type LoggingConfig struct {
	// Level is the logging level: debug, info, warn, error.
	Level string `yaml:"level" json:"level"`

	// ToFile toggles writing logs to files instead of stdout.
	ToFile bool `yaml:"to_file" json:"to_file"`

	// Dir is the directory for log files.
	Dir string `yaml:"dir" json:"dir"`

	// MaxSizeMB is the maximum size of a log file in megabytes.
	MaxSizeMB int `yaml:"max_size_mb" json:"max_size_mb"`

	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int `yaml:"max_backups" json:"max_backups"`

	// MaxAgeDays is the maximum age of log files in days.
	MaxAgeDays int `yaml:"max_age_days" json:"max_age_days"`

	// Compress toggles compression of old log files.
	Compress bool `yaml:"compress" json:"compress"`
}

// ProvidersConfig defines all provider configurations.
type ProvidersConfig struct {
	// LLMux contains LLMux OAuth provider configurations.
	LLMux LLMuxConfig `yaml:"llmux" json:"llmux"`

	// Ctonew contains ctonew provider configuration.
	Ctonew CtonewConfig `yaml:"ctonew" json:"ctonew"`

	// AIstudio contains AIstudio browser automation configuration.
	AIstudio AIstudioConfig `yaml:"aistudio" json:"aistudio"`

	// WebAI contains WebAI service configuration.
	WebAI WebAIConfig `yaml:"webai" json:"webai"`
}

// LLMuxConfig defines LLMux OAuth provider configurations.
type LLMuxConfig struct {
	// ClaudePro contains Claude Pro OAuth configuration.
	ClaudePro LLMuxProviderConfig `yaml:"claude_pro" json:"claude_pro"`

	// ChatGPTPlus contains ChatGPT Plus OAuth configuration.
	ChatGPTPlus LLMuxProviderConfig `yaml:"chatgpt_plus" json:"chatgpt_plus"`
}

// LLMuxProviderConfig defines configuration for a single LLMux provider.
type LLMuxProviderConfig struct {
	// Enabled toggles this provider.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// AutoStart loads this provider automatically on startup.
	AutoStart bool `yaml:"auto_start" json:"auto_start"`

	// OAuth contains OAuth configuration.
	OAuth OAuthConfig `yaml:"oauth" json:"oauth"`

	// Models lists models available through this provider.
	Models []string `yaml:"models" json:"models"`
}

// OAuthConfig defines OAuth 2.0 configuration.
type OAuthConfig struct {
	// ClientID is the OAuth client ID (empty = use default).
	ClientID string `yaml:"client_id" json:"client_id"`

	// RedirectPort is the local callback server port (0 = random).
	RedirectPort int `yaml:"redirect_port" json:"redirect_port"`

	// Scopes are the OAuth scopes to request.
	Scopes []string `yaml:"scopes" json:"scopes"`
}

// CtonewConfig defines ctonew provider configuration.
type CtonewConfig struct {
	// Enabled toggles this provider.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// AutoStart loads this provider automatically on startup.
	AutoStart bool `yaml:"auto_start" json:"auto_start"`

	// Clerk contains Clerk-specific configuration.
	Clerk ClerkConfig `yaml:"clerk" json:"clerk"`

	// Models lists models available through this provider.
	Models []string `yaml:"models" json:"models"`
}

// ClerkConfig defines Clerk authentication configuration.
type ClerkConfig struct {
	// APIURL is the Clerk API base URL.
	APIURL string `yaml:"api_url" json:"api_url"`
}

// AIstudioConfig defines AIstudio service configuration.
type AIstudioConfig struct {
	// Enabled toggles this provider.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// AutoStart starts the service automatically on router startup.
	AutoStart bool `yaml:"auto_start" json:"auto_start"`

	// Service contains service process configuration.
	Service ServiceConfig `yaml:"service" json:"service"`

	// HealthCheck contains health check configuration.
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check"`

	// Browser contains browser automation configuration.
	Browser BrowserConfig `yaml:"browser" json:"browser"`

	// Models lists models available through this provider.
	Models []string `yaml:"models" json:"models"`
}

// WebAIConfig defines WebAI service configuration.
type WebAIConfig struct {
	// Enabled toggles this provider.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// AutoStart starts the service automatically on router startup.
	AutoStart bool `yaml:"auto_start" json:"auto_start"`

	// Service contains service process configuration.
	Service ServiceConfig `yaml:"service" json:"service"`

	// HealthCheck contains health check configuration.
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check"`

	// Proxy contains HTTP proxy configuration.
	Proxy ProxyConfig `yaml:"proxy" json:"proxy"`

	// Models lists models available through this provider.
	Models []string `yaml:"models" json:"models"`
}

// ServiceConfig defines external service process configuration.
type ServiceConfig struct {
	// Command is the command to execute.
	Command string `yaml:"command" json:"command"`

	// Cwd is the working directory for the command.
	Cwd string `yaml:"cwd" json:"cwd"`

	// Port is the port the service listens on (for HTTP services).
	Port int `yaml:"port" json:"port"`

	// Env contains environment variables for the service.
	Env map[string]string `yaml:"env" json:"env"`
}

// HealthCheckConfig defines health check configuration.
type HealthCheckConfig struct {
	// Enabled toggles health checking.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// URL is the health check endpoint (for HTTP services).
	URL string `yaml:"url" json:"url"`

	// Interval is the check interval in seconds.
	Interval int `yaml:"interval" json:"interval"`

	// Timeout is the check timeout in seconds.
	Timeout int `yaml:"timeout" json:"timeout"`

	// MaxFailures is the number of failures before restart.
	MaxFailures int `yaml:"max_failures" json:"max_failures"`
}

// BrowserConfig defines browser automation configuration.
type BrowserConfig struct {
	// Type is the browser type: camoufox, playwright.
	Type string `yaml:"type" json:"type"`

	// Headless toggles headless mode.
	Headless bool `yaml:"headless" json:"headless"`

	// IdleTimeout is the browser idle timeout in seconds.
	IdleTimeout int `yaml:"idle_timeout" json:"idle_timeout"`

	// MaxInstances is the maximum concurrent browser instances.
	MaxInstances int `yaml:"max_instances" json:"max_instances"`
}

// ProxyConfig defines HTTP proxy configuration.
type ProxyConfig struct {
	// Endpoint is the proxy endpoint URL.
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// Timeout is the proxy request timeout in seconds.
	Timeout int `yaml:"timeout" json:"timeout"`

	// Retry is the number of retry attempts.
	Retry int `yaml:"retry" json:"retry"`
}

// ModelsConfig defines model routing and capabilities.
type ModelsConfig struct {
	// Routing defines model routing rules.
	Routing []ModelRoutingRule `yaml:"routing" json:"routing"`

	// Defaults defines default model capabilities.
	Defaults ModelCapabilities `yaml:"defaults" json:"defaults"`

	// Overrides defines per-model capability overrides.
	Overrides map[string]ModelCapabilities `yaml:"overrides" json:"overrides"`
}

// ModelRoutingRule defines a single routing rule.
type ModelRoutingRule struct {
	// Pattern is the regex pattern to match model names.
	Pattern string `yaml:"pattern" json:"pattern"`

	// Providers lists providers to try in order.
	Providers []string `yaml:"providers" json:"providers"`
}

// ModelCapabilities defines model capabilities.
type ModelCapabilities struct {
	// MaxTokens is the maximum token limit.
	MaxTokens int `yaml:"max_tokens" json:"max_tokens"`

	// SupportsStreaming indicates streaming support.
	SupportsStreaming bool `yaml:"supports_streaming" json:"supports_streaming"`

	// SupportsVision indicates vision/image support.
	SupportsVision bool `yaml:"supports_vision" json:"supports_vision"`

	// SupportsReasoning indicates reasoning/chain-of-thought support.
	SupportsReasoning bool `yaml:"supports_reasoning" json:"supports_reasoning"`
}

// AdvancedConfig defines advanced configuration options.
type AdvancedConfig struct {
	// RequestLogging configures request/response logging.
	RequestLogging RequestLoggingConfig `yaml:"request_logging" json:"request_logging"`

	// RateLimiting configures rate limiting.
	RateLimiting RateLimitingConfig `yaml:"rate_limiting" json:"rate_limiting"`

	// Failover configures automatic failover.
	Failover FailoverConfig `yaml:"failover" json:"failover"`

	// ServiceManagement configures service lifecycle management.
	ServiceManagement ServiceManagementConfig `yaml:"service_management" json:"service_management"`

	// QuotaExceeded defines quota exceeded behavior (legacy compatibility).
	QuotaExceeded QuotaExceeded `yaml:"quota_exceeded" json:"quota_exceeded"`
}

// RequestLoggingConfig defines request/response logging configuration.
type RequestLoggingConfig struct {
	// Enabled toggles request/response logging.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// SanitizeTokens removes tokens from logs.
	SanitizeTokens bool `yaml:"sanitize_tokens" json:"sanitize_tokens"`
}

// RateLimitingConfig defines rate limiting configuration.
type RateLimitingConfig struct {
	// Enabled toggles rate limiting.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// RequestsPerMinute is the maximum requests per minute.
	RequestsPerMinute int `yaml:"requests_per_minute" json:"requests_per_minute"`
}

// FailoverConfig defines failover configuration.
type FailoverConfig struct {
	// Enabled toggles automatic failover.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MaxRetries is the maximum retry attempts per provider.
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// RetryDelayMs is the delay between retries in milliseconds.
	RetryDelayMs int `yaml:"retry_delay_ms" json:"retry_delay_ms"`
}

// ServiceManagementConfig defines service management configuration.
type ServiceManagementConfig struct {
	// AutoRestart toggles automatic service restart on failure.
	AutoRestart bool `yaml:"auto_restart" json:"auto_restart"`

	// RestartDelayS is the delay before restart in seconds.
	RestartDelayS int `yaml:"restart_delay_s" json:"restart_delay_s"`

	// MaxRestarts is the maximum restart attempts before giving up.
	MaxRestarts int `yaml:"max_restarts" json:"max_restarts"`
}

// LoadConfig reads a YAML configuration file from the given path,
// unmarshals it into a Config struct, applies environment variable overrides,
// and returns it.
//
// Parameters:
//   - configFile: The path to the YAML configuration file
//
// Returns:
//   - *Config: The loaded configuration
//   - error: An error if the configuration could not be loaded
func LoadConfig(configFile string) (*Config, error) {
	return LoadConfigOptional(configFile, false)
}

// LoadConfigOptional reads YAML from configFile.
// If optional is true and the file is missing, it returns an empty Config.
// If optional is true and the file is empty or invalid, it returns an empty Config.
func LoadConfigOptional(configFile string, optional bool) (*Config, error) {
	// Read the entire configuration file into memory.
	data, err := os.ReadFile(configFile)
	if err != nil {
		if optional {
			if os.IsNotExist(err) || errors.Is(err, syscall.EISDIR) {
				// Missing and optional: return empty config (cloud deploy standby).
				return &Config{}, nil
			}
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// In cloud deploy mode (optional=true), if file is empty or contains only whitespace, return empty config.
	if optional && len(data) == 0 {
		return &Config{}, nil
	}

	// Unmarshal the YAML data into the Config struct.
	var cfg Config
	// Set defaults before unmarshal so that absent keys keep defaults.
	cfg.LoggingToFile = false
	cfg.UsageStatisticsEnabled = false
	cfg.DisableCooling = false
	cfg.AmpRestrictManagementToLocalhost = true // Default to secure: only localhost access
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		if optional {
			// In cloud deploy mode, if YAML parsing fails, return empty config instead of error.
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Hash remote management key if plaintext is detected (nested)
	// We consider a value to be already hashed if it looks like a bcrypt hash ($2a$, $2b$, or $2y$ prefix).
	if cfg.RemoteManagement.SecretKey != "" && !looksLikeBcrypt(cfg.RemoteManagement.SecretKey) {
		hashed, errHash := hashSecret(cfg.RemoteManagement.SecretKey)
		if errHash != nil {
			return nil, fmt.Errorf("failed to hash remote management key: %w", errHash)
		}
		cfg.RemoteManagement.SecretKey = hashed

		// Persist the hashed value back to the config file to avoid re-hashing on next startup.
		// Preserve YAML comments and ordering; update only the nested key.
		_ = SaveConfigPreserveCommentsUpdateNestedScalar(configFile, []string{"remote-management", "secret-key"}, hashed)
	}

	// Sync request authentication providers with inline API keys for backwards compatibility.
	syncInlineAccessProvider(&cfg)

	// Sanitize Gemini API key configuration and migrate legacy entries.
	cfg.SanitizeGeminiKeys()

	// Sanitize Codex keys: drop entries without base-url
	cfg.SanitizeCodexKeys()

	// Sanitize Claude key headers
	cfg.SanitizeClaudeKeys()

	// Sanitize OpenAI compatibility providers: drop entries without base-url
	cfg.SanitizeOpenAICompatibility()

	// Return the populated configuration struct.
	return &cfg, nil
}

// SanitizeOpenAICompatibility removes OpenAI-compatibility provider entries that are
// not actionable, specifically those missing a BaseURL. It trims whitespace before
// evaluation and preserves the relative order of remaining entries.
func (cfg *Config) SanitizeOpenAICompatibility() {
	if cfg == nil || len(cfg.OpenAICompatibility) == 0 {
		return
	}
	out := make([]OpenAICompatibility, 0, len(cfg.OpenAICompatibility))
	for i := range cfg.OpenAICompatibility {
		e := cfg.OpenAICompatibility[i]
		e.Name = strings.TrimSpace(e.Name)
		e.BaseURL = strings.TrimSpace(e.BaseURL)
		e.Headers = NormalizeHeaders(e.Headers)
		if e.BaseURL == "" {
			// Skip providers with no base-url; treated as removed
			continue
		}
		out = append(out, e)
	}
	cfg.OpenAICompatibility = out
}

// SanitizeCodexKeys removes Codex API key entries missing a BaseURL.
// It trims whitespace and preserves order for remaining entries.
func (cfg *Config) SanitizeCodexKeys() {
	if cfg == nil || len(cfg.CodexKey) == 0 {
		return
	}
	out := make([]CodexKey, 0, len(cfg.CodexKey))
	for i := range cfg.CodexKey {
		e := cfg.CodexKey[i]
		e.BaseURL = strings.TrimSpace(e.BaseURL)
		e.Headers = NormalizeHeaders(e.Headers)
		if e.BaseURL == "" {
			continue
		}
		out = append(out, e)
	}
	cfg.CodexKey = out
}

// SanitizeClaudeKeys normalizes headers for Claude credentials.
func (cfg *Config) SanitizeClaudeKeys() {
	if cfg == nil || len(cfg.ClaudeKey) == 0 {
		return
	}
	for i := range cfg.ClaudeKey {
		entry := &cfg.ClaudeKey[i]
		entry.Headers = NormalizeHeaders(entry.Headers)
	}
}

// SanitizeGeminiKeys deduplicates and normalizes Gemini credentials.
func (cfg *Config) SanitizeGeminiKeys() {
	if cfg == nil {
		return
	}

	seen := make(map[string]struct{}, len(cfg.GeminiKey))
	out := cfg.GeminiKey[:0]
	for i := range cfg.GeminiKey {
		entry := cfg.GeminiKey[i]
		entry.APIKey = strings.TrimSpace(entry.APIKey)
		if entry.APIKey == "" {
			continue
		}
		entry.BaseURL = strings.TrimSpace(entry.BaseURL)
		entry.ProxyURL = strings.TrimSpace(entry.ProxyURL)
		entry.Headers = NormalizeHeaders(entry.Headers)
		if _, exists := seen[entry.APIKey]; exists {
			continue
		}
		seen[entry.APIKey] = struct{}{}
		out = append(out, entry)
	}
	cfg.GeminiKey = out

	if len(cfg.GlAPIKey) > 0 {
		for _, raw := range cfg.GlAPIKey {
			key := strings.TrimSpace(raw)
			if key == "" {
				continue
			}
			if _, exists := seen[key]; exists {
				continue
			}
			cfg.GeminiKey = append(cfg.GeminiKey, GeminiKey{APIKey: key})
			seen[key] = struct{}{}
		}
	}

	cfg.GlAPIKey = nil
}

func syncInlineAccessProvider(cfg *Config) {
	if cfg == nil {
		return
	}
	if len(cfg.APIKeys) == 0 {
		if provider := cfg.ConfigAPIKeyProvider(); provider != nil && len(provider.APIKeys) > 0 {
			cfg.APIKeys = append([]string(nil), provider.APIKeys...)
		}
	}
	cfg.Access.Providers = nil
}

// looksLikeBcrypt returns true if the provided string appears to be a bcrypt hash.
func looksLikeBcrypt(s string) bool {
	return len(s) > 4 && (s[:4] == "$2a$" || s[:4] == "$2b$" || s[:4] == "$2y$")
}

// NormalizeHeaders trims header keys and values and removes empty pairs.
func NormalizeHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	clean := make(map[string]string, len(headers))
	for k, v := range headers {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" || val == "" {
			continue
		}
		clean[key] = val
	}
	if len(clean) == 0 {
		return nil
	}
	return clean
}

// hashSecret hashes the given secret using bcrypt.
func hashSecret(secret string) (string, error) {
	// Use default cost for simplicity.
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// SaveConfigPreserveComments writes the config back to YAML while preserving existing comments
// and key ordering by loading the original file into a yaml.Node tree and updating values in-place.
func SaveConfigPreserveComments(configFile string, cfg *Config) error {
	persistCfg := sanitizeConfigForPersist(cfg)
	// Load original YAML as a node tree to preserve comments and ordering.
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	var original yaml.Node
	if err = yaml.Unmarshal(data, &original); err != nil {
		return err
	}
	if original.Kind != yaml.DocumentNode || len(original.Content) == 0 {
		return fmt.Errorf("invalid yaml document structure")
	}
	if original.Content[0] == nil || original.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("expected root mapping node")
	}

	// Marshal the current cfg to YAML, then unmarshal to a yaml.Node we can merge from.
	rendered, err := yaml.Marshal(persistCfg)
	if err != nil {
		return err
	}
	var generated yaml.Node
	if err = yaml.Unmarshal(rendered, &generated); err != nil {
		return err
	}
	if generated.Kind != yaml.DocumentNode || len(generated.Content) == 0 || generated.Content[0] == nil {
		return fmt.Errorf("invalid generated yaml structure")
	}
	if generated.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("expected generated root mapping node")
	}

	// Remove deprecated auth block before merging to avoid persisting it again.
	removeMapKey(original.Content[0], "auth")
	removeLegacyOpenAICompatAPIKeys(original.Content[0])

	// Merge generated into original in-place, preserving comments/order of existing nodes.
	mergeMappingPreserve(original.Content[0], generated.Content[0])
	normalizeCollectionNodeStyles(original.Content[0])

	// Write back.
	f, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err = enc.Encode(&original); err != nil {
		_ = enc.Close()
		return err
	}
	if err = enc.Close(); err != nil {
		return err
	}
	data = NormalizeCommentIndentation(buf.Bytes())
	_, err = f.Write(data)
	return err
}

func sanitizeConfigForPersist(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.SDKConfig = cfg.SDKConfig
	clone.SDKConfig.Access = config.AccessConfig{}
	return &clone
}

// SaveConfigPreserveCommentsUpdateNestedScalar updates a nested scalar key path like ["a","b"]
// while preserving comments and positions.
func SaveConfigPreserveCommentsUpdateNestedScalar(configFile string, path []string, value string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	var root yaml.Node
	if err = yaml.Unmarshal(data, &root); err != nil {
		return err
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return fmt.Errorf("invalid yaml document structure")
	}
	node := root.Content[0]
	// descend mapping nodes following path
	for i, key := range path {
		if i == len(path)-1 {
			// set final scalar
			v := getOrCreateMapValue(node, key)
			v.Kind = yaml.ScalarNode
			v.Tag = "!!str"
			v.Value = value
		} else {
			next := getOrCreateMapValue(node, key)
			if next.Kind != yaml.MappingNode {
				next.Kind = yaml.MappingNode
				next.Tag = "!!map"
			}
			node = next
		}
	}
	f, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err = enc.Encode(&root); err != nil {
		_ = enc.Close()
		return err
	}
	if err = enc.Close(); err != nil {
		return err
	}
	data = NormalizeCommentIndentation(buf.Bytes())
	_, err = f.Write(data)
	return err
}

// NormalizeCommentIndentation removes indentation from standalone YAML comment lines to keep them left aligned.
func NormalizeCommentIndentation(data []byte) []byte {
	lines := bytes.Split(data, []byte("\n"))
	changed := false
	for i, line := range lines {
		trimmed := bytes.TrimLeft(line, " \t")
		if len(trimmed) == 0 || trimmed[0] != '#' {
			continue
		}
		if len(trimmed) == len(line) {
			continue
		}
		lines[i] = append([]byte(nil), trimmed...)
		changed = true
	}
	if !changed {
		return data
	}
	return bytes.Join(lines, []byte("\n"))
}

// getOrCreateMapValue finds the value node for a given key in a mapping node.
// If not found, it appends a new key/value pair and returns the new value node.
func getOrCreateMapValue(mapNode *yaml.Node, key string) *yaml.Node {
	if mapNode.Kind != yaml.MappingNode {
		mapNode.Kind = yaml.MappingNode
		mapNode.Tag = "!!map"
		mapNode.Content = nil
	}
	for i := 0; i+1 < len(mapNode.Content); i += 2 {
		k := mapNode.Content[i]
		if k.Value == key {
			return mapNode.Content[i+1]
		}
	}
	// append new key/value
	mapNode.Content = append(mapNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key})
	val := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ""}
	mapNode.Content = append(mapNode.Content, val)
	return val
}

// mergeMappingPreserve merges keys from src into dst mapping node while preserving
// key order and comments of existing keys in dst. Unknown keys from src are appended
// to dst at the end, copying their node structure from src.
func mergeMappingPreserve(dst, src *yaml.Node) {
	if dst == nil || src == nil {
		return
	}
	if dst.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		// If kinds do not match, prefer replacing dst with src semantics in-place
		// but keep dst node object to preserve any attached comments at the parent level.
		copyNodeShallow(dst, src)
		return
	}
	// Build a lookup of existing keys in dst
	for i := 0; i+1 < len(src.Content); i += 2 {
		sk := src.Content[i]
		sv := src.Content[i+1]
		idx := findMapKeyIndex(dst, sk.Value)
		if idx >= 0 {
			// Merge into existing value node
			dv := dst.Content[idx+1]
			mergeNodePreserve(dv, sv)
		} else {
			if shouldSkipEmptyCollectionOnPersist(sk.Value, sv) {
				continue
			}
			// Append new key/value pair by deep-copying from src
			dst.Content = append(dst.Content, deepCopyNode(sk), deepCopyNode(sv))
		}
	}
}

// mergeNodePreserve merges src into dst for scalars, mappings and sequences while
// reusing destination nodes to keep comments and anchors. For sequences, it updates
// in-place by index.
func mergeNodePreserve(dst, src *yaml.Node) {
	if dst == nil || src == nil {
		return
	}
	switch src.Kind {
	case yaml.MappingNode:
		if dst.Kind != yaml.MappingNode {
			copyNodeShallow(dst, src)
		}
		mergeMappingPreserve(dst, src)
	case yaml.SequenceNode:
		// Preserve explicit null style if dst was null and src is empty sequence
		if dst.Kind == yaml.ScalarNode && dst.Tag == "!!null" && len(src.Content) == 0 {
			// Keep as null to preserve original style
			return
		}
		if dst.Kind != yaml.SequenceNode {
			dst.Kind = yaml.SequenceNode
			dst.Tag = "!!seq"
			dst.Content = nil
		}
		reorderSequenceForMerge(dst, src)
		// Update elements in place
		minContent := len(dst.Content)
		if len(src.Content) < minContent {
			minContent = len(src.Content)
		}
		for i := 0; i < minContent; i++ {
			if dst.Content[i] == nil {
				dst.Content[i] = deepCopyNode(src.Content[i])
				continue
			}
			mergeNodePreserve(dst.Content[i], src.Content[i])
		}
		// Append any extra items from src
		for i := len(dst.Content); i < len(src.Content); i++ {
			dst.Content = append(dst.Content, deepCopyNode(src.Content[i]))
		}
		// Truncate if dst has extra items not in src
		if len(src.Content) < len(dst.Content) {
			dst.Content = dst.Content[:len(src.Content)]
		}
	case yaml.ScalarNode, yaml.AliasNode:
		// For scalars, update Tag and Value but keep Style from dst to preserve quoting
		dst.Kind = src.Kind
		dst.Tag = src.Tag
		dst.Value = src.Value
		// Keep dst.Style as-is intentionally
	case 0:
		// Unknown/empty kind; do nothing
	default:
		// Fallback: replace shallowly
		copyNodeShallow(dst, src)
	}
}

// findMapKeyIndex returns the index of key node in dst mapping (index of key, not value).
// Returns -1 when not found.
func findMapKeyIndex(mapNode *yaml.Node, key string) int {
	if mapNode == nil || mapNode.Kind != yaml.MappingNode {
		return -1
	}
	for i := 0; i+1 < len(mapNode.Content); i += 2 {
		if mapNode.Content[i] != nil && mapNode.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func shouldSkipEmptyCollectionOnPersist(key string, node *yaml.Node) bool {
	switch key {
	case "generative-language-api-key",
		"gemini-api-key",
		"claude-api-key",
		"codex-api-key",
		"openai-compatibility":
		return isEmptyCollectionNode(node)
	default:
		return false
	}
}

func isEmptyCollectionNode(node *yaml.Node) bool {
	if node == nil {
		return true
	}
	switch node.Kind {
	case yaml.SequenceNode:
		return len(node.Content) == 0
	case yaml.ScalarNode:
		return node.Tag == "!!null"
	default:
		return false
	}
}

// deepCopyNode creates a deep copy of a yaml.Node graph.
func deepCopyNode(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	cp := *n
	if len(n.Content) > 0 {
		cp.Content = make([]*yaml.Node, len(n.Content))
		for i := range n.Content {
			cp.Content[i] = deepCopyNode(n.Content[i])
		}
	}
	return &cp
}

// copyNodeShallow copies type/tag/value and resets content to match src, but
// keeps the same destination node pointer to preserve parent relations/comments.
func copyNodeShallow(dst, src *yaml.Node) {
	if dst == nil || src == nil {
		return
	}
	dst.Kind = src.Kind
	dst.Tag = src.Tag
	dst.Value = src.Value
	// Replace content with deep copy from src
	if len(src.Content) > 0 {
		dst.Content = make([]*yaml.Node, len(src.Content))
		for i := range src.Content {
			dst.Content[i] = deepCopyNode(src.Content[i])
		}
	} else {
		dst.Content = nil
	}
}

func reorderSequenceForMerge(dst, src *yaml.Node) {
	if dst == nil || src == nil {
		return
	}
	if len(dst.Content) == 0 {
		return
	}
	if len(src.Content) == 0 {
		return
	}
	original := append([]*yaml.Node(nil), dst.Content...)
	used := make([]bool, len(original))
	ordered := make([]*yaml.Node, len(src.Content))
	for i := range src.Content {
		if idx := matchSequenceElement(original, used, src.Content[i]); idx >= 0 {
			ordered[i] = original[idx]
			used[idx] = true
		}
	}
	dst.Content = ordered
}

func matchSequenceElement(original []*yaml.Node, used []bool, target *yaml.Node) int {
	if target == nil {
		return -1
	}
	switch target.Kind {
	case yaml.MappingNode:
		id := sequenceElementIdentity(target)
		if id != "" {
			for i := range original {
				if used[i] || original[i] == nil || original[i].Kind != yaml.MappingNode {
					continue
				}
				if sequenceElementIdentity(original[i]) == id {
					return i
				}
			}
		}
	case yaml.ScalarNode:
		val := strings.TrimSpace(target.Value)
		if val != "" {
			for i := range original {
				if used[i] || original[i] == nil || original[i].Kind != yaml.ScalarNode {
					continue
				}
				if strings.TrimSpace(original[i].Value) == val {
					return i
				}
			}
		}
	default:
	}
	// Fallback to structural equality to preserve nodes lacking explicit identifiers.
	for i := range original {
		if used[i] || original[i] == nil {
			continue
		}
		if nodesStructurallyEqual(original[i], target) {
			return i
		}
	}
	return -1
}

func sequenceElementIdentity(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.MappingNode {
		return ""
	}
	identityKeys := []string{"id", "name", "alias", "api-key", "api_key", "apikey", "key", "provider", "model"}
	for _, k := range identityKeys {
		if v := mappingScalarValue(node, k); v != "" {
			return k + "=" + v
		}
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if keyNode == nil || valNode == nil || valNode.Kind != yaml.ScalarNode {
			continue
		}
		val := strings.TrimSpace(valNode.Value)
		if val != "" {
			return strings.ToLower(strings.TrimSpace(keyNode.Value)) + "=" + val
		}
	}
	return ""
}

func mappingScalarValue(node *yaml.Node, key string) string {
	if node == nil || node.Kind != yaml.MappingNode {
		return ""
	}
	lowerKey := strings.ToLower(key)
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if keyNode == nil || valNode == nil || valNode.Kind != yaml.ScalarNode {
			continue
		}
		if strings.ToLower(strings.TrimSpace(keyNode.Value)) == lowerKey {
			return strings.TrimSpace(valNode.Value)
		}
	}
	return ""
}

func nodesStructurallyEqual(a, b *yaml.Node) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case yaml.MappingNode:
		if len(a.Content) != len(b.Content) {
			return false
		}
		for i := 0; i+1 < len(a.Content); i += 2 {
			if !nodesStructurallyEqual(a.Content[i], b.Content[i]) {
				return false
			}
			if !nodesStructurallyEqual(a.Content[i+1], b.Content[i+1]) {
				return false
			}
		}
		return true
	case yaml.SequenceNode:
		if len(a.Content) != len(b.Content) {
			return false
		}
		for i := range a.Content {
			if !nodesStructurallyEqual(a.Content[i], b.Content[i]) {
				return false
			}
		}
		return true
	case yaml.ScalarNode:
		return strings.TrimSpace(a.Value) == strings.TrimSpace(b.Value)
	case yaml.AliasNode:
		return nodesStructurallyEqual(a.Alias, b.Alias)
	default:
		return strings.TrimSpace(a.Value) == strings.TrimSpace(b.Value)
	}
}

func removeMapKey(mapNode *yaml.Node, key string) {
	if mapNode == nil || mapNode.Kind != yaml.MappingNode || key == "" {
		return
	}
	for i := 0; i+1 < len(mapNode.Content); i += 2 {
		if mapNode.Content[i] != nil && mapNode.Content[i].Value == key {
			mapNode.Content = append(mapNode.Content[:i], mapNode.Content[i+2:]...)
			return
		}
	}
}

func removeLegacyOpenAICompatAPIKeys(root *yaml.Node) {
	if root == nil || root.Kind != yaml.MappingNode {
		return
	}
	idx := findMapKeyIndex(root, "openai-compatibility")
	if idx < 0 || idx+1 >= len(root.Content) {
		return
	}
	seq := root.Content[idx+1]
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return
	}
	for i := range seq.Content {
		if seq.Content[i] != nil && seq.Content[i].Kind == yaml.MappingNode {
			removeMapKey(seq.Content[i], "api-keys")
		}
	}
}

// normalizeCollectionNodeStyles forces YAML collections to use block notation, keeping
// lists and maps readable. Empty sequences retain flow style ([]) so empty list markers
// remain compact.
func normalizeCollectionNodeStyles(node *yaml.Node) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.MappingNode:
		node.Style = 0
		for i := range node.Content {
			normalizeCollectionNodeStyles(node.Content[i])
		}
	case yaml.SequenceNode:
		if len(node.Content) == 0 {
			node.Style = yaml.FlowStyle
		} else {
			node.Style = 0
		}
		for i := range node.Content {
			normalizeCollectionNodeStyles(node.Content[i])
		}
	default:
		// Scalars keep their existing style to preserve quoting
	}
}
