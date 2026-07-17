package cmd

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	loginAppURL    string
	loginBaseURL   string
	loginNoBrowser bool
	loginDevice    bool
)

// loginTimeout bounds how long we wait for the user to complete the loopback
// browser flow. The device flow instead uses the server-provided expires_in.
const loginTimeout = 5 * time.Minute

// minDevicePollInterval floors the server-provided polling interval so a
// misbehaving or misconfigured server can't make the CLI hammer the endpoint.
const minDevicePollInterval = 3 * time.Second

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in to Scheduler0 in your browser",
	Long: `Sign in to Scheduler0 by authorizing the CLI in your browser.

This opens app.scheduler0.com, where you approve access using your existing
signed-in session. A short-lived credential is then stored locally and used for
subsequent commands. Re-run 'scheduler0 login' when it expires.

On a headless or remote machine (e.g. an SSH session into a server), there is
no local browser that can complete the redirect back to the CLI, so this
automatically falls back to a device-code flow: a short code is printed for you
to enter at a URL you open on any device (your laptop, your phone, ...). Force
this explicitly with --device, or pass --no-browser for the same effect.`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&loginAppURL, "app-url", "", "Scheduler0 web app URL (default https://app.scheduler0.com)")
	loginCmd.Flags().StringVar(&loginBaseURL, "base-url", "", "Scheduler0 API base URL (default https://api.scheduler0.com)")
	loginCmd.Flags().BoolVar(&loginNoBrowser, "no-browser", false, "Use the device-code flow instead of a local browser redirect (for headless/remote machines)")
	loginCmd.Flags().BoolVar(&loginDevice, "device", false, "Alias for --no-browser")
}

// callbackResult is delivered by the loopback handler once the browser returns.
type callbackResult struct {
	code string
	err  error
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Resolve endpoints: flag > existing config > default.
	existing, _ := config.LoadConfig() // ignore error; may be a first-time login
	appURL := firstNonEmpty(loginAppURL, cfgAppURL(existing), config.DefaultAppURL)
	apiURL := firstNonEmpty(loginBaseURL, cfgBaseURL(existing), config.DefaultBaseURL)

	if loginDevice || loginNoBrowser || isHeadlessSSHSession() {
		return runLoginDevice(cmd, appURL, apiURL, existing)
	}
	return runLoginLoopback(cmd, appURL, apiURL, existing)
}

// isHeadlessSSHSession reports whether we appear to be running inside an SSH
// session with no local GUI to open a browser in. This is only a heuristic
// (SSH with X11 forwarding or a remote desktop session would still have a
// usable display), so it only steers the default — --device/--no-browser
// always override it explicitly, and a false positive just means the user sees
// the device-code flow instead of a browser popping open.
func isHeadlessSSHSession() bool {
	sshSession := os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CLIENT") != ""
	if !sshSession {
		return false
	}
	if runtime.GOOS == "linux" {
		return os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == ""
	}
	return true
}

