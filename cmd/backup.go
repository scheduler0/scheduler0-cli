package cmd

import (
	"encoding/json"
	"fmt"

	scheduler0_client "github.com/scheduler0/scheduler0-go-client"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Database backup and restore operations",
	Long:  "Manage database backups and restores for Scheduler0 cluster",
}

var backupStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start an automatic timestamped backup",
	Long:  "Initiate a database backup with automatic timestamped filename",
	RunE:  runBackupStart,
}

var restoreCmd = &cobra.Command{
	Use:   "restore [file-name]",
	Short: "Restore database from a backup file",
	Long:  "Initiate a database restore. Pass the backup file name (S3 object key when S3 is configured, or local path otherwise).",
	Args:  cobra.ExactArgs(1),
	RunE:  runRestore,
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupStartCmd)
	backupCmd.AddCommand(restoreCmd)
}

func runBackupStart(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := scheduler0_client.NewBasicAuthClient(cfg.BaseURL, "v1", cfg.Username, cfg.Password)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	result, err := cl.BackupDatabase()
	if err != nil {
		return fmt.Errorf("failed to start backup: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runRestore(cmd *cobra.Command, args []string) error {
	fileName := args[0]

	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := scheduler0_client.NewBasicAuthClient(cfg.BaseURL, "v1", cfg.Username, cfg.Password)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	result, err := cl.RestoreDatabase(fileName)
	if err != nil {
		return fmt.Errorf("failed to start restore: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
