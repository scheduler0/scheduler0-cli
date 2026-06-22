package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	"github.com/spf13/cobra"
)

var asyncTasksCmd = &cobra.Command{
	Use:   "async-tasks",
	Short: "Manage async tasks",
	Long:  "Commands for managing async tasks",
}

var asyncTasksGetCmd = &cobra.Command{
	Use:   "get [request-id]",
	Short: "Get async task status",
	Long:  "Get the status of an async task by request ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runAsyncTasksGet,
}

func init() {
	rootCmd.AddCommand(asyncTasksCmd)
	asyncTasksCmd.AddCommand(asyncTasksGetCmd)

	asyncTasksGetCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
}

func runAsyncTasksGet(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	requestID := args[0]
	result, err := cl.GetAsyncTask(requestID)
	if err != nil {
		return fmt.Errorf("failed to get async task: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