// runLoginLoopback implements the browser+PKCE flow: the CLI opens a local
// browser and receives the authorization code via a loopback redirect. This
// only works when the browser and the CLI process share a loopback interface
// (i.e. not over a plain SSH session — see runLoginDevice for that case).
func runLoginLoopback(cmd *cobra.Command, appURL, apiURL string, existing *config.Config) error {
	out := cmd.OutOrStdout()

	// PKCE + CSRF state.
	verifier, err := randomURLToken(48)
	if err != nil {
		return fmt.Errorf("failed to generate code verifier: %w", err)
	}
	challenge := pkceChallenge(verifier)
	state, err := randomURLToken(24)
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Loopback listener on a random free port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start local callback server: %w", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	resultCh := make(chan callbackResult, 1)
	server := &http.Server{Handler: loopbackHandler(state, resultCh)}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authorizeURL := buildAuthorizeURL(appURL, redirectURI, state, challenge)

	_, _ = fmt.Fprintln(out, "Opening your browser to sign in...")
	if err := openBrowser(authorizeURL); err != nil {
		_, _ = fmt.Fprintf(out, "\nCould not open a browser automatically. Visit this URL to authorize the CLI:\n\n%s\n\n", authorizeURL)
	}

	// Wait for the callback (or timeout / cancellation).
	ctx, cancel := context.WithTimeout(cmd.Context(), loginTimeout)
	defer cancel()

	var code string
	select {
	case <-ctx.Done():
		return fmt.Errorf("login timed out after %s; please try again", loginTimeout)
	case res := <-resultCh:
		if res.err != nil {
			return fmt.Errorf("login failed: %w", res.err)
		}
		code = res.code
	}

	// Exchange the one-time code for a credential (PKCE back-channel).
	tok, err := exchangeCode(ctx, appURL, code, verifier)
	if err != nil {
		return fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	return saveSessionAndReport(out, existing, appURL, apiURL, tok)
}

// saveSessionAndReport persists the minted credential (preserving any existing
// local executor registration) and prints a confirmation. Shared by both the
// loopback and device login flows.
func saveSessionAndReport(out io.Writer, existing *config.Config, appURL, apiURL string, tok *tokenResponse) error {
	cfg := existing
	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.BaseURL = apiURL
	cfg.AppURL = appURL
	cfg.APIKey = tok.APIKey
	cfg.APISecret = tok.APISecret
	cfg.AccountID = tok.AccountID
	cfg.ClerkUserID = tok.ClerkUserID
	cfg.ExpiresAt = tok.ExpiresAt
	cfg.Scopes = tok.Scopes

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	_, _ = fmt.Fprintf(out, "✓ Signed in. Account %s.", cfg.AccountID)
	if expiry, ok := cfg.SessionExpiry(); ok {
		_, _ = fmt.Fprintf(out, " Session valid until %s.", expiry.Local().Format(time.RFC1123))
	}
	_, _ = fmt.Fprintln(out)
	return nil
}

// loopbackHandler serves the browser redirect, validates state, and reports the code.
func loopbackHandler(expectedState string, resultCh chan<- callbackResult) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if errParam := q.Get("error"); errParam != "" {
			writeBrowserMessage(w, false, "Authorization was denied. You can close this tab.")
			resultCh <- callbackResult{err: fmt.Errorf("authorization denied (%s)", errParam)}
			return
		}
		if q.Get("state") != expectedState {
			writeBrowserMessage(w, false, "Sign-in could not be verified. You can close this tab.")
			resultCh <- callbackResult{err: fmt.Errorf("state mismatch; possible CSRF, aborting")}
			return
		}
		code := q.Get("code")
		if code == "" {
			writeBrowserMessage(w, false, "No authorization code was returned. You can close this tab.")
			resultCh <- callbackResult{err: fmt.Errorf("no authorization code in callback")}
			return
		}

		writeBrowserMessage(w, true, "You're signed in to the Scheduler0 CLI. You can close this tab and return to your terminal.")
		resultCh <- callbackResult{code: code}
	})
	return mux
}

type tokenResponse struct {
	APIKey      string   `json:"api_key"`
	APISecret   string   `json:"api_secret"`
	AccountID   string   `json:"account_id"`
	ClerkUserID string   `json:"clerk_user_id"`
	ExpiresAt   string   `json:"expires_at"`
	Scopes      []string `json:"scopes"`
}

// exchangeCode performs the back-channel token exchange for the PKCE/loopback
// flow. Unlike device polling, any structured API error here is terminal.
func exchangeCode(ctx context.Context, appURL, code, verifier string) (*tokenResponse, error) {
	tok, apiErr, err := postToken(ctx, appURL, map[string]string{"code": code, "code_verifier": verifier})
	if err != nil {
		return nil, err
	}
	if apiErr != "" {
		return nil, fmt.Errorf("token endpoint error: %s", apiErr)
	}
	return tok, nil
}

// postToken posts body to {appURL}/cli/token. On success it returns the
// decoded credential. On a structured {"error": "..."} response it returns the
// error string (apiErr) instead of a Go error, so callers like the device-flow
// poller can distinguish "keep waiting" (authorization_pending) from a genuine
// failure. A non-nil err means a transport-level or malformed-response failure.
func postToken(ctx context.Context, appURL string, body map[string]string) (tok *tokenResponse, apiErr string, err error) {
	tokenURL := strings.TrimRight(appURL, "/") + "/cli/token"
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(string(raw)))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(bodyBytes, &e)
		if e.Error != "" {
			return nil, e.Error, nil
		}
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if wwwAuth != "" {
			return nil, "", fmt.Errorf("token endpoint returned %d (WWW-Authenticate: %s) — the server requires HTTP Basic Auth; URL: %s", resp.StatusCode, wwwAuth, tokenURL)
		}
		bodySnippet := strings.TrimSpace(string(bodyBytes))
		if len(bodySnippet) > 120 {
			bodySnippet = bodySnippet[:120] + "…"
		}
		if bodySnippet != "" {
			return nil, "", fmt.Errorf("token endpoint returned %d; URL: %s; body: %s", resp.StatusCode, tokenURL, bodySnippet)
		}
		return nil, "", fmt.Errorf("token endpoint returned %d; URL: %s", resp.StatusCode, tokenURL)
	}

	var t tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, "", fmt.Errorf("failed to decode token response: %w", err)
	}
	if t.APIKey == "" || t.APISecret == "" || t.AccountID == "" {
		return nil, "", fmt.Errorf("token response missing credential fields")
	}
	return &t, "", nil
}

// deviceCodeResponse is the response from POST /cli/device/code.
type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

