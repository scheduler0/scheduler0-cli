package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Create and schedule jobs from an AI prompt",
	Long: `Turn a natural language prompt into scheduled jobs. The server runs the prompt
pipeline (intent guardrail + generation), resolves or creates a project, picks the
executor whose description/tags best match the prompt (or uses a pinned/only executor),
and creates the jobs synchronously. This endpoint requires credits (1 credit per prompt).`,
	RunE: runSchedule,
}

func init() {
	rootCmd.AddCommand(scheduleCmd)

	scheduleCmd.Flags().String("prompt", "", "Natural language prompt describing the jobs to create (required)")
	scheduleCmd.Flags().StringSlice("purposes", []string{}, "Array of business purposes")
	scheduleCmd.Flags().StringSlice("events", []string{}, "Array of event types")
	scheduleCmd.Flags().StringSlice("recipients", []string{}, "Array of recipient identifiers")
	scheduleCmd.Flags().StringSlice("channels", []string{}, "Array of delivery channels")
	scheduleCmd.Flags().String("timezone", "", "Optional IANA timezone for scheduling calculations (e.g., UTC, America/New_York). Defaults to UTC when omitted.")
	scheduleCmd.Flags().String("locale", "", "Optional locale for interpreting the prompt")
	scheduleCmd.Flags().Int64("project-id", 0, "Reuse an existing project by ID (takes precedence over a generated project)")
	scheduleCmd.Flags().Int64("executor-id", 0, "Pin a specific executor and skip LLM matching")
	_ = scheduleCmd.MarkFlagRequired("prompt")
}

func runSchedule(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	prompt, _ := cmd.Flags().GetString("prompt")
	purposes, _ := cmd.Flags().GetStringSlice("purposes")
	events, _ := cmd.Flags().GetStringSlice("events")
	recipients, _ := cmd.Flags().GetStringSlice("recipients")
	channels, _ := cmd.Flags().GetStringSlice("channels")
	timezone, _ := cmd.Flags().GetString("timezone")
	locale, _ := cmd.Flags().GetString("locale")

	if len(strings.TrimSpace(prompt)) == 0 {
		return fmt.Errorf("prompt cannot be empty")
	}

	// createdBy is required by the API; derive it from the signed-in session
	// (or SCHEDULER0_ACTOR for CI) like the other create commands.
	createdBy, err := actor(cfg)
	if err != nil {
		return err
	}

	request := &scheduler0_client.SchedulePromptRequest{
		Prompt:    strings.TrimSpace(prompt),
		Timezone:  timezone,
		Locale:    locale,
		CreatedBy: createdBy,
	}
	if len(purposes) > 0 {
		request.Purposes = purposes
	}
	if len(events) > 0 {
		request.Events = events
	}
	if len(recipients) > 0 {
		request.Recipients = recipients
	}
	if len(channels) > 0 {
		request.Channels = channels
	}
	if cmd.Flags().Changed("project-id") {
		projectID, _ := cmd.Flags().GetInt64("project-id")
		request.ProjectID = &projectID
	}
	if cmd.Flags().Changed("executor-id") {
		executorID, _ := cmd.Flags().GetInt64("executor-id")
		request.ExecutorID = &executorID
	}

	result, err := cl.ScheduleFromPrompt(request)
	if err != nil {
		// When the intent guardrail rejects the prompt, surface the structured
		// classification so the user can see why and how to rephrase.
		var skipped *scheduler0_client.PromptSkippedError
		if errors.As(err, &skipped) {
			if skipped.Classification != nil {
				fmt.Printf("Prompt %s: %s\n", skipped.Classification.Decision, skipped.Classification.Reason)
				out, _ := json.MarshalIndent(skipped.Classification, "", "  ")
				fmt.Println(string(out))
				return fmt.Errorf("prompt not accepted by intent guardrail (decision=%s)", skipped.Classification.Decision)
			}
			return fmt.Errorf("prompt not accepted by intent guardrail: %s", skipped.Message)
		}
		return fmt.Errorf("failed to schedule from prompt: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
