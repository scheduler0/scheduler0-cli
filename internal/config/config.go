package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	configDirName  = ".scheduler0"
	configFileName = "config.json"

	// DefaultAppURL is the Scheduler0 web app used for browser-based login.
	DefaultAppURL = "https://app.scheduler0.com"
	// DefaultBaseURL is the Scheduler0 API endpoint for the hosted product.
	DefaultBaseURL = "https://api.scheduler0.com"

	// expirySkew treats a session as expired slightly early to avoid racing the
	// server's own expiry check under clock drift.
	expirySkew = 30 * time.Second
)

// Environment variable names recognized by LoadConfigFromEnv, for CI use where
// an interactive `scheduler0 login` isn't possible.
const (
	EnvBaseURL   = "SCHEDULER0_BASE_URL"
	EnvAPIKey    = "SCHEDULER0_API_KEY"
	EnvAPISecret = "SCHEDULER0_API_SECRET"
	EnvAccountID = "SCHEDULER0_ACCOUNT_ID"
	// EnvActor supplies the identity recorded for created/modified/deleted/
	// archived fields. A CI credential isn't tied to a Clerk user, so this is a
	// free-form string (e.g. "github-actions").
	EnvActor = "SCHEDULER0_ACTOR"
	// EnvExpiresAt is optional (RFC3339). If unset, an env-sourced session is
	// treated as valid regardless of expiry and the server is left to enforce
	// it (a truly expired credential is rejected with 401 on the first call).
	EnvExpiresAt = "SCHEDULER0_EXPIRES_AT"
)

// LocalExecutorEntry holds the single persisted local executor for this machine.
type LocalExecutorEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Command    string `json:"command,omitempty"`
	WorkingDir string `json:"working_dir,omitempty"`
}

// Config holds the CLI's persisted state: the API/app endpoints and the
// short-lived session credential obtained via `scheduler0 login`. Requests are
// authenticated with the api key/secret + account id (the same headers as before);
// the session simply carries an expiry so the CLI can prompt for re-login.
type Config struct {
	// BaseURL is the Scheduler0 API endpoint.
	BaseURL string `json:"base_url"`
	// AppURL is the Scheduler0 web app used for browser login (defaults to DefaultAppURL).
	AppURL string `json:"app_url,omitempty"`

	// Session credential minted by the login flow.
	APIKey      string   `json:"api_key,omitempty"`
	APISecret   string   `json:"api_secret,omitempty"`
	AccountID   string   `json:"account_id,omitempty"`
	ExpiresAt   string   `json:"expires_at,omitempty"` // RFC3339
	ClerkUserID string   `json:"clerk_user_id,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`

	// LocalExecutor is the single executor registered on this machine (one per machine).
	LocalExecutor *LocalExecutorEntry `json:"local_executor,omitempty"`

	// EnvSourced is true when this Config came from LoadConfigFromEnv (CI)
	// rather than the on-disk session file. Never persisted; see IsSessionValid.
	EnvSourced bool `json:"-"`
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, configDirName)
	return filepath.Join(configDir, configFileName), nil
}

// LoadConfig loads the configuration from the config file
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found. Please run 'scheduler0 login' to sign in")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// LoadConfigFromEnv builds a Config from SCHEDULER0_* environment variables, so
// CI environments can authenticate without an interactive `scheduler0 login`.
// It returns ok=false when none of the credential variables are set at all, so
// callers can fall back to the on-disk session; if only some of
// EnvAPIKey/EnvAPISecret/EnvAccountID are set, ok is still true and the
// resulting Config simply fails IsSessionValid with a clear cause.
func LoadConfigFromEnv() (*Config, bool) {
	apiKey := os.Getenv(EnvAPIKey)
	apiSecret := os.Getenv(EnvAPISecret)
	accountID := os.Getenv(EnvAccountID)
	if apiKey == "" && apiSecret == "" && accountID == "" {
		return nil, false
	}

	baseURL := os.Getenv(EnvBaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Config{
		BaseURL:     baseURL,
		APIKey:      apiKey,
		APISecret:   apiSecret,
		AccountID:   accountID,
		ClerkUserID: os.Getenv(EnvActor),
		ExpiresAt:   os.Getenv(EnvExpiresAt),
		EnvSourced:  true,
	}, true
}

// SaveConfig saves the configuration to the config file
func SaveConfig(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate ensures the config carries a usable, unexpired session credential.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if c.APIKey == "" || c.APISecret == "" || c.AccountID == "" {
		return fmt.Errorf("not signed in: run 'scheduler0 login'")
	}
	if !c.IsSessionValid() {
		return fmt.Errorf("session expired: run 'scheduler0 login'")
	}
	return nil
}

// AppBaseURL returns the configured web app URL, or the default.
func (c *Config) AppBaseURL() string {
	if c.AppURL != "" {
		return c.AppURL
	}
	return DefaultAppURL
}

// SessionExpiry returns the parsed ExpiresAt and whether it was present/valid.
func (c *Config) SessionExpiry() (time.Time, bool) {
	if c.ExpiresAt == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, c.ExpiresAt)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// IsSessionValid reports whether a session credential is present and not expired.
// A missing/blank ExpiresAt on a file-sourced session is treated as expired so a
// truncated config forces login; an env-sourced (CI) session without an
// explicit expiry is instead treated as valid, leaving the server to enforce
// actual expiry (see EnvExpiresAt).
func (c *Config) IsSessionValid() bool {
	if c.APIKey == "" || c.APISecret == "" || c.AccountID == "" {
		return false
	}
	expiry, ok := c.SessionExpiry()
	if !ok {
		return c.EnvSourced
	}
	return time.Now().Add(expirySkew).Before(expiry)
}

// ClearSession removes the stored session credential, leaving endpoints and the
// local executor registration intact.
func (c *Config) ClearSession() {
	c.APIKey = ""
	c.APISecret = ""
	c.AccountID = ""
	c.ExpiresAt = ""
	c.ClerkUserID = ""
	c.Scopes = nil
}

// loadOrCreateConfig loads the config from disk, or returns an empty Config if the file does not
// exist yet (unlike LoadConfig which errors on missing file).
func loadOrCreateConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// SetLocalExecutor persists the local executor entry, creating the config file if necessary.
// Calling it again overwrites the previous entry (one executor per machine).
func SetLocalExecutor(entry LocalExecutorEntry) error {
	cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}
	cfg.LocalExecutor = &entry
	return SaveConfig(cfg)
}

// GetLocalExecutor returns the persisted local executor entry, or nil if none has been registered.
func GetLocalExecutor() (*LocalExecutorEntry, error) {
	cfg, err := loadOrCreateConfig()
	if err != nil {
		return nil, err
	}
	return cfg.LocalExecutor, nil
}