// requestDeviceCode asks the web app for a new device/user code pair.
func requestDeviceCode(ctx context.Context, appURL string) (*deviceCodeResponse, error) {
	deviceCodeURL := strings.TrimRight(appURL, "/") + "/cli/device/code"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceCodeURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(bodyBytes, &e)
		if e.Error != "" {
			return nil, fmt.Errorf("device code request failed: %s", e.Error)
		}
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if wwwAuth != "" {
			return nil, fmt.Errorf("device code endpoint returned %d (WWW-Authenticate: %s) — the server requires HTTP Basic Auth; URL: %s", resp.StatusCode, wwwAuth, deviceCodeURL)
		}
		bodySnippet := strings.TrimSpace(string(bodyBytes))
		if len(bodySnippet) > 120 {
			bodySnippet = bodySnippet[:120] + "…"
		}
		if bodySnippet != "" {
			return nil, fmt.Errorf("device code endpoint returned %d; URL: %s; body: %s", resp.StatusCode, deviceCodeURL, bodySnippet)
		}
		return nil, fmt.Errorf("device code endpoint returned %d; URL: %s", resp.StatusCode, deviceCodeURL)
	}

	var dc deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, fmt.Errorf("failed to decode device code response: %w", err)
	}
	if dc.DeviceCode == "" || dc.UserCode == "" {
		return nil, fmt.Errorf("device code response missing fields")
	}
	return &dc, nil
}

// runLoginDevice implements the headless/SSH-friendly device-authorization
// flow: a short code is printed for the user to enter at a URL opened on any
// device, and the CLI polls until that approval completes (or the code
// expires). No loopback listener or local browser is required.
func runLoginDevice(cmd *cobra.Command, appURL, apiURL string, existing *config.Config) error {
	out := cmd.OutOrStdout()

	reqCtx, reqCancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	dc, err := requestDeviceCode(reqCtx, appURL)
	reqCancel()
	if err != nil {
		return fmt.Errorf("failed to request a device code: %w", err)
	}

	_, _ = fmt.Fprintln(out, "This looks like a headless or remote session, so let's sign in with a device code.")
	_, _ = fmt.Fprintf(out, "\nFirst, copy your one-time code:\n\n    %s\n\n", dc.UserCode)
	_, _ = fmt.Fprintf(out, "Then open this URL in a browser on any device and enter the code:\n\n    %s\n\n", dc.VerificationURI)
	if dc.VerificationURIComplete != "" {
		_, _ = fmt.Fprintf(out, "Or open this link to skip typing the code:\n\n    %s\n\n", dc.VerificationURIComplete)
	}
	_, _ = fmt.Fprintln(out, "Waiting for approval...")

	interval := time.Duration(dc.Interval) * time.Second
	if interval < minDevicePollInterval {
		interval = minDevicePollInterval
	}
	deadline := time.Duration(dc.ExpiresIn) * time.Second
	if deadline <= 0 {
		deadline = loginTimeout
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), deadline)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("device code expired before it was approved; please run 'scheduler0 login' again")
		case <-ticker.C:
			tok, apiErr, err := postToken(ctx, appURL, map[string]string{"device_code": dc.DeviceCode})
			if err != nil {
				return fmt.Errorf("failed to check device approval status: %w", err)
			}
			switch apiErr {
			case "":
				return saveSessionAndReport(out, existing, appURL, apiURL, tok)
			case "authorization_pending":
				continue
			case "expired_token":
				return fmt.Errorf("device code expired before it was approved; please run 'scheduler0 login' again")
			case "access_denied":
				return fmt.Errorf("authorization was denied")
			default:
				return fmt.Errorf("token endpoint error: %s", apiErr)
			}
		}
	}
}

func buildAuthorizeURL(appURL, redirectURI, state, challenge string) string {
	u := strings.TrimRight(appURL, "/") + "/cli/authorize"
	q := url.Values{}
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return u + "?" + q.Encode()
}

// pkceChallenge returns base64url(sha256(verifier)) per RFC 7636 (S256).
func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// randomURLToken returns an unpadded base64url string with n bytes of entropy.
func randomURLToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// openBrowser opens the given URL in the user's default browser.
func openBrowser(target string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
		args = []string{target}
	case "windows":
		name = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", target}
	default: // linux, bsd, ...
		name = "xdg-open"
		args = []string{target}
	}
	return exec.Command(name, args...).Start()
}

func writeBrowserMessage(w http.ResponseWriter, success bool, message string) {
	title := "Signed in"
	if !success {
		title = "Sign-in problem"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Scheduler0 CLI</title>
<style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#0f172a;color:#e2e8f0;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}.card{background:#1e293b;border:1px solid #334155;border-radius:12px;padding:32px;max-width:420px;text-align:center}h1{font-size:20px;margin:0 0 8px}p{color:#94a3b8;line-height:1.5;margin:0}</style>
</head><body><div class="card"><h1>%s</h1><p>%s</p></div></body></html>`, title, message)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func cfgAppURL(c *config.Config) string {
	if c == nil {
		return ""
	}
	return c.AppURL
}

func cfgBaseURL(c *config.Config) string {
	if c == nil {
		return ""
	}
	return c.BaseURL
}
