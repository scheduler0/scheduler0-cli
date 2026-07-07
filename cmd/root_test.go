package cmd

import (
	"testing"
	"time"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActor(t *testing.T) {
	// Sourced from the Clerk user id captured at login.
	got, err := actor(&config.Config{ClerkUserID: "user_123"})
	require.NoError(t, err)
	assert.Equal(t, "user_123", got)

	// Missing identity points the user at login.
	_, err = actor(&config.Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheduler0 login")

	_, err = actor(nil)
	require.Error(t, err)
}

// clearSessionEnv ensures ambient SCHEDULER0_* env vars in the test runner's
// environment can't leak into tests exercising the "no session at all" path.
func clearSessionEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{config.EnvBaseURL, config.EnvAPIKey, config.EnvAPISecret, config.EnvAccountID, config.EnvActor, config.EnvExpiresAt} {
		t.Setenv(name, "")
	}
}

func TestGetClientConfig_NoConfig(t *testing.T) {
	// Point HOME at an empty temp dir so no config file exists.
	t.Setenv("HOME", t.TempDir())
	clearSessionEnv(t)
	baseURL = ""
	accountID = ""

	_, err := GetClientConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheduler0 login")
	assert.Contains(t, err.Error(), config.EnvAPIKey)
}

func TestGetClientConfig_FromEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no config file either way
	clearSessionEnv(t)
	baseURL = ""
	accountID = ""

	t.Setenv(config.EnvAPIKey, "env-key")
	t.Setenv(config.EnvAPISecret, "env-secret")
	t.Setenv(config.EnvAccountID, "77")
	t.Setenv(config.EnvActor, "github-actions")

	cfg, err := GetClientConfig()
	require.NoError(t, err)
	assert.Equal(t, "env-key", cfg.APIKey)
	assert.Equal(t, "env-secret", cfg.APISecret)
	assert.Equal(t, "77", cfg.AccountID)
	assert.Equal(t, config.DefaultBaseURL, cfg.BaseURL)
	assert.True(t, cfg.EnvSourced)

	got, err := actor(cfg)
	require.NoError(t, err)
	assert.Equal(t, "github-actions", got)
}

func TestGetClientConfig_FromEnv_Incomplete(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	clearSessionEnv(t)
	baseURL = ""
	accountID = ""

	// Only two of the three required vars set.
	t.Setenv(config.EnvAPIKey, "env-key")
	t.Setenv(config.EnvAPISecret, "env-secret")

	_, err := GetClientConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), config.EnvAccountID)
}

func TestGetClientConfig_EnvOverridesFileSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	clearSessionEnv(t)
	baseURL = ""
	accountID = ""

	require.NoError(t, config.SaveConfig(&config.Config{
		BaseURL: "https://api.file.example.com", APIKey: "file-key", APISecret: "file-secret",
		AccountID: "1", ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}))

	t.Setenv(config.EnvAPIKey, "env-key")
	t.Setenv(config.EnvAPISecret, "env-secret")
	t.Setenv(config.EnvAccountID, "77")

	cfg, err := GetClientConfig()
	require.NoError(t, err)
	assert.Equal(t, "env-key", cfg.APIKey, "env vars should take priority over an on-disk session")
}

func TestActor_EnvSourced(t *testing.T) {
	_, err := actor(&config.Config{EnvSourced: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), config.EnvActor)
}
