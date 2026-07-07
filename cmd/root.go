package cmd

import (
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	baseURL   string
	accountID string
)

var (
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "scheduler0",
	Short: "Scheduler0 CLI - A command-line interface for Scheduler0",
	Long: `Scheduler0 CLI is a command-line tool for interacting with the Scheduler0 API.

Run 'scheduler0 login' to sign in before using other commands.`,
	Version: version,
}

// SetVersion sets the version for the CLI (used during build)
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Config is loaded per-command as needed
}

// GetClientConfig resolves the session credential and applies per-command flag
// overrides. It prefers SCHEDULER0_* environment variables (see
// config.LoadConfigFromEnv), for CI environments where an interactive
// `scheduler0 login` isn't possible, and otherwise falls back to the session
// saved by `scheduler0 login`. It errors when neither yields a valid session.
func GetClientConfig() (*config.Config, error) {
	cfg, err := resolveSessionConfig()
	if err != nil {
		return nil, err
	}

	// Allow single-command overrides of the endpoint/account.
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	if accountID != "" {
		cfg.AccountID = accountID
	}

	if !cfg.IsSessionValid() {
		if cfg.EnvSourced {
			return nil, fmt.Errorf("%s, %s, and %s must all be set", config.EnvAPIKey, config.EnvAPISecret, config.EnvAccountID)
		}
		return nil, fmt.Errorf("session expired or missing: run 'scheduler0 login'")
	}

	return cfg, nil
}

// resolveSessionConfig prefers a CI session from SCHEDULER0_* environment
// variables over the on-disk session, so that setting them always wins
// (matching how the --base-url/--account-id flags override both further up).
func resolveSessionConfig() (*config.Config, error) {
	if envCfg, ok := config.LoadConfigFromEnv(); ok {
		return envCfg, nil
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("not signed in: run 'scheduler0 login', or set %s/%s/%s for CI", config.EnvAPIKey, config.EnvAPISecret, config.EnvAccountID)
	}
	return cfg, nil
}

// actor returns the identity recorded for created_by/modified_by/deleted_by/
// archived_by fields: the Clerk user id captured at login, or SCHEDULER0_ACTOR
// for an environment-authenticated (CI) session.
func actor(cfg *config.Config) (string, error) {
	if cfg == nil || cfg.ClerkUserID == "" {
		if cfg != nil && cfg.EnvSourced {
			return "", fmt.Errorf("%s must be set to perform this action", config.EnvActor)
		}
		return "", fmt.Errorf("no signed-in user found: run 'scheduler0 login'")
	}
	return cfg.ClerkUserID, nil
}

// applyAccountIDFlag reads the local --account-id flag on cmd and patches cfg if
// the flag was explicitly provided. Subcommands that register their own
// --account-id flag should call this immediately after GetClientConfig so the
// per-command value takes precedence over the global persistent flag and config.
func applyAccountIDFlag(cmd *cobra.Command, cfg *config.Config) {
	if f := cmd.Flags().Lookup("account-id"); f != nil && f.Changed {
		cfg.AccountID = f.Value.String()
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Scheduler0 API base URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&accountID, "account-id", "", "Account ID (overrides the signed-in account for this command)")
}
