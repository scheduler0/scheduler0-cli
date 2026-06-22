package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/scheduler0/scheduler0-cli/internal/localexecutor"
	scheduler0 "github.com/scheduler0/scheduler0-go-client"
	"github.com/spf13/cobra"
)

var localExecutorCmd = &cobra.Command{
	Use:   "local-executor",
	Short: "Manage and run the local executor",
	Long:  "Commands for registering and starting the local job executor on this machine.",
}

var localExecutorRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register this machine as a local executor",
	Long: `Register this machine as a local executor with scheduler0-private.

The registration saves the executor ID to ~/.scheduler0/config.json so that
subsequent 'start' calls work without any additional flags.

Re-running this command overwrites the previously registered executor.`,
	RunE: runLocalExecutorRegister,
}

var localExecutorStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the local executor service",
	Long: `Start the long-running local executor service.

The service uses the executor registered via 'local-executor register'. It polls
the scheduler0-private server every poll-interval for jobs assigned to this
executor, runs them locally as shell commands, and batches execution reports
back to the server. It continues executing jobs when temporarily offline and
batches the reports when reconnected.`,
	RunE: runLocalExecutorStart,
}

func init() {
	rootCmd.AddCommand(localExecutorCmd)
	localExecutorCmd.AddCommand(localExecutorRegisterCmd)
	localExecutorCmd.AddCommand(localExecutorStartCmd)

	localExecutorRegisterCmd.Flags().String("name", "", "Executor name (required)")
	localExecutorRegisterCmd.Flags().String("command", "", "Shell command to run for each job (required)")
	localExecutorRegisterCmd.Flags().String("working-dir", "", "Working directory for the command")
	localExecutorRegisterCmd.Flags().String("created-by", "", "User who is registering the executor")
	if err := localExecutorRegisterCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	if err := localExecutorRegisterCmd.MarkFlagRequired("command"); err != nil {
		panic(err)
	}

	localExecutorStartCmd.Flags().Duration("poll-interval", time.Minute, "How often to poll for new jobs (e.g. 1m, 30s)")
}

func runLocalExecutorRegister(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	command, _ := cmd.Flags().GetString("command")
	workingDir, _ := cmd.Flags().GetString("working-dir")
	createdBy, _ := cmd.Flags().GetString("created-by")

	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	c, err := client.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	resp, err := c.RegisterLocalExecutor(&scheduler0.LocalExecutorRegisterRequest{
		Name:       name,
		Command:    command,
		WorkingDir: workingDir,
		CreatedBy:  createdBy,
	})
	if err != nil {
		return fmt.Errorf("failed to register local executor: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("server returned failure registering executor")
	}

	idStr := strconv.FormatInt(resp.Data.ID, 10)

	entry := config.LocalExecutorEntry{
		ID:         idStr,
		Name:       name,
		Command:    command,
		WorkingDir: workingDir,
	}

	if err := config.SetLocalExecutor(entry); err != nil {
		return fmt.Errorf("executor registered (id=%s) but failed to save to config: %w", idStr, err)
	}

	fmt.Printf("Local executor registered successfully.\n  ID:      %s\n  Name:    %s\n  Command: %s\n", idStr, name, command)
	fmt.Println("Run 'scheduler0 local-executor start' to start the service.")
	return nil
}

func runLocalExecutorStart(cmd *cobra.Command, _ []string) error {
	pollInterval, _ := cmd.Flags().GetDuration("poll-interval")
	if pollInterval <= 0 {
		pollInterval = time.Minute
	}

	entry, err := config.GetLocalExecutor()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if entry == nil {
		return fmt.Errorf("no local executor registered on this machine.\nRun 'scheduler0 local-executor register' first")
	}

	executorID, err := strconv.ParseInt(entry.ID, 10, 64)
	if err != nil || executorID <= 0 {
		return fmt.Errorf("invalid executor ID %q in config; try re-registering", entry.ID)
	}

	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	c, clientErr := client.NewClient(cfg)
	if clientErr != nil {
		return fmt.Errorf("failed to create client: %w", clientErr)
	}

	logger := log.New(os.Stderr, "[local-executor] ", log.LstdFlags)

	exec, newErr := localexecutor.New(c, executorID, entry.Command, entry.WorkingDir, pollInterval, logger)
	if newErr != nil {
		return fmt.Errorf("failed to initialise local executor: %w", newErr)
	}
	defer exec.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT / SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Printf("received signal %s, shutting down...", sig)
		cancel()
	}()

	exec.Run(ctx)
	return nil
}
