package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCommand returns a throwaway *cobra.Command carrying a bounded
// context, so tests exercising runLoginDevice/runLoginLoopback don't need (and
// don't mutate) the package-level loginCmd singleton.
func newTestCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	cmd.SetContext(ctx)
	return cmd
}

func TestIsHeadlessSSHSession(t *testing.T) {
	// No SSH env vars at all: never headless, regardless of platform.
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CLIENT", "")
	assert.False(t, isHeadlessSSHSession())

	// SSH session present. On non-linux we always treat it as headless; on
	// linux it depends on DISPLAY/WAYLAND_DISPLAY, which we can't force here
	// without affecting the current OS, so only assert the non-linux case
	// directly and otherwise just exercise the linux DISPLAY branch.
	t.Setenv("SSH_TTY", "/dev/pts/0")
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	assert.True(t, isHeadlessSSHSession(), "SSH session with no display should be headless")

	t.Setenv("DISPLAY", ":0")
	if got := isHeadlessSSHSession(); got {
		// Only linux is expected to honor DISPLAY; other platforms always
		// treat any SSH session as headless.
		t.Logf("isHeadlessSSHSession()=true with DISPLAY set (expected off-linux)")
	}
}

func TestPostToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/cli/token", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(tokenResponse{
				APIKey: "key", APISecret: "secret", AccountID: "1",
			})
		}))
		defer server.Close()

		tok, apiErr, err := postToken(context.Background(), server.URL, map[string]string{"code": "abc"})
		require.NoError(t, err)
		assert.Empty(t, apiErr)
		require.NotNil(t, tok)
		assert.Equal(t, "key", tok.APIKey)
	})

	t.Run("structured error is not a go error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
		}))
		defer server.Close()

		tok, apiErr, err := postToken(context.Background(), server.URL, map[string]string{"device_code": "abc"})
		require.NoError(t, err)
		assert.Nil(t, tok)
		assert.Equal(t, "authorization_pending", apiErr)
	})

	t.Run("malformed success response is a go error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{}) // missing api_key etc.
		}))
		defer server.Close()

		tok, apiErr, err := postToken(context.Background(), server.URL, nil)
		assert.Error(t, err)
		assert.Empty(t, apiErr)
		assert.Nil(t, tok)
	})
}

func TestRequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/cli/device/code", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(deviceCodeResponse{
			DeviceCode:              "devcode123",
			UserCode:                "ABCD-EFGH",
			VerificationURI:         "https://app.example.com/cli/device",
			VerificationURIComplete: "https://app.example.com/cli/device?user_code=ABCD-EFGH",
			ExpiresIn:               600,
			Interval:                5,
		})
	}))
	defer server.Close()

	dc, err := requestDeviceCode(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, "devcode123", dc.DeviceCode)
	assert.Equal(t, "ABCD-EFGH", dc.UserCode)
	assert.EqualValues(t, 600, dc.ExpiresIn)
}

func TestRunLoginDevice_PollsUntilApproved(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var pollCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli/device/code":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(deviceCodeResponse{
				DeviceCode: "devcode123",
				UserCode:   "ABCD-EFGH",
				ExpiresIn:  60,
				Interval:   0, // exercise the minDevicePollInterval floor
			})
		case "/cli/token":
			n := atomic.AddInt32(&pollCount, 1)
			if n < 3 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(tokenResponse{
				APIKey: "final-key", APISecret: "final-secret", AccountID: "42",
				ClerkUserID: "user_1", ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	// A fresh, throwaway *cobra.Command carries the context runLoginDevice reads
	// via cmd.Context()/cmd.OutOrStdout(); it deliberately doesn't reuse the
	// package-level loginCmd singleton so tests can't mutate shared state.
	cmd := newTestCommand(t)

	err := runLoginDevice(cmd, server.URL, "https://api.example.com", nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&pollCount), int32(3))

	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "final-key", cfg.APIKey)
	assert.Equal(t, "42", cfg.AccountID)
	assert.Equal(t, "user_1", cfg.ClerkUserID)
}

func TestRunLoginDevice_ExpiredCode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli/device/code":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(deviceCodeResponse{
				DeviceCode: "devcode123",
				UserCode:   "ABCD-EFGH",
				ExpiresIn:  60,
				Interval:   0,
			})
		case "/cli/token":
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "expired_token"})
		}
	}))
	defer server.Close()

	cmd := newTestCommand(t)
	err := runLoginDevice(cmd, server.URL, "https://api.example.com", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestRunLoginDevice_AccessDenied(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli/device/code":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(deviceCodeResponse{
				DeviceCode: "devcode123",
				UserCode:   "ABCD-EFGH",
				ExpiresIn:  60,
				Interval:   0,
			})
		case "/cli/token":
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "access_denied"})
		}
	}))
	defer server.Close()

	cmd := newTestCommand(t)
	err := runLoginDevice(cmd, server.URL, "https://api.example.com", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "denied")
}
