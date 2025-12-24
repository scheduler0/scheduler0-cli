package cmd

import (
	"testing"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMaskString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "short string",
			input:    "abc",
			expected: "****",
		},
		{
			name:     "4 characters",
			input:    "abcd",
			expected: "****",
		},
		{
			name:     "long string",
			input:    "abcdefghijklmnop",
			expected: "ab****op",
		},
		{
			name:     "very long string",
			input:    "this-is-a-very-long-api-key-that-should-be-masked",
			expected: "th****ed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRunInit_SavesConfig(t *testing.T) {
	// This test requires mocking file system operations
	// For unit tests, we focus on the validation logic
	// Integration tests can verify the full save/load flow
	t.Skip("Skipping test that requires file system mocking")
}

func TestRunInit_ValidatesConfig(t *testing.T) {
	// Test config validation directly
	cfg := &config.Config{
		BaseURL: "https://api.test.com",
		// Missing required fields
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_key is required")
}
