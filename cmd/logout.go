package cmd

import (
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out and remove the stored session credential",
	Long: `Remove the locally stored Scheduler0 session credential.

The remote credential will also expire on its own; logout simply clears it from
this machine. Endpoints and any local executor registration are preserved.`,
	RunE: runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	cfg, err := config.LoadConfig()
	if err != nil {
		// No config at all means there's nothing to clear.
		_, _ = fmt.Fprintln(out, "You are not signed in.")
		return nil
	}

	if cfg.APIKey == "" && cfg.APISecret == "" {
		_, _ = fmt.Fprintln(out, "You are not signed in.")
		return nil
	}

	cfg.ClearSession()
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to clear stored credentials: %w", err)
	}

	_, _ = fmt.Fprintln(out, "✓ Signed out.")
	return nil
}
