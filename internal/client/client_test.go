package client

import (
	"net/url"
	"testing"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.test.com",
		APIKey:    "test-key",
		APISecret: "test-secret",
		AccountID: "123",
		AuthType:  "api_key",
	}

	cl, err := NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, cl)

	// Verify client configuration
	assert.Equal(t, "test-key", cl.APIKey)
	assert.Equal(t, "test-secret", cl.APISecret)
	assert.Equal(t, "123", cl.AccountID)
	assert.Equal(t, "v1", cl.Version)

	// Verify base URL
	expectedURL, _ := url.Parse("https://api.test.com")
	assert.Equal(t, expectedURL.String(), cl.BaseURL.String())
}

func TestNewClient_InvalidURL(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "not-a-valid-url",
		APIKey:    "test-key",
		APISecret: "test-secret",
		AccountID: "123",
		AuthType:  "api_key",
	}

	cl, err := NewClient(cfg)
	// URL parsing might succeed but be invalid, so we just check it doesn't panic
	// The actual validation happens when making requests
	assert.NotNil(t, cl)
	assert.NoError(t, err)
}

func TestNewClient_BasicAuth(t *testing.T) {
	cfg := &config.Config{
		BaseURL:  "http://localhost:7070",
		Username: "admin",
		Password: "secret",
		AuthType: "basic",
	}	cl, err := NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, cl)	// Verify client configuration
	assert.Equal(t, "admin", cl.Username)
	assert.Equal(t, "secret", cl.Password)
	assert.Equal(t, "v1", cl.Version)	// Verify base URL
	expectedURL, _ := url.Parse("http://localhost:7070")
	assert.Equal(t, expectedURL.String(), cl.BaseURL.String())
}
